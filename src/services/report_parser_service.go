package services

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"server/src/schemas"
	"server/src/utils"
	"server/src/utils/render"
	"slices"
	"strconv"
	"strings"
	"text/template"

	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/go-gota/gota/dataframe"
)

// findProjectRoot finds the project root directory by looking for go.mod file
func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		// Check if go.mod exists in current directory
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}

		// Move up one directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached root directory without finding go.mod
			return "", fmt.Errorf("could not find go.mod file in any parent directory")
		}
		dir = parent
	}
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

type ReportParserServiceI interface {
	ParseAccountsReportToPDF(ctx context.Context, dataframesAndCharts *schemas.ReportDataframes) ([]byte, error)
}

type ReportParserService struct{}

func NewReportParserService() *ReportParserService {
	return &ReportParserService{}
}

// ParseAccountsReportToPDF generates bar graphs and pie charts, embeds them in HTML, and creates a PDF.
func (rc *ReportParserService) ParseAccountsReportToPDF(ctx context.Context, dataframesAndCharts *schemas.ReportDataframes) ([]byte, error) {
	var htmlContents []string
	returnsDF := *dataframesAndCharts.ReturnDF
	referenceVariablesDF := *dataframesAndCharts.ReferenceVariablesDF
	returnWithReferencesDF := utils.UnionDataFramesByIndex(returnsDF, referenceVariablesDF, "DateRequested")
	orderedReturnWithReferencesDF := utils.SortDataFrameColumns(&returnWithReferencesDF, []string{"DateRequested"}, []string{"TOTAL"})
	// Generate bar graphs for each dataframe
	for _, report := range []*ReportConfig{
		{name: "RETORNO", df: orderedReturnWithReferencesDF, columnsToInclude: []string{"Inflacion Mensual", "USD A3500 Variacion", "TOTAL"}, graphType: "line", isPercentage: true, includeTable: true},
		{name: "TENENCIA POR CATEGORIAS", df: dataframesAndCharts.CategoryDF, columnsToExclude: []string{"TOTAL"}, graphType: "line", includeTable: true},
		// {name: "TENENCIA POR CATEGORIAS PORCENTAJE", df: dataframesAndCharts.CategoryPercentageDF, columnsToExclude: []string{"TOTAL"}, graphType: "bar", isPercentage: true},
		{name: "TENENCIA POR CATEGORIAS PORCENTAJE", df: dataframesAndCharts.ReportPercentageDf, graphType: "pie", columnsToExclude: []string{"TOTAL"}, isPercentage: true},
		{name: "TENENCIA POR CATEGORIAS PORCENTAJE", df: dataframesAndCharts.ReportPercentageDf, columnsToExclude: []string{"TOTAL"}, graphType: "bar", isPercentage: true},
		{name: "TENENCIA TOTAL", df: dataframesAndCharts.ReportDF, columnsToInclude: []string{"TOTAL"}, graphType: "line", includeTable: true},
	} {
		if report.df == nil {
			continue
		}
		var htmlContent string
		var err error

		if report.includeTable {
			htmlContent, err = render.GetTableHTML(report.name, report.df)
			if err != nil {
				return nil, fmt.Errorf("failed to generate table for %s: %w", report.name, err)
			}
			htmlContents = append(htmlContents, htmlContent)
		}

		// Generate bar graph and embed in HTML
		switch report.graphType {
		case "bar":
			htmlContent, err = rc.generateStackBarGraphHTML(report)
		case "pie":
			htmlContent, err = rc.generatePieChartHTML(report)
		case "line":
			htmlContent, err = rc.generateLineGraphHTML(report)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to generate graph for %s: %w", report.name, err)
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

func (rc *ReportParserService) generateLineGraphHTML(report *ReportConfig) (string, error) {
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
			Width:  "1100px",
			Height: "600px",
		}),
	)

	// Extract labels (dates) and data
	labels := df.Col("DateRequested").Records()
	line.SetXAxis(labels)

	colorIndex := 0
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
			data = append(data, opts.LineData{Name: label, Value: v})
		}
		line.AddSeries(asset, data,
			charts.WithLabelOpts(opts.Label{
				Show:      opts.Bool(true),
				Formatter: "{b}",
			}),
			charts.WithAreaStyleOpts(opts.AreaStyle{
				Opacity: opts.Float(0.2),
			}),
			charts.WithLineChartOpts(opts.LineChart{
				Smooth: opts.Bool(true),
			}),
			// Add distinct color for this series
			charts.WithItemStyleOpts(opts.ItemStyle{
				Color: utils.GetChartColor(colorIndex),
			}),
		)
		colorIndex++
	}
	baseDir, err := findProjectRoot()
	if err != nil {
		return "", fmt.Errorf("failed to find project root: %w", err)
	}
	// Load HTML template
	tmpl, err := template.ParseFiles(fmt.Sprintf("%s/templates/bar_graph.html", baseDir))
	if err != nil {
		return "", fmt.Errorf("failed to load bar graph template: %w", err)
	}

	// Render HTML embedding the chart image
	var htmlBuffer bytes.Buffer
	err = tmpl.Execute(&htmlBuffer, map[string]interface{}{
		"Graph": strings.ReplaceAll(string(line.RenderContent()), "let ", "var "),
		"Title": report.name,
	})
	if err != nil {
		return "", fmt.Errorf("failed to render bar graph HTML: %w", err)
	}

	return htmlBuffer.String(), nil
}

