package controllers

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"server/src/clients/bcra"
	"server/src/clients/esco"
	"server/src/models"
	"server/src/schemas"
	"server/src/utils"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-gota/gota/dataframe"
	"github.com/go-gota/gota/series"
	"github.com/jung-kurt/gofpdf"
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

	ParseAccountsReportToXLSX(ctx context.Context, accountsReport *schemas.AccountsReports, startDate, endDate time.Time, interval time.Duration) (*excelize.File, error)
	ParseAccountsReportToPDF(ctx context.Context, accountsReport *schemas.AccountsReports, startDate, endDate time.Time, interval time.Duration) ([]byte, error)
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

func (rc *ReportsController) ParseAccountsReportToXLSX(ctx context.Context, accountsReport *schemas.AccountsReports, startDate, endDate time.Time, interval time.Duration) (*excelize.File, error) {
	reportDf, err := rc.ParseAccountsReportToDataFrame(ctx, accountsReport, startDate, endDate, interval)
	if err != nil {
		return nil, err
	}
	returnsDf, err := rc.ParseAccountsReturnToDataFrame(ctx, accountsReport, startDate, endDate, interval)
	if err != nil {
		return nil, err
	}
	referenceVariablesDf, err := rc.ParseReferenceVariablesToDataFrame(ctx, accountsReport, startDate, endDate, interval)
	if err != nil {
		return nil, err
	}
	file, err := convertReportDataframeToExcel(nil, reportDf, "Tenencia")
	if err != nil {
		return nil, err
	}
	file, err = convertReportDataframeToExcel(file, returnsDf, "Retorno")
	if err != nil {
		return nil, err
	}
	file, err = convertReportDataframeToExcel(file, referenceVariablesDf, "Referencias")
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
				voucherValues[i] = "-" // Default value if no match found
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
								voucherValues[i] = "-"
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
					totalValues[i] = "-"
				}
				break
			}
		}
	}
	for i, v := range totalValues {
		if v == "" {
			totalValues[i] = "-"
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
				voucherValues[i] = "-" // Default value if no match found
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
			totalValues[i] = "-"
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
				valuationValues[i] = "-" // Default value if no match found
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

func (rc *ReportsController) ParseAccountsReportToPDF(ctx context.Context, accountsReport *schemas.AccountsReports, startDate, endDate time.Time, interval time.Duration) ([]byte, error) {
	excelFile, err := rc.ParseAccountsReportToXLSX(ctx, accountsReport, startDate, endDate, interval)
	if err != nil {
		return nil, err
	}
	logoPath := os.Getenv("LOGO_PATH")
	return ParseExcelToPDFBuffer(excelFile, "Reporte Tenencia y Rendimientos", logoPath)
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

func convertReportDataframeToExcel(file *excelize.File, reportDf *dataframe.DataFrame, sheetName string) (*excelize.File, error) {
	// Create a new Excel file
	var err error
	var index int
	f := file

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

	// Now fill in the data for the rest of the rows starting from the third row
	for rowIndex, row := range reportDf.Records()[1:] { // Skip the first row (headers)
		for colIndex, cellValue := range row {
			cell := fmt.Sprintf("%s%d", toAlphaString(colIndex+1), rowIndex+3) // colIndex+1 to skip DateRequested
			err = f.SetCellValue(sheetName, cell, cellValue)
			if err != nil {
				return nil, err
			}
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
			if existingVal == "-" {
				existingFloat = 0.0
			} else {
				existingFloat, err = strconv.ParseFloat(existingVal, 64)
				if err != nil {
					return nil, err
				}
			}
			if newValues[i] == "-" {
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

func ParseExcelToPDFBuffer(excelFile *excelize.File, title, logoPath string) ([]byte, error) {
	// Create a new PDF document with landscape orientation
	pdf := gofpdf.New("L", "mm", "A4", "")
	pdf.SetFont("Arial", "", 10) // Smaller font size to fit more content

	// Add a cover page with the logo and title
	pdf.AddPage()

	// Add logo
	if logoPath != "" {
		// Set larger dimensions for the logo
		imageWidth := 180.0
		imageHeight := 50.0
		pageWidth, pageHeight := pdf.GetPageSize()

		// Center the image horizontally
		imageX := (pageWidth - imageWidth) / 2
		imageY := (pageHeight / 2) - (imageHeight / 2)

		pdf.ImageOptions(logoPath, imageX, imageY, imageWidth, imageHeight, false, gofpdf.ImageOptions{ImageType: "PNG"}, 0, "")
	}

	pdf.Ln(20) // Space after the logo

	pdf.SetFont("Arial", "B", 16)
	pdf.CellFormat(0, 20, title, "", 1, "C", false, 0, "")
	pdf.Ln(10)

	// Set the page width, height, and a padding margin
	pageWidth, _ := pdf.GetPageSize()
	margin := 10.0
	usableWidth := pageWidth - 2*margin

	// Iterate through each sheet in the Excel file
	sheets := excelFile.GetSheetList()
	for _, sheet := range sheets {
		// Add a new page for each sheet
		pdf.AddPage()
		pdf.SetLeftMargin(margin)
		// Set font back to regular for sheet content
		pdf.SetFont("Arial", "", 10)
		pdf.CellFormat(0, 15, sheet, "", 1, "C", false, 0, "")
		pdf.Ln(4) // Smaller line spacing
		// Set font back to regular for sheet content
		pdf.SetFont("Arial", "", 5)

		// Get all rows and columns in the current sheet
		rows, err := excelFile.GetRows(sheet)
		if err != nil {
			return nil, err
		}

		// Determine the number of columns in the widest row
		maxCols := 0
		for _, row := range rows {
			if len(row) > maxCols {
				maxCols = len(row)
			}
		}

		// Calculate dynamic cell width based on the max number of columns
		cellWidth := usableWidth / float64(maxCols)

		// Get merged cells for the sheet
		mergedCells, err := excelFile.GetMergeCells(sheet)
		if err != nil {
			return nil, err
		}
		mergedCellWidths := map[string]float64{}
		mergedCellsStart := map[string]int{}

		// Populate merged cell widths based on the number of columns spanned
		for _, mc := range mergedCells {
			startCol, _, _ := excelize.CellNameToCoordinates(mc.GetStartAxis())
			endCol, _, _ := excelize.CellNameToCoordinates(mc.GetEndAxis())
			numCols := endCol - startCol + 1
			mergedCellWidths[mc.GetStartAxis()] = cellWidth * float64(numCols)
			mergedCellsStart[mc.GetStartAxis()] = numCols - 1 // Adjust by -1 to account for the cell itself
		}

		// Print each row to the PDF
		for rowIndex, row := range rows {
			positionWidth := margin
			columnsToSkip := 0
			for colIndex, cell := range row {
				if columnsToSkip > 0 {
					columnsToSkip--
					continue
				}
				cellRef, _ := excelize.CoordinatesToCellName(colIndex+1, rowIndex+1)

				// Determine width based on whether it is merged
				cellWidthToUse := cellWidth
				if width, ok := mergedCellWidths[cellRef]; ok {
					cellWidthToUse = width
				}
				// Move to the correct position based on current column
				pdf.SetX(positionWidth)

				// Write cell to the PDF with smaller row height
				var parsedValue string
				if sheet == "Tenencia" {
					parsedValue = formatMonetaryValue(cell)
				} else if sheet == "Retorno" {
					parsedValue = formatPercentageValue(cell)
				} else {
					parsedValue = cell
				}
				pdf.CellFormat(cellWidthToUse, 6, parsedValue, "1", 0, "L", false, 0, "")
				positionWidth += cellWidthToUse
				columnsToSkip += int(cellWidthToUse/cellWidth) - 1
			}
			pdf.Ln(6) // Smaller row height
		}
	}

	// Write PDF output to an in-memory buffer
	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		return nil, fmt.Errorf("could not generate PDF: %w", err)
	}

	return buf.Bytes(), nil
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
		return "-"
	}
	return fmt.Sprintf("%.2f%%", value)
}

// ==================================================================//
// GetAllReportSchedules loads all report schedules and schedules them
func (rc *ReportsController) GetAllReportSchedules(ctx context.Context) ([]*schemas.ReportScheduleResponse, error) {
	var reportSchedules []*models.ReportSchedule
	if err := rc.DB.WithContext(ctx).Find(&reportSchedules).Error; err != nil {
		return nil, err
	}

	var responses []*schemas.ReportScheduleResponse
	for _, rs := range reportSchedules {
		responses = append(responses, &schemas.ReportScheduleResponse{
			ID:                      rs.ID,
			SenderID:                rs.SenderID,
			RecipientOrganizationID: rs.RecipientOrganizationID,
			ReportTemplateID:        rs.ReportTemplateID,
			CronTime:                rs.CronTime,
			LastSentAt:              rs.LastSentAt,
			CreatedAt:               rs.CreatedAt,
			UpdatedAt:               rs.UpdatedAt,
			Active:                  rs.Active,
		})
	}

	return responses, nil
}

// GetReportScheduleByID loads a report schedule by ID and schedules it
func (rc *ReportsController) GetReportScheduleByID(ctx context.Context, ID uint) (*schemas.ReportScheduleResponse, error) {
	var reportSchedule models.ReportSchedule
	if err := rc.DB.WithContext(ctx).First(&reportSchedule, "id = ?", ID).Error; err != nil {
		return nil, err
	}

	response := &schemas.ReportScheduleResponse{
		ID:                      reportSchedule.ID,
		SenderID:                reportSchedule.SenderID,
		RecipientOrganizationID: reportSchedule.RecipientOrganizationID,
		ReportTemplateID:        reportSchedule.ReportTemplateID,
		CronTime:                reportSchedule.CronTime,
		LastSentAt:              reportSchedule.LastSentAt,
		CreatedAt:               reportSchedule.CreatedAt,
		UpdatedAt:               reportSchedule.UpdatedAt,
		Active:                  reportSchedule.Active,
	}

	return response, nil
}

func (rc *ReportsController) CreateReportSchedule(ctx context.Context, req *schemas.CreateReportScheduleRequest) (*schemas.ReportScheduleResponse, error) {
	reportSchedule := models.ReportSchedule{
		SenderID:                req.SenderID,
		RecipientOrganizationID: req.RecipientOrganizationID,
		ReportTemplateID:        req.ReportTemplateID,
		CronTime:                req.CronTime,
	}

	if err := rc.DB.WithContext(ctx).Create(&reportSchedule).Error; err != nil {
		return nil, err
	}

	response := &schemas.ReportScheduleResponse{
		ID:                      reportSchedule.ID,
		SenderID:                reportSchedule.SenderID,
		RecipientOrganizationID: reportSchedule.RecipientOrganizationID,
		ReportTemplateID:        reportSchedule.ReportTemplateID,
		CronTime:                reportSchedule.CronTime,
		LastSentAt:              reportSchedule.LastSentAt,
		CreatedAt:               reportSchedule.CreatedAt,
		UpdatedAt:               reportSchedule.UpdatedAt,
		Active:                  reportSchedule.Active,
	}

	return response, nil
}

func (rc *ReportsController) UpdateReportSchedule(ctx context.Context, req *schemas.UpdateReportScheduleRequest) (*schemas.ReportScheduleResponse, error) {
	var reportSchedule models.ReportSchedule
	if err := rc.DB.WithContext(ctx).First(&reportSchedule, "id = ?", req.ID).Error; err != nil {
		return nil, err
	}

	// Update fields only if they are provided
	if req.SenderID != nil {
		reportSchedule.SenderID = *req.SenderID
	}
	if req.RecipientOrganizationID != nil {
		reportSchedule.RecipientOrganizationID = *req.RecipientOrganizationID
	}
	if req.ReportTemplateID != nil {
		reportSchedule.ReportTemplateID = *req.ReportTemplateID
	}
	if req.CronTime != nil {
		reportSchedule.CronTime = *req.CronTime
	}
	if req.Active != nil {
		reportSchedule.Active = *req.Active
	}

	if err := rc.DB.WithContext(ctx).Save(&reportSchedule).Error; err != nil {
		return nil, err
	}

	response := &schemas.ReportScheduleResponse{
		ID:                      reportSchedule.ID,
		SenderID:                reportSchedule.SenderID,
		RecipientOrganizationID: reportSchedule.RecipientOrganizationID,
		ReportTemplateID:        reportSchedule.ReportTemplateID,
		CronTime:                reportSchedule.CronTime,
		LastSentAt:              reportSchedule.LastSentAt,
		CreatedAt:               reportSchedule.CreatedAt,
		UpdatedAt:               reportSchedule.UpdatedAt,
		Active:                  reportSchedule.Active,
	}

	return response, nil
}

func (rc *ReportsController) DeleteReportSchedule(ctx context.Context, id uint) error {
	if err := rc.DB.WithContext(ctx).Delete(&models.ReportSchedule{}, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return gorm.ErrRecordNotFound
		}
		return err
	}
	return nil
}
