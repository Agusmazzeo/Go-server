package controllers

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"server/src/clients/bcra"
	"server/src/clients/esco"
	"server/src/schemas"
	"server/src/services"
	"server/src/utils"
	"server/src/utils/render"
	"slices"
	"sort"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/go-echarts/v2/types"
	"github.com/go-gota/gota/dataframe"
	"github.com/go-gota/gota/series"
	"github.com/xuri/excelize/v2"
)

type ReportsControllerI interface {
	GetReport(ctx context.Context, clientIDs []string, variablesWithValuations map[string]*schemas.VariableWithValuationResponse, startDate, endDate time.Time, interval time.Duration) (*schemas.AccountsReports, error)

	ParseAccountsReportToDataFrames(ctx context.Context, accountsReport *schemas.AccountsReports, startDate, endDate time.Time, interval time.Duration) (*schemas.ReportDataframes, error)
	ParseAccountsReportToXLSX(ctx context.Context, dataframesAndCharts *schemas.ReportDataframes) (*excelize.File, error)
	ParseAccountsReportToPDF(ctx context.Context, dataframesAndCharts *schemas.ReportDataframes) ([]byte, error)
	ParseAccountsReportToDataFrame(ctx context.Context, accountsReport *schemas.AccountsReports, startDate, endDate time.Time, interval time.Duration) (*dataframe.DataFrame, error)
	ParseAccountsReturnToDataFrame(ctx context.Context, accountsReport *schemas.AccountsReports, startDate, endDate time.Time, interval time.Duration) (*dataframe.DataFrame, error)
	ParseAccountsCategoryToDataFrame(ctx context.Context, accountsReport *schemas.AccountsReports, startDate, endDate time.Time, interval time.Duration) (*dataframe.DataFrame, error)
}

type ReportsController struct {
	ESCOClient     esco.ESCOServiceClientI
	BCRAClient     bcra.BCRAServiceClientI
	ReportService  services.ReportServiceI
	AccountService services.AccountServiceI
}

func NewReportsController(
	escoClient esco.ESCOServiceClientI,
	bcraClient bcra.BCRAServiceClientI,
	reportService services.ReportServiceI,
	accountService services.AccountServiceI,
) *ReportsController {
	return &ReportsController{
		ESCOClient:     escoClient,
		BCRAClient:     bcraClient,
		ReportService:  reportService,
		AccountService: accountService,
	}
}

func (rc *ReportsController) GetReport(
	ctx context.Context,
	clientIDs []string,
	variablesWithValuations map[string]*schemas.VariableWithValuationResponse,
	startDate, endDate time.Time,
	interval time.Duration,
) (*schemas.AccountsReports, error) {
	// Build account state from client ID using existing AccountService
	accountStateByCategory, err := rc.AccountService.GetMultiAccountStateByCategory(ctx, clientIDs, startDate, endDate, interval)
	if err != nil {
		return nil, err
	}

	// Generate report from account state
	accountReports, err := rc.ReportService.GenerateReport(ctx, accountStateByCategory, startDate, endDate, interval)
	if err != nil {
		return nil, err
	}
	accountReports.ReferenceVariables = &variablesWithValuations
	return accountReports, nil
}

func (rc *ReportsController) ParseAccountsReportToDataFrames(
	ctx context.Context,
	accountsReport *schemas.AccountsReports,
	startDate, endDate time.Time,
	interval time.Duration,
) (*schemas.ReportDataframes, error) {
	var reportDf *dataframe.DataFrame
	var reportPercentageDf *dataframe.DataFrame
	var returnsDf *dataframe.DataFrame
	var referenceVariablesDf *dataframe.DataFrame
	var categoryDf *dataframe.DataFrame
	var categoryPercentageDf *dataframe.DataFrame

	var wg sync.WaitGroup
	wg.Add(4)
	var errChan = make(chan error, 4)
	go func() {
		defer wg.Done()
		var err error
		reportDf, err = rc.ParseAccountsReportToDataFrame(ctx, accountsReport, startDate, endDate, interval)
		if err != nil {
			errChan <- err
			return
		}
		reportPercentageDf = divideByTotal(reportDf)
	}()

	go func() {
		defer wg.Done()
		var err error
		categoryDf, err = rc.ParseAccountsCategoryToDataFrame(ctx, accountsReport, startDate, endDate, interval)
		if err != nil {
			errChan <- err
			return
		}
		categoryPercentageDf = divideByTotal(categoryDf)
	}()

	go func() {
		defer wg.Done()
		var err error
		returnsDf, err = rc.ParseAccountsReturnToDataFrame(ctx, accountsReport, startDate, endDate, interval)
		if err != nil {
			errChan <- err
			return
		}
	}()

	go func() {
		defer wg.Done()
		var err error
		referenceVariablesDf, err = rc.ParseReferenceVariablesToDataFrame(ctx, accountsReport, startDate, endDate, interval)
		if err != nil {
			errChan <- err
			return
		}
	}()

	wg.Wait()
	close(errChan)
	if err := <-errChan; err != nil {
		return nil, err
	}
	return &schemas.ReportDataframes{
		ReportDF:             reportDf,
		ReportPercentageDf:   reportPercentageDf,
		ReturnDF:             returnsDf,
		ReferenceVariablesDF: referenceVariablesDf,
		CategoryDF:           categoryDf,
		CategoryPercentageDF: categoryPercentageDf,
	}, nil
}