func (rc *ReportParserService) generateStackBarGraphHTML(report *ReportConfig) (string, error) {
	df := report.df
	// Create a bar chart
	bar := charts.NewBar()
	bar.SetGlobalOptions(
		charts.WithAnimation(false),
		charts.WithYAxisOpts(opts.YAxis{
			SplitLine: &opts.SplitLine{
				Show: opts.Bool(true),
			},
		}),
		charts.WithInitializationOpts(opts.Initialization{
			Width:  "1100px",
			Height: "600px",
		}),
		// Add stacking configuration
		charts.WithTooltipOpts(opts.Tooltip{
			Show: opts.Bool(true),
		}),
	)

	// Extract labels (dates) and data
	labels := df.Col("DateRequested").Records()
	bar.SetXAxis(labels)

	colorIndex := 0
	for _, asset := range df.Names()[1:] {
		if (len(report.columnsToInclude) != 0 && !slices.Contains(report.columnsToInclude, asset)) || slices.Contains(report.columnsToExclude, asset) {
			continue
		}
		data := make([]opts.BarData, 0)
		for _, value := range df.Col(asset).Records() {
			v, _ := strconv.ParseFloat(value, 32)
			var label string
			if report.isPercentage {
				label = render.FormatPercentageValue(value)
			} else {
				label = render.FormatMonetaryValue(value)
			}
			if v <= 0 {
				data = append(data, opts.BarData{Name: label, Value: 0})
				continue
			}
			data = append(data, opts.BarData{Name: label, Value: roundFloat(v), Label: &opts.Label{
				Show:      opts.Bool(true),
				Formatter: "{c}%",
			}})
		}
		bar.AddSeries(asset, data,
			// Enable stacking for this series
			charts.WithBarChartOpts(opts.BarChart{
				Stack: "Total",
			}),
			// Add distinct color for this series
			charts.WithItemStyleOpts(opts.ItemStyle{
				Color: utils.GetChartColor(colorIndex),
			}),
		)
		colorIndex++
	}
	baseDir, err := findProjectRoot()
	if err != nil {
		return "", fmt.Errorf("failed to find project root: %w", err)
	}
	// Load HTML template
	tmpl, err := template.ParseFiles(fmt.Sprintf("%s/templates/bar_graph.html", baseDir))
	if err != nil {
		return "", fmt.Errorf("failed to load bar graph template: %w", err)
	}

	// Render HTML embedding the chart image
	var htmlBuffer bytes.Buffer
	err = tmpl.Execute(&htmlBuffer, map[string]interface{}{
		"Graph": strings.ReplaceAll(string(bar.RenderContent()), "let ", "var "),
		"Title": report.name,
	})
	if err != nil {
		return "", fmt.Errorf("failed to render bar graph HTML: %w", err)
	}

	return htmlBuffer.String(), nil
}

func (rc *ReportParserService) generatePieChartHTML(report *ReportConfig) (string, error) {
	df := report.df
	// Create a pie chart
	pie := charts.NewPie()
	pie.SetGlobalOptions(
		charts.WithAnimation(false),
		charts.WithInitializationOpts(opts.Initialization{
			Width:  "1100px",
			Height: "600px",
		}),
	)

	// Get the last row of data for pie chart
	lastRowIndex := df.Nrow() - 1
	if lastRowIndex < 0 {
		return "", fmt.Errorf("no data available for pie chart")
	}

	// Extract data from the last row
	var pieData []opts.PieData
	colorIndex := 0
	for _, asset := range df.Names()[1:] {
		if (len(report.columnsToInclude) != 0 && !slices.Contains(report.columnsToInclude, asset)) || slices.Contains(report.columnsToExclude, asset) {
			continue
		}
		value := df.Col(asset).Elem(lastRowIndex).String()
		v, _ := strconv.ParseFloat(value, 32)
		if v > 0 {
			pieData = append(pieData, opts.PieData{
				Name:  asset,
				Value: roundFloat(v),
				ItemStyle: &opts.ItemStyle{
					Color: utils.GetChartColor(colorIndex),
				},
			})
			colorIndex++
		}
	}

	pie.AddSeries("", pieData,
		charts.WithLabelOpts(opts.Label{
			Show:      opts.Bool(true),
			Formatter: "{b}: {c}%",
		}),
	)

	baseDir, err := findProjectRoot()
	if err != nil {
		return "", fmt.Errorf("failed to find project root: %w", err)
	}
	// Load HTML template
	tmpl, err := template.ParseFiles(fmt.Sprintf("%s/templates/pie_graph.html", baseDir))
	if err != nil {
		return "", fmt.Errorf("failed to load pie graph template: %w", err)
	}

	// Render HTML embedding the chart image
	var htmlBuffer bytes.Buffer
	err = tmpl.Execute(&htmlBuffer, map[string]interface{}{
		"Graph": strings.ReplaceAll(string(pie.RenderContent()), "let ", "var "),
		"Title": report.name,
	})
	if err != nil {
		return "", fmt.Errorf("failed to render pie graph HTML: %w", err)
	}

	return htmlBuffer.String(), nil
}

// Function for rounding float to float with 2 decimal places
func roundFloat(value float64) float64 {
	return math.Round(value*100) / 100
}
