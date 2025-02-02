package controllers

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"server/src/clients/bcra"
	"server/src/clients/esco"
	"server/src/schemas"
	"server/src/utils"
	"sort"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/SebastiaanKlippert/go-wkhtmltopdf"
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-echarts/go-echarts/v2/types"
	"github.com/go-gota/gota/dataframe"
	"github.com/go-gota/gota/series"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

type ReportsControllerI interface {
	GetReport(ctx context.Context, accountsStates *schemas.AccountStateByCategory, variablesWithValuations []*schemas.VariableWithValuationResponse, startDate, endDate time.Time, interval time.Duration) (*schemas.AccountsReports, error)
	GetReportScheduleByID(ctx context.Context, ID uint) (*schemas.ReportScheduleResponse, error)
	GetAllReportSchedules(ctx context.Context) ([]*schemas.ReportScheduleResponse, error)
	CreateReportSchedule(ctx context.Context, req *schemas.CreateReportScheduleRequest) (*schemas.ReportScheduleResponse, error)
	UpdateReportSchedule(ctx context.Context, req *schemas.UpdateReportScheduleRequest) (*schemas.ReportScheduleResponse, error)
	DeleteReportSchedule(ctx context.Context, id uint) error

	ParseAccountsReportToDataFrames(ctx context.Context, accountsReport *schemas.AccountsReports, startDate, endDate time.Time, interval time.Duration) (*schemas.ReportDataframes, error)
	ParseAccountsReportToXLSX(ctx context.Context, dataframesAndCharts *schemas.ReportDataframes) (*excelize.File, error)
	ParseAccountsReportToPDF(ctx context.Context, dataframesAndCharts *schemas.ReportDataframes) ([]byte, error)
	ParseAccountsReportToDataFrame(ctx context.Context, accountsReport *schemas.AccountsReports, startDate, endDate time.Time, interval time.Duration) (*dataframe.DataFrame, error)
	ParseAccountsReturnToDataFrame(ctx context.Context, accountsReport *schemas.AccountsReports, startDate, endDate time.Time, interval time.Duration) (*dataframe.DataFrame, error)
}

type ReportsController struct {
	ESCOClient esco.ESCOServiceClientI
	BCRAClient bcra.BCRAServiceClientI
	DB         *gorm.DB
}

func NewReportsController(escoClient esco.ESCOServiceClientI, bcraClient bcra.BCRAServiceClientI, db *gorm.DB) *ReportsController {
	return &ReportsController{ESCOClient: escoClient, BCRAClient: bcraClient, DB: db}
}

func (rc *ReportsController) GetReport(
	ctx context.Context,
	accountsStates *schemas.AccountStateByCategory,
	variablesWithValuations []*schemas.VariableWithValuationResponse,
	startDate, endDate time.Time,
	interval time.Duration,
) (*schemas.AccountsReports, error) {
	accountReports, err := GenerateAccountReports(accountsStates, startDate, endDate, interval)
	if err != nil {
		return nil, err
	}
	accountReports.ReferenceVariables = variablesWithValuations
	return accountReports, nil
}

// GenerateAccountReports calculates the return for each voucher per category and returns an AccountsReports struct.
func GenerateAccountReports(
	accountStateByCategory *schemas.AccountStateByCategory,
	startDate, endDate time.Time,
	interval time.Duration) (*schemas.AccountsReports, error) {
	voucherReturnsByCategory := make(map[string][]schemas.VoucherReturn)

	// Iterate through each category and its associated vouchers
	for category, vouchers := range *accountStateByCategory.VouchersByCategory {
		if category == "ARS" {
			continue
		}
		for _, voucher := range vouchers {
			voucherReturn, _ := CalculateVoucherReturn(voucher, interval)
			voucherReturnsByCategory[category] = append(voucherReturnsByCategory[category], voucherReturn)
		}
	}

	totalHoldingsByDate := make([]schemas.Holding, 0, len(*accountStateByCategory.TotalHoldingsByDate))
	for _, holding := range *accountStateByCategory.TotalHoldingsByDate {
		totalHoldingsByDate = append(totalHoldingsByDate, holding)
	}
	totalTransactionsByDate := make([]schemas.Transaction, 0, len(*accountStateByCategory.TotalTransactionsByDate))
	for _, transaction := range *accountStateByCategory.TotalTransactionsByDate {
		totalTransactionsByDate = append(totalTransactionsByDate, transaction)
	}
	totalReturns := CalculateHoldingsReturn(totalHoldingsByDate, []schemas.Transaction{}, interval)
	finalIntervalReturn := CalculateFinalIntervalReturn(totalReturns)
	filteredVouchers := filterVoucherHoldingsByInterval(accountStateByCategory.VouchersByCategory, startDate, endDate, interval)
	filteredTotalHoldings := filterHoldingsByInterval(totalHoldingsByDate, startDate, endDate, interval)
	return &schemas.AccountsReports{
		VouchersByCategory:       &filteredVouchers,
		VouchersReturnByCategory: &voucherReturnsByCategory,
		TotalHoldingsByDate:      filteredTotalHoldings,
		TotalTransactionsByDate:  totalTransactionsByDate,
		TotalReturns:             totalReturns,
		FinalIntervalReturn:      finalIntervalReturn,
	}, nil
}