func (rc *ReportsController) ParseAccountsReportToXLSX(
	ctx context.Context,
	dataframesAndCharts *schemas.ReportDataframes,
) (*excelize.File, error) {
	file, err := convertReportDataframeToExcel(nil, dataframesAndCharts.ReportDF, "Tenencia", false, true, true)
	if err != nil {
		return nil, err
	}
	file, err = convertReportDataframeToExcel(file, dataframesAndCharts.ReportPercentageDf, "Tenencia_Porcentaje", true, true, false)
	if err != nil {
		return nil, err
	}
	file, err = convertReportDataframeToExcel(file, dataframesAndCharts.ReturnDF, "Retorno", false, true, false)
	if err != nil {
		return nil, err
	}
	file, err = convertReportDataframeToExcel(file, dataframesAndCharts.ReferenceVariablesDF, "Referencias", false, true, false)
	if err != nil {
		return nil, err
	}
	err = applyStylesToAllSheets(file)
	if err != nil {
		return nil, err
	}
	return file, nil
}

func (rc *ReportsController) ParseAccountsReportToDataFrame(ctx context.Context, accountsReport *schemas.AccountsReports, startDate, endDate time.Time, interval time.Duration) (*dataframe.DataFrame, error) {
	dates, err := utils.GenerateDates(startDate, endDate, interval)
	if err != nil {
		return nil, err
	}
	dateStrs := make([]string, len(dates))
	for i, date := range dates {
		dateStrs[i] = date.Format("2006-01-02")
	}

	// Initialize an empty DataFrame with the DateRequested as the index (as the first column)
	df := dataframe.New(
		series.New(dateStrs, series.String, "DateRequested"),
	)

	// Iterate through the assets and add each as a new column
	for _, assets := range *accountsReport.AssetsByCategory {
		for _, asset := range assets {
			assetValues := make([]string, len(dates))

			// Initialize all rows with empty values for this asset
			for i := range assetValues {
				assetValues[i] = "0.0" // Default value if no match found
			}

			// Iterate through holdings and match the dates to fill the corresponding values
			for _, holding := range asset.Holdings {
				if holding.DateRequested != nil {
					dateStr := holding.DateRequested.Format("2006-01-02")
					// Find the index in the dates array that matches this holding's date
					for i, date := range dateStrs {
						if date == dateStr {
							if holding.Value >= 1.0 || holding.Value <= -1.0 {
								assetValues[i] = fmt.Sprintf("%.2f", holding.Value)
							} else {
								assetValues[i] = "0.0"
							}
							break
						}
					}
				}
			}

			// Add the new series (column) for this asset to the DataFrame
			updatedDf, err := updateDataFrame(df, fmt.Sprintf("%s-%s", asset.Category, asset.ID), assetValues)
			if err != nil {
				return nil, err
			}
			df = *updatedDf
		}
	}

	totalValues := make([]string, len(dates))
	for _, totalHolding := range accountsReport.TotalHoldingsByDate {
		dateStr := totalHolding.DateRequested.Format("2006-01-02")
		// Find the index in the dates array that matches this holding's date
		for i, date := range dateStrs {
			if date == dateStr {
				if totalHolding.Value >= 1.0 {
					totalValues[i] = fmt.Sprintf("%.2f", totalHolding.Value)
				} else {
					totalValues[i] = "0.0"
				}
				break
			}
		}
	}
	for i, v := range totalValues {
		if v == "" {
			totalValues[i] = "0.0"
		}
	}
	// Add the new series (column) for this asset to the DataFrame
	updatedDf, err := updateDataFrame(df, "TOTAL", totalValues)
	if err != nil {
		return nil, err
	}
	df = *updatedDf

	return sortDataFrameColumns(&df), nil
}

