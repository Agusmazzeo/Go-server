package render

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strconv"

	"github.com/SebastiaanKlippert/go-wkhtmltopdf"
	"github.com/go-gota/gota/dataframe"
)

func GeneratePDF(htmlContents []string) (*bytes.Buffer, error) {
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
	cover, err := GetReportCoverHTML("Reporte de Rendimientos", "Criteria 2025", imagePath)
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

// getReportCoverHTML reads the cover template and injects the title, subtitle, and image path
func GetReportCoverHTML(title, subtitle, imagePath string) (string, error) {
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

func GetTableHTML(title string, df *dataframe.DataFrame) (string, error) {
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
		"Title":   title,
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

// getSeparatorPageHTML generates a separator page HTML with a given title and subtitle
func GetSeparatorPageHTML(title string) (string, error) {
	baseDir, _ := os.Getwd()
	tmplPath := filepath.Join(baseDir, "templates", "separator.html")

	// Parse the HTML template
	tmpl, err := template.ParseFiles(tmplPath)
	if err != nil {
		return "", fmt.Errorf("failed to load separator template: %w", err)
	}

	// Define the data to be inserted into the template
	data := map[string]string{
		"Title": title,
	}

	// Render the HTML template with data
	var htmlBuffer bytes.Buffer
	err = tmpl.Execute(&htmlBuffer, data)
	if err != nil {
		return "", fmt.Errorf("failed to render separator HTML: %w", err)
	}

	return htmlBuffer.String(), nil
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

func SaveHTMLToFile(htmlContent, filePath string) error {
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

func FormatMonetaryValue(v string) string {
	value, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return ""
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

func FormatPercentageValue(v string) string {
	value, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return ""
	}
	if value == 0 {
		return ""
	}
	return fmt.Sprintf("%.2f%%", value)
}