func filterVoucherHoldingsByInterval(vouchersByCategory *map[string][]schemas.Voucher, startDate, endDate time.Time, interval time.Duration) map[string][]schemas.Voucher {
	filteredVouchersByCategory := make(map[string][]schemas.Voucher)

	for category, vouchers := range *vouchersByCategory {
		for _, voucher := range vouchers {
			filteredHoldings := filterHoldingsByInterval(voucher.Holdings, startDate, endDate, interval)

			if len(filteredHoldings) > 0 {
				voucher.Holdings = filteredHoldings
				filteredVouchersByCategory[category] = append(filteredVouchersByCategory[category], voucher)
			}
		}
	}

	return filteredVouchersByCategory
}

func filterHoldingsByInterval(holdings []schemas.Holding, startDate, endDate time.Time, interval time.Duration) []schemas.Holding {

	filteredHoldings := []schemas.Holding{}

	// Generate interval boundaries
	for date := startDate; date.Before(endDate); date = date.Add(interval) {

		for _, holding := range holdings {
			// Include holdings that fall within the exact interval
			if date == *holding.DateRequested {
				filteredHoldings = append(filteredHoldings, holding)
			}
		}
	}

	return filteredHoldings
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

	var wg sync.WaitGroup
	wg.Add(3)
	var errChan = make(chan error, 3)
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

	// Iterate through the vouchers and add each as a new column
	for _, vouchers := range *accountsReport.VouchersByCategory {
		for _, voucher := range vouchers {
			voucherValues := make([]string, len(dates))

			// Initialize all rows with empty values for this voucher
			for i := range voucherValues {
				voucherValues[i] = "0.0" // Default value if no match found
			}

			// Iterate through holdings and match the dates to fill the corresponding values
			for _, holding := range voucher.Holdings {
				if holding.DateRequested != nil {
					dateStr := holding.DateRequested.Format("2006-01-02")
					// Find the index in the dates array that matches this holding's date
					for i, date := range dateStrs {
						if date == dateStr {
							if holding.Value >= 1.0 {
								voucherValues[i] = fmt.Sprintf("%.2f", holding.Value)
							} else {
								voucherValues[i] = "0.0"
							}
							break
						}
					}
				}
			}

			// Add the new series (column) for this voucher to the DataFrame
			updatedDf, err := updateDataFrame(df, fmt.Sprintf("%s-%s", voucher.Category, voucher.ID), voucherValues)
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
	// Add the new series (column) for this voucher to the DataFrame
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

	// Iterate through the vouchers and add each as a new column
	for _, vouchers := range *accountsReport.VouchersReturnByCategory {
		for _, voucher := range vouchers {
			voucherValues := make([]string, len(dates))

			// Initialize all rows with empty values for this voucher
			for i := range voucherValues {
				voucherValues[i] = "0.0" // Default value if no match found
			}

			// Iterate through vouchers return and match the dates to fill the corresponding values
			for _, returnsByDate := range voucher.ReturnsByDateRange {
				dateStr := returnsByDate.EndDate.Format("2006-01-02")
				// Find the index in the dates array that matches this holding's date
				for i, date := range dateStrs {
					if date == dateStr {
						voucherValues[i] = fmt.Sprintf("%.2f", returnsByDate.ReturnPercentage)
						break
					}
				}
			}

			// Add the new series (column) for this voucher to the DataFrame
			updatedDf, err := updateDataFrame(df, fmt.Sprintf("%s-%s", voucher.Category, voucher.ID), voucherValues)
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
	// Add the new series (column) for this voucher to the DataFrame
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

	// Iterate through the vouchers and add each as a new column
	for _, referenceVariable := range (*accountsReport).ReferenceVariables {
		for _, valuation := range referenceVariable.Valuations {
			valuationValues := make([]string, len(dates))

			// Initialize all rows with empty values for this voucher
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

			// Add the new series (column) for this voucher to the DataFrame
			updatedDf, err := updateDataFrame(df, fmt.Sprintf("Variables de Referencia-%s", referenceVariable.Description), valuationValues)
			if err != nil {
				return nil, err
			}
			df = *updatedDf
		}
	}

	return sortDataFrameColumns(&df), nil
}

type ReportConfig struct {
	df        *dataframe.DataFrame
	graphType string
	onlyTotal bool
}

// ParseAccountsReportToPDF generates bar graphs and pie charts, embeds them in HTML, and creates a PDF.
func (rc *ReportsController) ParseAccountsReportToPDF(ctx context.Context, dataframesAndCharts *schemas.ReportDataframes) ([]byte, error) {
	var htmlContents []string

	// Generate bar graphs for each dataframe
	for name, report := range map[string]*ReportConfig{
		"Tenencia":                {df: dataframesAndCharts.ReportDF, onlyTotal: false, graphType: "line"},
		"Tenencia Total":          {df: dataframesAndCharts.ReportDF, onlyTotal: true, graphType: "line"},
		"Porcentaje Tenencia":     {df: dataframesAndCharts.ReportPercentageDf, graphType: "pie"},
		"Retorno":                 {df: dataframesAndCharts.ReturnDF, onlyTotal: true, graphType: "line"},
		"Variables de Referencia": {df: dataframesAndCharts.ReferenceVariablesDF, graphType: "line"},
	} {
		if report.df == nil {
			continue
		}
		var htmlContent string
		var err error

		htmlContent, err = getTableHTML(report.df)
		if err != nil {
			return nil, fmt.Errorf("failed to generate table for %s: %w", name, err)
		}
		htmlContents = append(htmlContents, htmlContent)
		// Generate bar graph and embed in HTML
		if report.graphType == "line" {
			htmlContent, err = rc.generateLineGraphHTML(name, report)
		} else if report.graphType == "pie" {
			htmlContent, err = rc.generatePieChartHTML(name, report)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to generate graph for %s: %w", name, err)
		}
		htmlContents = append(htmlContents, htmlContent)
	}

	// Convert all HTML content into a PDF
	pdfBuffer, err := generatePDF(htmlContents)
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
		if report.onlyTotal && asset != "TOTAL" || !report.onlyTotal && asset == "TOTAL" {
			continue
		}
		data := make([]opts.LineData, 0)
		for _, value := range df.Col(asset).Records() {
			v, _ := strconv.ParseFloat(value, 32)
			data = append(data, opts.LineData{Value: int(v)})
		}
		line.AddSeries(asset, data,
			charts.WithLabelOpts(opts.Label{
				Show: opts.Bool(true),
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
		value := df.Elem(lastRowIndex, colIndex).Float()
		items = append(items, opts.PieData{Name: colName, Value: value})
	}

	pie.AddSeries("Data", items).SetSeriesOptions(
		charts.WithLabelOpts(opts.Label{
			Position: "top",
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

// getReportCoverHTML reads the cover template and injects the title, subtitle, and image path
func getReportCoverHTML(title, subtitle, imagePath string) (string, error) {
	// Get the working directory
	baseDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	// Define the template path
	templatePath := filepath.Join(baseDir, "templates", "cover.html")

	// Read and parse the template file
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return "", fmt.Errorf("failed to load cover page template: %w", err)
	}

	// Define template data
	data := map[string]string{
		"Title":     title,
		"Subtitle":  subtitle,
		"ImagePath": imagePath,
	}

	// Execute the template with provided data
	var htmlBuffer bytes.Buffer
	err = tmpl.Execute(&htmlBuffer, data)
	if err != nil {
		return "", fmt.Errorf("failed to render cover page HTML: %w", err)
	}

	return htmlBuffer.String(), nil
}

func getTableHTML(df *dataframe.DataFrame) (string, error) {
	if df == nil || df.Nrow() == 0 {
		return "", fmt.Errorf("dataframe is empty or nil")
	}

	// Get the working directory
	baseDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	// Define the template path
	templatePath := filepath.Join(baseDir, "templates", "table.html")

	// Parse the HTML template file
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return "", fmt.Errorf("failed to load table template: %w", err)
	}

	// Extract headers and rows from dataframe
	headers := df.Names()
	rows := make([][]interface{}, df.Nrow())

	for i := 0; i < df.Nrow(); i++ {
		row := make([]interface{}, len(headers))
		for j, _ := range headers {
			row[j] = df.Elem(i, j).String()
		}
		rows[i] = row
	}

	// Define the template data
	data := map[string]interface{}{
		"Headers": headers,
		"Rows":    rows,
	}

	// Execute the template with the data
	var htmlBuffer bytes.Buffer
	err = tmpl.Execute(&htmlBuffer, data)
	if err != nil {
		return "", fmt.Errorf("failed to render table HTML: %w", err)
	}

	return htmlBuffer.String(), nil
}

func generatePDF(htmlContents []string) (*bytes.Buffer, error) {
	pdfg, err := wkhtmltopdf.NewPDFGenerator()
	if err != nil {
		return nil, err
	}
	baseDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get working directory: %w", err)
	}

	// Define the template path
	imagePath := filepath.Join(baseDir, "assets", "criteria_logo.png")
	cover, err := getReportCoverHTML("Reporte de Rendimientos", "Criteria 2025", imagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create cover: %w", err)
	}
	html := joinHTMLPages(append([]string{cover}, htmlContents...))
	page := wkhtmltopdf.NewPageReader(bytes.NewReader([]byte(html)))
	page.EnableLocalFileAccess.Set(true)

	pdfg.AddPage(page)

	pdfg.Orientation.Set(wkhtmltopdf.OrientationLandscape)
	pdfg.PageSize.Set(wkhtmltopdf.PageSizeA4)

	err = pdfg.Create()
	if err != nil {
		return nil, err
	}

	return bytes.NewBuffer(pdfg.Bytes()), nil
}

func saveHTMLToFile(htmlContent, filePath string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create HTML file: %w", err)
	}
	defer file.Close()

	_, err = file.WriteString(htmlContent)
	if err != nil {
		return fmt.Errorf("failed to write HTML content to file: %w", err)
	}

	return nil
}

func joinHTMLPages(htmlContents []string) string {
	// Define the CSS to enforce page breaks between sections
	pageBreakCSS := `<style>
		.page-break { page-break-before: always; }
	</style>`

	// Start building the final HTML document
	var htmlBuilder bytes.Buffer
	htmlBuilder.WriteString("<!DOCTYPE html><html><head><meta charset='UTF-8'><title>Report</title>")
	htmlBuilder.WriteString(pageBreakCSS) // Add CSS styling for page breaks
	htmlBuilder.WriteString("</head><body>")

	// Append each HTML content with a page break
	for i, html := range htmlContents {
		htmlBuilder.WriteString(html)
		if i < len(htmlContents)-1 {
			htmlBuilder.WriteString("<div class='page-break'></div>") // Add page break between sections
		}
	}

	htmlBuilder.WriteString("</body></html>")

	return htmlBuilder.String()
}

// CalculateVoucherReturn calculates the return for a single voucher by taking holdings in pairs and applying transactions within the date ranges.
func CalculateVoucherReturn(voucher schemas.Voucher, interval time.Duration) (schemas.VoucherReturn, error) {
	if len(voucher.Holdings) < 2 {
		return schemas.VoucherReturn{}, fmt.Errorf("insufficient holdings data to calculate return for voucher %s", voucher.ID)
	}

	returnsByInterval := CalculateHoldingsReturn(voucher.Holdings, voucher.Transactions, interval)

	// Return the result
	return schemas.VoucherReturn{
		ID:                 voucher.ID,
		Type:               voucher.Type,
		Denomination:       voucher.Denomination,
		Category:           voucher.Category,
		ReturnsByDateRange: returnsByInterval,
	}, nil
}

func CalculateFinalIntervalReturn(totalReturns []schemas.ReturnByDate) float64 {
	intervalReturn := 1.0
	for _, totalReturn := range totalReturns {
		intervalReturn *= 1 + (totalReturn.ReturnPercentage / 100)
	}
	return intervalReturn
}

func CalculateHoldingsReturn(holdings []schemas.Holding, transactions []schemas.Transaction, interval time.Duration) []schemas.ReturnByDate {
	// Sort holdings by date
	sortedHoldings := sortHoldingsByDate(holdings)
	var dailyReturns []schemas.ReturnByDate

	// Iterate through each pair of consecutive holdings
	for i := 0; i < len(sortedHoldings)-1; i++ {
		startingHolding := sortedHoldings[i]
		endingHolding := sortedHoldings[i+1]

		startDate := *startingHolding.DateRequested
		endDate := *endingHolding.DateRequested
		if endDate.Sub(startDate) > 24*time.Hour {
			continue
		}
		startingValue := startingHolding.Value
		endingValue := endingHolding.Value

		// Calculate the net transactions within the date range
		var netTransactions float64
		for _, transaction := range transactions {
			if transaction.Date != nil && transaction.Date.Equal(endDate) {
				netTransactions += transaction.Value
			}
		}
		netEndValue := endingValue + netTransactions
		// Calculate return for this date range
		if startingValue < 1 && startingValue > -1 {
			continue
		}

		returnPercentage := ((netEndValue - startingValue) / startingValue) * 100
		// Append the return for this date range
		dailyReturns = append(dailyReturns, schemas.ReturnByDate{
			StartDate:        startDate,
			EndDate:          endDate,
			ReturnPercentage: returnPercentage,
		})
	}
	// Collapse daily returns into intervals
	return CollapseReturnsByInterval(dailyReturns, interval)
}

func CollapseReturnsByInterval(dailyReturns []schemas.ReturnByDate, interval time.Duration) []schemas.ReturnByDate {
	var returnsByInterval []schemas.ReturnByDate
	var (
		currentIntervalStart time.Time
		currentIntervalEnd   time.Time
		compoundReturn       float64 = 1.0
	)

	for _, dailyReturn := range dailyReturns {
		if currentIntervalStart.IsZero() {
			currentIntervalStart = dailyReturn.StartDate
			currentIntervalEnd = currentIntervalStart.Add(interval)
		}

		if dailyReturn.EndDate.Before(currentIntervalEnd) {
			// Apply compound calculation
			compoundReturn *= 1 + (dailyReturn.ReturnPercentage / 100)
		} else {
			// Close the current interval
			returnsByInterval = append(returnsByInterval, schemas.ReturnByDate{
				StartDate:        currentIntervalStart,
				EndDate:          currentIntervalEnd,
				ReturnPercentage: (compoundReturn - 1) * 100,
			})

			// Reset for the new interval
			currentIntervalStart = currentIntervalEnd
			currentIntervalEnd = currentIntervalStart.Add(interval)
			compoundReturn = 1 + (dailyReturn.ReturnPercentage / 100)
		}
	}

	// Append the last interval
	if !currentIntervalStart.IsZero() {
		returnsByInterval = append(returnsByInterval, schemas.ReturnByDate{
			StartDate:        currentIntervalStart,
			EndDate:          currentIntervalEnd,
			ReturnPercentage: (compoundReturn - 1) * 100,
		})
	}

	return returnsByInterval
}

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

	// Define a style for the second row (Voucher IDs)
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
func sortHoldingsByDate(holdings []schemas.Holding) []schemas.Holding {
	sort.Slice(holdings, func(i, j int) bool {
		return holdings[i].DateRequested.Before(*holdings[j].DateRequested)
	})
	return holdings
}

func formatMonetaryValue(v string) string {
	value, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return v
	}
	if value >= 1_000_000_000 {
		return fmt.Sprintf("$ %.3f MM", float64(value/1_000_000_000))
	} else if value >= 1_000_000 {
		return fmt.Sprintf("$ %.1f M", float64(value/1_000_000))
	} else if value >= 1_000 {
		return fmt.Sprintf("$ %.1f K", float64(value/1_000))
	}
	return fmt.Sprintf("$ %s", v)
}

func formatPercentageValue(v string) string {
	value, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return v
	}
	if value == 0 {
		return "0.0%"
	}
	return fmt.Sprintf("%.2f%%", value)
}

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