func (rc *ReportsController) ParseAccountsReturnToDataFrame(ctx context.Context, accountsReport *schemas.AccountsReports, startDate, endDate time.Time, interval time.Duration) (*dataframe.DataFrame, error) {
	dates, err := utils.GenerateDates(startDate, endDate, interval)
	if err != nil {
		return nil, err
	}
	dateStrs := make([]string, len(dates))
	for i, date := range dates {
		dateStrs[i] = date.Format("2006-01-02")
	}

	// Initialize an empty DataFrame with the DateRequested as the index (as the first column)
	df := dataframe.New(
		series.New(dateStrs, series.String, "DateRequested"),
	)

	// Iterate through the assets and add each as a new column
	for _, assets := range *accountsReport.AssetsReturnByCategory {
		for _, asset := range assets {
			assetValues := make([]string, len(dates))

			// Initialize all rows with empty values for this asset
			for i := range assetValues {
				assetValues[i] = "0.0" // Default value if no match found
			}

			// Iterate through assets return and match the dates to fill the corresponding values
			for _, returnsByDate := range asset.ReturnsByDateRange {
				dateStr := returnsByDate.EndDate.Format("2006-01-02")
				// Find the index in the dates array that matches this holding's date
				for i, date := range dateStrs {
					if date == dateStr {
						assetValues[i] = fmt.Sprintf("%.2f", returnsByDate.ReturnPercentage)
						break
					}
				}
			}

			// Add the new series (column) for this asset to the DataFrame
			updatedDf, err := updateDataFrame(df, fmt.Sprintf("%s-%s", asset.Category, asset.ID), assetValues)
			if err != nil {
				return nil, err
			}
			df = *updatedDf
		}
	}
	totalValues := make([]string, len(dates))
	for _, totalReturn := range accountsReport.TotalReturns {
		dateStr := totalReturn.EndDate.Format("2006-01-02")
		// Find the index in the dates array that matches this holding's date
		for i, date := range dateStrs {
			if date == dateStr {
				totalValues[i] = fmt.Sprintf("%.2f", totalReturn.ReturnPercentage)
				break
			}
		}
	}
	for i, v := range totalValues {
		if v == "" {
			totalValues[i] = "0.0"
		}
	}
	// Add the new series (column) for this asset to the DataFrame
	updatedDf, err := updateDataFrame(df, "TOTAL", totalValues)
	if err != nil {
		return nil, err
	}
	df = *updatedDf
	return sortDataFrameColumns(&df), nil
}

func (rc *ReportsController) ParseReferenceVariablesToDataFrame(ctx context.Context, accountsReport *schemas.AccountsReports, startDate, endDate time.Time, interval time.Duration) (*dataframe.DataFrame, error) {
	dates, err := utils.GenerateDates(startDate, endDate, interval)
	if err != nil {
		return nil, err
	}
	dateStrs := make([]string, len(dates))
	for i, date := range dates {
		dateStrs[i] = date.Format("2006-01-02")
	}

	// Initialize an empty DataFrame with the DateRequested as the index (as the first column)
	df := dataframe.New(
		series.New(dateStrs, series.String, "DateRequested"),
	)

	// Iterate through the assets and add each as a new column
	for name, referenceVariable := range *(*accountsReport).ReferenceVariables {
		for _, valuation := range referenceVariable.Valuations {
			valuationValues := make([]string, len(dates))

			// Initialize all rows with empty values for this asset
			for i := range valuationValues {
				valuationValues[i] = "0.0" // Default value if no match found
			}

			dateStr := valuation.Date
			// Find the index in the dates array that matches this holding's date
			for i, date := range dateStrs {
				if date == dateStr {
					valuationValues[i] = fmt.Sprintf("%.2f", valuation.Value)
					break
				}
			}

			// Add the new series (column) for this asset to the DataFrame
			updatedDf, err := updateDataFrame(df, name, valuationValues)
			if err != nil {
				return nil, err
			}
			df = *updatedDf
		}
	}

	return sortDataFrameColumns(&df), nil
}

