package render

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"html/template"
	"os"

	"github.com/SebastiaanKlippert/go-wkhtmltopdf"
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
)

func RenderHTMLWithCSS(templatePath string, cssPath string) (string, error) {
	// Read the HTML template
	tpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return "", err
	}

	// Read the CSS content
	css, err := os.ReadFile(cssPath)
	if err != nil {
		return "", err
	}

	// Render the HTML template with CSS
	var output bytes.Buffer
	err = tpl.Execute(&output, map[string]string{"CSS": string(css)})
	if err != nil {
		return "", err
	}

	return output.String(), nil
}

func RenderPieGraph(templatePath string, cssPath string, data map[string]float64) (string, error) {
	// Generate the pie chart
	pie := charts.NewPie()
	pie.SetGlobalOptions(charts.WithTitleOpts(opts.Title{Title: "Pie Chart"}))

	items := make([]opts.PieData, 0)
	for k, v := range data {
		items = append(items, opts.PieData{Name: k, Value: v})
	}
	pie.AddSeries("Categories", items)

	// Render chart to an image (Base64 encoded)
	var chartBuffer bytes.Buffer
	if err := pie.Render(&chartBuffer); err != nil {
		return "", err
	}

	chartBase64 := base64.StdEncoding.EncodeToString(chartBuffer.Bytes())

	// Read CSS and HTML template
	tpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return "", err
	}

	css, err := os.ReadFile(cssPath)
	if err != nil {
		return "", err
	}

	// Render the final HTML
	var output bytes.Buffer
	err = tpl.Execute(&output, map[string]string{
		"CSS":       string(css),
		"GraphBase": chartBase64,
	})
	if err != nil {
		return "", err
	}

	return output.String(), nil
}

func RenderBarGraph(templatePath string, cssPath string, data map[string]float64) (string, error) {
	// Generate the bar chart
	bar := charts.NewBar()
	bar.SetGlobalOptions(charts.WithTitleOpts(opts.Title{Title: "Bar Chart"}))

	items := make([]opts.BarData, 0)
	labels := make([]string, 0)
	for k, v := range data {
		items = append(items, opts.BarData{Value: v})
		labels = append(labels, k)
	}
	bar.SetXAxis(labels).AddSeries("Values", items)

	// Render chart to an image (Base64 encoded)
	var chartBuffer bytes.Buffer
	if err := bar.Render(&chartBuffer); err != nil {
		return "", err
	}

	chartBase64 := base64.StdEncoding.EncodeToString(chartBuffer.Bytes())

	// Read CSS and HTML template
	tpl, err := template.ParseFiles(templatePath)
	if err != nil {
		return "", err
	}

	css, err := os.ReadFile(cssPath)
	if err != nil {
		return "", err
	}

	// Render the final HTML
	var output bytes.Buffer
	err = tpl.Execute(&output, map[string]string{
		"CSS":       string(css),
		"GraphBase": chartBase64,
	})
	if err != nil {
		return "", err
	}

	return output.String(), nil
}

// GeneratePDF generates a PDF from an array of HTML strings
func GeneratePDF(htmlContents []string) (*bytes.Buffer, error) {
	// Create a new PDF generator
	pdfg, err := wkhtmltopdf.NewPDFGenerator()
	if err != nil {
		return nil, fmt.Errorf("failed to create PDF generator: %w", err)
	}

	// Add each HTML string as a page in the PDF
	for _, html := range htmlContents {
		page := wkhtmltopdf.NewPageReader(bytes.NewReader([]byte(html)))
		pdfg.AddPage(page)
	}

	// Set global options
	pdfg.Dpi.Set(300)                                     // Set DPI for high-quality output
	pdfg.Orientation.Set(wkhtmltopdf.OrientationPortrait) // Set orientation
	pdfg.PageSize.Set(wkhtmltopdf.PageSizeA4)             // Set page size

	// Generate the PDF
	err = pdfg.Create()
	if err != nil {
		return nil, fmt.Errorf("failed to generate PDF: %w", err)
	}

	// Return the generated PDF as a buffer
	return bytes.NewBuffer(pdfg.Bytes()), nil
}
