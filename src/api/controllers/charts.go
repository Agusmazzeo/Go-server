package controllers

import (
	"fmt"

	"github.com/xuri/excelize/v2"
)

// addBarGraphFromSheet adds a line graph to a new sheet based on data from an existing sheet.
func addBarGraphFromSheet(file *excelize.File, dataSheet string) error {
	// Read all rows from the data sheet
	rows, err := file.GetRows(dataSheet)
	if err != nil {
		return fmt.Errorf("failed to read rows from sheet %s: %v", dataSheet, err)
	}

	// Ensure the sheet has at least headers and some data
	if len(rows) < 2 {
		return fmt.Errorf("sheet %s does not contain enough rows for a graph", dataSheet)
	}

	// Determine the range of the chart
	startColumn := "A"
	startRow := 3 // Data starts from the second row
	endRow := len(rows)

	// Generate ranges for chart data
	categories := fmt.Sprintf("%s!$%s$%d:$%s$%d", dataSheet, startColumn, startRow, startColumn, endRow)
	series := []excelize.ChartSeries{}
	for col := 1; col < len(rows[0])-1; col++ {
		colName, _ := excelize.ColumnNumberToName(col + 1)
		series = append(series, excelize.ChartSeries{
			Name:       fmt.Sprintf("%s!$%s$2", dataSheet, colName),
			Categories: categories,
			Values:     fmt.Sprintf("%s!$%s$%d:$%s$%d", dataSheet, colName, startRow, colName, endRow),
		})
	}

	titleFont := excelize.Font{
		Bold: true,
		Size: 35,
	}

	// Create the chart
	chart := excelize.Chart{
		Type:   excelize.ColStacked,
		Series: series,
		Title: []excelize.RichTextRun{
			{
				Text: dataSheet,
				Font: &titleFont,
			},
		},
		Legend: excelize.ChartLegend{
			Position:      "right",
			ShowLegendKey: true,
		},
		XAxis: excelize.ChartAxis{
			Font: excelize.Font{
				Size: 15,
			},
			MajorGridLines: true,
		},
		YAxis: excelize.ChartAxis{
			Font: excelize.Font{
				Size: 15,
			},
			MajorGridLines: true,
		},
		Dimension: excelize.ChartDimension{
			Width:  1600, // Set chart width
			Height: 1000, // Set chart height
		},
		PlotArea: excelize.ChartPlotArea{
			ShowVal: true,
			Fill: excelize.Fill{ // Set plot area background fill
				Type:  "solid",
				Color: []string{"#E6F7FF"}, // Light blue background
			},
		},
		Format: excelize.GraphicOptions{
			OffsetX: 15,
			OffsetY: 10,
		},
	}

	// Add a new sheet for the graph
	graphSheetName := fmt.Sprintf("%s - Barras", dataSheet)
	graphSheetIndex, _ := file.NewSheet(graphSheetName)

	// Add the chart to the new sheet
	if err := file.AddChart(graphSheetName, "A1", &chart); err != nil {
		return fmt.Errorf("failed to add chart to sheet %s: %v", graphSheetName, err)
	}

	// Set the new sheet as active
	file.SetActiveSheet(graphSheetIndex)

	return nil
}

// addPieChartFromLastRow adds a pie chart to a new sheet based on the last row of an existing sheet.
func addPieChartFromLastRow(file *excelize.File, dataSheet string) error {
	// Read all rows from the data sheet
	rows, err := file.GetRows(dataSheet)
	if err != nil {
		return fmt.Errorf("failed to read rows from sheet %s: %v", dataSheet, err)
	}

	// Ensure the sheet has at least headers and some data
	if len(rows) < 2 {
		return fmt.Errorf("sheet %s does not contain enough rows for a chart", dataSheet)
	}

	// Determine the range of the chart
	startColumn := "B"
	endColumn, _ := excelize.ColumnNumberToName(len(rows[0]) - 1)
	startRow := 2 // Data starts from the second row
	endRow := len(rows)

	// Generate ranges for chart data
	categories := fmt.Sprintf("%s!$%s$%d:$%s$%d", dataSheet, startColumn, startRow, endColumn, startRow)
	values := fmt.Sprintf("%s!$%s$%d:$%s$%d", dataSheet, startColumn, endRow, endColumn, endRow)

	titleFont := excelize.Font{
		Bold: true,
		Size: 35,
	}

	// Create the pie chart
	chart := excelize.Chart{
		Type: excelize.Pie,
		Series: []excelize.ChartSeries{
			{
				Name:       dataSheet,
				Categories: categories,
				Values:     values,
			},
		},
		Title: []excelize.RichTextRun{
			{
				Text: dataSheet,
				Font: &titleFont,
			},
		},
		Legend: excelize.ChartLegend{
			Position: "right",
		},
		Dimension: excelize.ChartDimension{
			Width:  1600, // Set chart width
			Height: 1000, // Set chart height
		},
		PlotArea: excelize.ChartPlotArea{
			ShowCatName: true,
			ShowVal:     true,
			ShowPercent: true,
		},
	}

	// Add a new sheet for the chart
	chartSheetName := fmt.Sprintf("%s - Pie Chart", dataSheet)
	chartSheetIndex, _ := file.NewSheet(chartSheetName)

	// Add the pie chart to the new sheet
	if err := file.AddChart(chartSheetName, "A1", &chart); err != nil {
		return fmt.Errorf("failed to add pie chart to sheet %s: %v", chartSheetName, err)
	}

	// Set the new sheet as active
	file.SetActiveSheet(chartSheetIndex)

	return nil
}