// ParseAccountsCategoryToDataFrame converts account reports into a DataFrame
// for the specified time range and interval.
func (rc *ReportsController) ParseAccountsCategoryToDataFrame(_ context.Context, accountsReport *schemas.AccountsReports, startDate, endDate time.Time, interval time.Duration) (*dataframe.DataFrame, error) {
	dates, err := utils.GenerateDates(startDate, endDate, interval)
	if err != nil {
		return nil, err
	}
	dateStrs := make([]string, len(dates))
	for i, date := range dates {
		dateStrs[i] = date.Format("2006-01-02")
	}

	// Initialize an empty DataFrame with the DateRequested as the index (as the first column)
	df := dataframe.New(
		series.New(dateStrs, series.String, "DateRequested"),
	)

	// Iterate through the assets and add each as a new column
	for category, asset := range *accountsReport.CategoryAssets {
		assetValues := make([]string, len(dates))

		// Initialize all rows with empty values for this asset
		for i := range assetValues {
			assetValues[i] = "0.0" // Default value if no match found
		}

		// Iterate through holdings and match the dates to fill the corresponding values
		for _, holding := range asset.Holdings {
			if holding.DateRequested != nil {
				dateStr := holding.DateRequested.Format("2006-01-02")
				// Find the index in the dates array that matches this holding's date
				for i, date := range dateStrs {
					if date == dateStr {
						if holding.Value >= 1.0 || holding.Value <= -1.0 {
							assetValues[i] = fmt.Sprintf("%.2f", holding.Value)
						} else {
							assetValues[i] = "0.0"
						}
						break
					}
				}
			}
		}

		// Add the new series (column) for this asset to the DataFrame
		var updatedDf *dataframe.DataFrame
		updatedDf, err = updateDataFrame(df, category, assetValues)
		if err != nil {
			return nil, err
		}
		df = *updatedDf

	}

	totalValues := make([]string, len(dates))
	for _, totalHolding := range accountsReport.TotalHoldingsByDate {
		dateStr := totalHolding.DateRequested.Format("2006-01-02")
		// Find the index in the dates array that matches this holding's date
		for i, date := range dateStrs {
			if date == dateStr {
				if totalHolding.Value >= 1.0 {
					totalValues[i] = fmt.Sprintf("%.2f", totalHolding.Value)
				} else {
					totalValues[i] = "0.0"
				}
				break
			}
		}
	}
	for i, v := range totalValues {
		if v == "" {
			totalValues[i] = "0.0"
		}
	}
	// Add the new series (column) for this asset to the DataFrame
	updatedDf, err := updateDataFrame(df, "TOTAL", totalValues)
	if err != nil {
		return nil, err
	}
	df = *updatedDf

	return sortDataFrameColumns(&df), nil
}

type ReportConfig struct {
	name             string
	df               *dataframe.DataFrame
	graphType        string
	columnsToInclude []string
	columnsToExclude []string
	isPercentage     bool
	includeTable     bool
}

// ParseAccountsReportToPDF generates bar graphs and pie charts, embeds them in HTML, and creates a PDF.
func (rc *ReportsController) ParseAccountsReportToPDF(ctx context.Context, dataframesAndCharts *schemas.ReportDataframes) ([]byte, error) {
	var htmlContents []string
	returnWithReferencesDF := utils.UnionDataFramesByIndex(*dataframesAndCharts.ReturnDF, *dataframesAndCharts.ReferenceVariablesDF, "DateRequested")
	// Generate bar graphs for each dataframe
	for _, report := range []*ReportConfig{
		{name: "TENENCIA POR CATEGORIAS", df: dataframesAndCharts.CategoryDF, columnsToExclude: []string{"TOTAL"}, graphType: "line", includeTable: true},
		{name: "TENENCIA POR CATEGORIAS PORCENTAJE", df: dataframesAndCharts.CategoryPercentageDF, columnsToExclude: []string{"TOTAL"}, graphType: "bar", isPercentage: true},
		{name: "TENENCIA", df: dataframesAndCharts.ReportPercentageDf, columnsToExclude: []string{"TOTAL"}, graphType: "bar", isPercentage: true},
		{name: "TENENCIA TOTAL", df: dataframesAndCharts.ReportDF, columnsToInclude: []string{"TOTAL"}, graphType: "line", includeTable: true},
		{name: "TENENCIA PORCENTAJE", df: dataframesAndCharts.ReportPercentageDf, graphType: "pie", isPercentage: true},
		{name: "RETORNO", df: &returnWithReferencesDF, columnsToInclude: []string{"Inflacion Mensual", "USD A3500 Variacion", "TOTAL"}, graphType: "line", isPercentage: true, includeTable: true},
	} {
		if report.df == nil {
			continue
		}
		var htmlContent string
		var err error
		// htmlContent, err = render.GetSeparatorPageHTML(report.name)
		// if err != nil {
		// 	return nil, fmt.Errorf("failed to generate separator for %s: %w", report.name, err)
		// }
		// htmlContents = append(htmlContents, htmlContent)

		// Generate bar graph and embed in HTML
		if report.graphType == "bar" {
			htmlContent, err = rc.generateStackBarGraphHTML(report.name, report)
		} else if report.graphType == "pie" {
			htmlContent, err = rc.generatePieChartHTML(report.name, report)
		} else if report.graphType == "line" {
			htmlContent, err = rc.generateLineGraphHTML(report.name, report)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to generate graph for %s: %w", report.name, err)
		}
		htmlContents = append(htmlContents, htmlContent)

		if !report.includeTable {
			continue
		}

		htmlContent, err = render.GetTableHTML(report.df)
		if err != nil {
			return nil, fmt.Errorf("failed to generate table for %s: %w", report.name, err)
		}
		htmlContents = append(htmlContents, htmlContent)
	}

	// Convert all HTML content into a PDF
	pdfBuffer, err := render.GeneratePDF(htmlContents)
	if err != nil {
		return nil, fmt.Errorf("failed to generate PDF: %w", err)
	}

	return pdfBuffer.Bytes(), nil
}

func (rc *ReportsController) generateLineGraphHTML(name string, report *ReportConfig) (string, error) {
	df := report.df
	// Create a bar chart
	line := charts.NewLine()
	line.SetGlobalOptions(
		charts.WithAnimation(false),
		charts.WithYAxisOpts(opts.YAxis{
			SplitLine: &opts.SplitLine{
				Show: opts.Bool(true),
			},
		}),
		charts.WithInitializationOpts(opts.Initialization{
			Width:  "1600px",
			Height: "900px",
		}),
	)

	// Extract labels (dates) and data
	labels := df.Col("DateRequested").Records()
	line.SetXAxis(labels)

	for _, asset := range df.Names()[1:] {
		if (len(report.columnsToInclude) != 0 && !slices.Contains(report.columnsToInclude, asset)) || slices.Contains(report.columnsToExclude, asset) {
			continue
		}
		data := make([]opts.LineData, 0)
		for _, value := range df.Col(asset).Records() {
			v, _ := strconv.ParseFloat(value, 32)
			var label string
			if report.isPercentage {
				label = render.FormatPercentageValue(value)
			} else {
				label = render.FormatMonetaryValue(value)
			}
			data = append(data, opts.LineData{Name: label, Value: int(v)})
		}
		line.AddSeries(asset, data,
			charts.WithLabelOpts(opts.Label{
				Show:      opts.Bool(true),
				Formatter: "{b}",
			}),
			charts.WithAreaStyleOpts(opts.AreaStyle{
				Opacity: 0.2,
			}),
			charts.WithLineChartOpts(opts.LineChart{
				Smooth: opts.Bool(true),
			}),
		)
	}
	baseDir, _ := os.Getwd()
	// Load HTML template
	tmpl, err := template.ParseFiles(fmt.Sprintf("%s/templates/bar_graph.html", baseDir))
	if err != nil {
		return "", fmt.Errorf("failed to load bar graph template: %w", err)
	}

	// Render HTML embedding the chart image
	var htmlBuffer bytes.Buffer
	err = tmpl.Execute(&htmlBuffer, map[string]interface{}{
		"Title": name,
		"Graph": strings.ReplaceAll(string(line.RenderContent()), "let ", "var "),
	})
	if err != nil {
		return "", fmt.Errorf("failed to render bar graph HTML: %w", err)
	}

	return htmlBuffer.String(), nil
}

func (rc *ReportsController) generateStackBarGraphHTML(name string, report *ReportConfig) (string, error) {
	df := report.df
	// Create a bar chart
	bar := charts.NewBar()
	bar.SetGlobalOptions(
		charts.WithAnimation(false),
		charts.WithInitializationOpts(opts.Initialization{
			Width:  "1600px",
			Height: "900px",
		}),
	)

	// Extract labels (dates) and data
	labels := df.Col("DateRequested").Records()
	bar.SetXAxis(labels)

	for _, asset := range df.Names()[1:] {
		if (len(report.columnsToInclude) != 0 && !slices.Contains(report.columnsToInclude, asset)) || slices.Contains(report.columnsToExclude, asset) {
			continue
		}
		data := make([]opts.BarData, 0)
		for _, value := range df.Col(asset).Records() {
			v, _ := strconv.ParseFloat(value, 32)
			var label string
			if report.isPercentage {
				value = fmt.Sprintf("%f", 100*v)
				label = render.FormatPercentageValue(value)
			} else {
				label = render.FormatMonetaryValue(value)
			}
			data = append(data, opts.BarData{Name: label, Value: v})
		}
		bar.AddSeries(asset, data,
			charts.WithLabelOpts(opts.Label{
				Show:      opts.Bool(true),
				Formatter: "{b}",
			}),
			charts.WithAreaStyleOpts(opts.AreaStyle{
				Opacity: 0.2,
			}),
			charts.WithLineChartOpts(opts.LineChart{
				Smooth: opts.Bool(true),
			}),
		)
	}
	bar.SetSeriesOptions(charts.WithBarChartOpts(opts.BarChart{
		Stack: "stackA",
	}))
	baseDir, _ := os.Getwd()
	// Load HTML template
	tmpl, err := template.ParseFiles(fmt.Sprintf("%s/templates/bar_graph.html", baseDir))
	if err != nil {
		return "", fmt.Errorf("failed to load bar graph template: %w", err)
	}

	// Render HTML embedding the chart image
	var htmlBuffer bytes.Buffer
	err = tmpl.Execute(&htmlBuffer, map[string]interface{}{
		"Title": name,
		"Graph": strings.ReplaceAll(string(bar.RenderContent()), "let ", "var "),
	})
	if err != nil {
		return "", fmt.Errorf("failed to render bar graph HTML: %w", err)
	}

	return htmlBuffer.String(), nil
}

func (rc *ReportsController) generatePieChartHTML(name string, report *ReportConfig) (string, error) {
	df := report.df

	// Create the pie chart
	pie := charts.NewPie()
	show := true
	pie.SetGlobalOptions(
		charts.WithAnimation(false),
		charts.WithLegendOpts(opts.Legend{Show: types.Bool(&show), Left: "center"}),
		charts.WithTooltipOpts(opts.Tooltip{Show: types.Bool(&show)}),
		charts.WithInitializationOpts(opts.Initialization{
			Width:  "1600px",
			Height: "900px",
		}),
	)

	// Extract the last row for the pie chart
	lastRowIndex := df.Nrow() - 1
	items := []opts.PieData{}
	for colIndex, colName := range df.Names() {
		if colName == "DateRequested" || colName == "TOTAL" {
			// Skip Date Column
			continue
		}
		value := df.Elem(lastRowIndex, colIndex).String()

		items = append(items, opts.PieData{Name: colName, Value: value})
	}

	var formatter string
	if report.isPercentage {
		formatter = "{b}: {d} %"
	} else {
		formatter = "{b}: US$ {c}"
	}

	pie.AddSeries("Data", items).SetSeriesOptions(
		charts.WithLabelOpts(opts.Label{
			Show:      opts.Bool(true),
			Formatter: formatter,
		}),
		charts.WithPieChartOpts(opts.PieChart{
			Radius: []string{"40%", "75%"},
		}),
	)

	baseDir, _ := os.Getwd()
	// Load HTML template
	tmpl, err := template.ParseFiles(fmt.Sprintf("%s/templates/pie_graph.html", baseDir))
	if err != nil {
		return "", fmt.Errorf("failed to load pie chart template: %w", err)
	}

	var htmlBuffer bytes.Buffer
	err = tmpl.Execute(&htmlBuffer, map[string]interface{}{
		"Title": name,
		"Graph": strings.ReplaceAll(string(pie.RenderContent()), "let ", "var "),
	})
	if err != nil {
		return "", fmt.Errorf("failed to render pie chart HTML for %s: %w", name, err)
	}

	return htmlBuffer.String(), nil
}

// CalculateAssetReturn calculates the return for a single asset by taking holdings in pairs and applying transactions within the date ranges.

func convertReportDataframeToExcel(
	f *excelize.File,
	reportDf *dataframe.DataFrame,
	sheetName string,
	percentageData bool,
	includeBarGraph bool,
	includePieGraph bool,
) (*excelize.File, error) {
	// Create a new Excel file
	var err error
	var index int

	if f == nil {
		f = excelize.NewFile()
		err := f.SetSheetName("Sheet1", sheetName)
		if err != nil {
			return nil, err
		}
	} else {
		index, err = f.NewSheet(sheetName)
		if err != nil {
			return nil, err
		}
		defer f.SetActiveSheet(index)
	}
	// Define variables for column and row indices
	startRow := 1
	categoryRow := startRow
	idRow := startRow + 1

	// Extract column names from DataFrame
	cols := reportDf.Names()

	// Variables to track categories and column ranges for merging
	categoryStartCol := make(map[string]int)
	categoryEndCol := make(map[string]int)

	// Set the DateRequested in the first column (A) for both the first and second rows
	err = f.SetCellValue(sheetName, "A2", "Fecha")
	if err != nil {
		return nil, err
	}

	// Iterate over the columns in the DataFrame, starting from the second one (ignoring DateRequested)
	columnIndex := 2               // Excel columns start from B (index 2) for data columns
	for _, col := range cols[1:] { // Skip the first column "DateRequested"

		var category, id string

		if col == "TOTAL" {
			category = "TOTAL"
			id = "-"
		} else {
			parts := strings.Split(col, "-")
			category = parts[0]
			id = parts[1]
		}

		// Set the ID in the second row (e.g., ID1, ID2, etc.)
		cell := fmt.Sprintf("%s%d", toAlphaString(columnIndex), idRow)
		err = f.SetCellValue(sheetName, cell, id)
		if err != nil {
			return nil, err
		}

		// Track the start and end columns for merging categories
		if _, exists := categoryStartCol[category]; !exists {
			categoryStartCol[category] = columnIndex
		}
		categoryEndCol[category] = columnIndex

		columnIndex++
	}

	// Merge cells for each category and set the category name in the first row
	for category, startCol := range categoryStartCol {
		endCol := categoryEndCol[category]
		startCell := fmt.Sprintf("%s%d", toAlphaString(startCol), categoryRow)
		endCell := fmt.Sprintf("%s%d", toAlphaString(endCol), categoryRow)
		err = f.MergeCell(sheetName, startCell, endCell)
		if err != nil {
			return nil, err
		}
		err = f.SetCellValue(sheetName, startCell, category)
		if err != nil {
			return nil, err
		}
	}
	numFmt := 8
	if percentageData {
		numFmt = 10
	}
	// Format cells as currency
	cellStyle, err := f.NewStyle(&excelize.Style{
		NumFmt: numFmt,
	})
	if err != nil {
		return nil, err
	}
	// Now fill in the data for the rest of the rows starting from the third row
	for rowIndex, row := range reportDf.Records()[1:] { // Skip the first row (headers)
		for colIndex, cellValue := range row {
			cell := fmt.Sprintf("%s%d", toAlphaString(colIndex+1), rowIndex+3) // colIndex+1 to skip DateRequested
			numCellValue, err := strconv.ParseFloat(cellValue, 64)
			if err != nil {
				err = f.SetCellValue(sheetName, cell, cellValue)
				if err != nil {
					return nil, err
				}
			} else {
				err = f.SetCellValue(sheetName, cell, numCellValue)
				if err != nil {
					return nil, err
				}
			}

			if err = f.SetCellStyle(sheetName, cell, cell, cellStyle); err != nil {
				return nil, err
			}
		}
	}
	if includeBarGraph {
		if err := addBarGraphFromSheet(f, sheetName); err != nil {
			return nil, err
		}
	}

	if includePieGraph {
		if err := addPieChartFromLastRow(f, sheetName); err != nil {
			return nil, err
		}
	}

	return f, nil
}

// toAlphaString converts a column index to an Excel column string (e.g., 1 -> A, 2 -> B, 28 -> AB)
func toAlphaString(column int) string {
	result := ""
	for column > 0 {
		column-- // Decrement to handle 1-based indexing for Excel columns
		result = string(rune('A'+(column%26))) + result
		column /= 26
	}
	return result
}

func sortDataFrameColumns(df *dataframe.DataFrame) *dataframe.DataFrame {
	// Get the column names
	cols := df.Names()

	// Separate DateRequested from the other columns
	var otherCols []string
	for _, col := range cols {
		if col != "DateRequested" {
			otherCols = append(otherCols, col)
		}
	}

	// Sort the remaining columns
	sort.Strings(otherCols)

	// Reassemble the column list, putting DateRequested first
	sortedCols := append([]string{"DateRequested"}, otherCols...)

	// Rearrange the DataFrame by the sorted columns
	sortedDf := df.Select(sortedCols)

	// Return the sorted DataFrame
	return &sortedDf
}

// updateDataFrame receives the attributes (columnName and values) and the DataFrame,
// and returns the updated DataFrame. It assumes that the values are floats.
func updateDataFrame(df dataframe.DataFrame, columnName string, newValues []string) (*dataframe.DataFrame, error) {
	// Check if the column already exists in the DataFrame
	var err error
	columnExists := false
	for _, name := range df.Names() {
		if name == columnName {
			columnExists = true
			break
		}
	}

	if columnExists {
		// If the column exists, add the new values to the existing ones
		existingCol := df.Col(columnName).Records()
		updatedValues := make([]string, len(existingCol))

		// Loop through and add the values (assuming string to float conversion)
		for i, existingVal := range existingCol {
			// Convert string to float for addition
			var existingFloat float64
			var newFloat float64
			if existingVal == "0.0" {
				existingFloat = 0.0
			} else {
				existingFloat, err = strconv.ParseFloat(existingVal, 64)
				if err != nil {
					return nil, err
				}
			}
			if newValues[i] == "0.0" {
				newFloat = 0.0
			} else {
				newFloat, err = strconv.ParseFloat(newValues[i], 64)
				if err != nil {
					return nil, err
				}
			}

			// Add the existing and new values together
			sum := existingFloat + newFloat

			// Convert back to string and store in updatedValues
			updatedValues[i] = fmt.Sprintf("%.2f", sum)
		}

		// Create a new series with the updated values
		updatedSeries := series.New(updatedValues, series.String, columnName)

		// Mutate the DataFrame with the updated series
		df = df.Mutate(updatedSeries)

	} else {
		// If the column doesn't exist, create a new column with the new values
		newSeries := series.New(newValues, series.String, columnName)
		df = df.Mutate(newSeries)
	}

	return &df, nil
}

func applyStylesToAllSheets(f *excelize.File) error {
	// Define styles for the first column (DateRequested)
	firstColumnStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "000000"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"DDEBF7"}, Pattern: 1},
	})
	if err != nil {
		return err
	}

	// Define two alternating tones of blue for the first row (header row)
	lightBlueStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"ADD8E6"}, Pattern: 1}, // Light blue
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
	})
	if err != nil {
		return err
	}

	blueStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "FFFFFF"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"4682B4"}, Pattern: 1}, // Blue
		Alignment: &excelize.Alignment{
			Horizontal: "center",
			Vertical:   "center",
		},
	})
	if err != nil {
		return err
	}

	// Define a style for the second row (Asset IDs)
	secondRowStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Italic: true, Color: "000000"},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"E2EFDA"}, Pattern: 1}, // Light green
	})
	if err != nil {
		return err
	}

	// Get all the sheet names
	sheets := f.GetSheetList()

	// Loop through each sheet and apply the styles
	for _, sheetName := range sheets {
		// Apply style to the first column (A)
		err = f.SetColStyle(sheetName, "A", firstColumnStyle)
		if err != nil {
			return err
		}

		// Loop through each column in the first row (starting from B) to alternate colors
		for i := 0; i < 10; i++ { // Assuming there are 10 columns, adjust as necessary
			colLetter := string('B' + rune(i)) // Columns starting from B

			// Alternate between light blue and blue
			if i%2 == 0 {
				err = f.SetCellStyle(sheetName, fmt.Sprintf("%s1", colLetter), fmt.Sprintf("%s1", colLetter), lightBlueStyle)
			} else {
				err = f.SetCellStyle(sheetName, fmt.Sprintf("%s1", colLetter), fmt.Sprintf("%s1", colLetter), blueStyle)
			}
			if err != nil {
				return err
			}
		}

		// Apply style to the second row
		err = f.SetRowStyle(sheetName, 2, 2, secondRowStyle)
		if err != nil {
			return err
		}
	}

	return nil
}

// sortHoldingsByDate sorts the holdings by DateRequested.

// divideByTotal divides each column value in a dataframe by the "TOTAL" column value for that row.
// Returns a new dataframe with percentages.
func divideByTotal(df *dataframe.DataFrame) *dataframe.DataFrame {
	// Apply the transformation row-wise
	newDF := df.Rapply(func(row series.Series) series.Series {
		// Get the total value for the current row (last column)
		total := row.Elem(df.Ncol() - 1).Float()
		if total == 0 {
			return row
		}
		// Create a new series to store the modified row
		newRow := make([]interface{}, row.Len())
		newRow[0] = row.Elem(0)
		// Divide all columns (except TOTAL) by the total value
		for i := 1; i < df.Ncol()-1; i++ {
			newRow[i] = (row.Elem(i).Float() / total)
		}

		// Keep the TOTAL column unchanged
		newRow[df.Ncol()-1] = total

		return series.New(newRow, series.String, row.Name)
	})

	_ = newDF.SetNames(df.Names()...)

	return &newDF
}
