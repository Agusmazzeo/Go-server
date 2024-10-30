package controllers

import (
	"context"
	"errors"
	"fmt"
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
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

type ReportsControllerI interface {
	GetReport(ctx context.Context, accountsStates *schemas.AccountStateByCategory, variablesWithValuations []*schemas.VariableWithValuationResponse, startDate, endDate time.Time, interval time.Duration) (*schemas.AccountsReports, error)
	GetXLSXReport(ctx context.Context, accountsStates []*schemas.AccountState, startDate, endDate time.Time, interval time.Duration) (*excelize.File, error)
	GetDataFrameReport(ctx context.Context, accountsStates []*schemas.AccountState, startDate, endDate time.Time, interval time.Duration) (*dataframe.DataFrame, error)
	GetReportScheduleByID(ctx context.Context, ID uint) (*schemas.ReportScheduleResponse, error)
	GetAllReportSchedules(ctx context.Context) ([]*schemas.ReportScheduleResponse, error)
	CreateReportSchedule(ctx context.Context, req *schemas.CreateReportScheduleRequest) (*schemas.ReportScheduleResponse, error)
	UpdateReportSchedule(ctx context.Context, req *schemas.UpdateReportScheduleRequest) (*schemas.ReportScheduleResponse, error)
	DeleteReportSchedule(ctx context.Context, id uint) error
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
	return GenerateAccountReports(accountsStates)
}

func (rc *ReportsController) GetXLSXReport(ctx context.Context, accountsStates []*schemas.AccountState, startDate, endDate time.Time, interval time.Duration) (*excelize.File, error) {
	reportDf, err := rc.GetDataFrameReport(ctx, accountsStates, startDate, endDate, interval)
	if err != nil {
		return nil, err
	}
	file, err := convertReportDataframeToExcel(nil, reportDf)
	if err != nil {
		return nil, err
	}
	variationDf := rc.GetDataFramePercentageVariation(reportDf)
	file, err = addVariationDataFrameToExcel(file, variationDf)
	if err != nil {
		return nil, err
	}
	err = applyStylesToAllSheets(file)
	if err != nil {
		return nil, err
	}
	return file, nil
}

// CreateDataFrameWithDatesAndVoucher creates a DataFrame with dates as rows and Voucher IDs as columns
func (rc *ReportsController) GetDataFrameReport(_ context.Context, accountsStates []*schemas.AccountState, startDate, endDate time.Time, interval time.Duration) (*dataframe.DataFrame, error) {
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
	for _, accountState := range accountsStates {
		for _, voucher := range *accountState.Vouchers {
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

	return sortDataFrameColumns(&df), nil
}

func (rc *ReportsController) GetDataFramePercentageVariation(df *dataframe.DataFrame) *dataframe.DataFrame {
	// Apply the percentage change calculation to all columns in parallel using Capply
	result := df.Capply(func(s series.Series) series.Series {
		// For the DateRequested column, return the same series
		if s.Name == "DateRequested" {
			return s
		}

		// Create a new slice for percentage changes
		newValues := make([]float64, s.Len())
		newValues[0] = 0 // No percentage change for the first row

		// Calculate the percentage variation for each row
		for i := 1; i < s.Len(); i++ {
			currentValue, err1 := strconv.ParseFloat(s.Elem(i).String(), 64)
			previousValue, err2 := strconv.ParseFloat(s.Elem(i-1).String(), 64)

			// If parsing fails or previous value is zero, set the change to 0
			if err1 != nil || err2 != nil || previousValue == 0 {
				newValues[i] = 0
			} else {
				newValues[i] = ((currentValue - previousValue) / previousValue) * 100
			}
		}

		// Return the new series with percentage changes
		return series.Floats(newValues)
	})

	return &result
}

func convertReportDataframeToExcel(file *excelize.File, reportDf *dataframe.DataFrame) (*excelize.File, error) {
	// Create a new Excel file
	var err error
	var index int
	f := file
	sheetName := "Tenencia"

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
		parts := strings.Split(col, "-")
		category := parts[0]
		id := parts[1]

		// Set the ID in the second row (e.g., ID1, ID2, etc.)
		cell := fmt.Sprintf("%s%d", ToAlphaString(columnIndex), idRow)
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
		startCell := fmt.Sprintf("%s%d", ToAlphaString(startCol), categoryRow)
		endCell := fmt.Sprintf("%s%d", ToAlphaString(endCol), categoryRow)
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
			cell := fmt.Sprintf("%s%d", ToAlphaString(colIndex+1), rowIndex+3) // colIndex+1 to skip DateRequested
			err = f.SetCellValue(sheetName, cell, cellValue)
			if err != nil {
				return nil, err
			}
		}
	}

	return f, nil
}

func addVariationDataFrameToExcel(file *excelize.File, variationDf *dataframe.DataFrame) (*excelize.File, error) {
	// Create a new Excel file
	var err error
	var index int
	f := file
	sheetName := "Variacion - Retorno"

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
	cols := variationDf.Names()

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
		parts := strings.Split(col, "-")
		category := parts[0]
		id := parts[1]

		// Set the ID in the second row (e.g., ID1, ID2, etc.)
		cell := fmt.Sprintf("%s%d", ToAlphaString(columnIndex), idRow)
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
		startCell := fmt.Sprintf("%s%d", ToAlphaString(startCol), categoryRow)
		endCell := fmt.Sprintf("%s%d", ToAlphaString(endCol), categoryRow)
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
	for rowIndex, row := range variationDf.Records()[1:] { // Skip the first row (headers)
		for colIndex, cellValue := range row {
			cell := fmt.Sprintf("%s%d", ToAlphaString(colIndex+1), rowIndex+3) // colIndex+1 to skip DateRequested
			err = f.SetCellValue(sheetName, cell, cellValue)
			if err != nil {
				return nil, err
			}
		}
	}

	return f, nil
}

// ToAlphaString converts a column index to an Excel column string (e.g., 1 -> A, 2 -> B, 28 -> AB)
func ToAlphaString(column int) string {
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

// findDateRange returns the earliest and latest DateRequested from the holdings.
func findDateRange(holdings []schemas.Holding) (time.Time, time.Time, error) {
	if len(holdings) == 0 {
		return time.Time{}, time.Time{}, fmt.Errorf("no holdings found")
	}

	// Initialize with the first holding's date
	startDate := *holdings[0].DateRequested
	endDate := *holdings[0].DateRequested

	// Loop through holdings to find the earliest and latest DateRequested
	for _, holding := range holdings {
		if holding.DateRequested != nil {
			if holding.DateRequested.Before(startDate) {
				startDate = *holding.DateRequested
			}
			if holding.DateRequested.After(endDate) {
				endDate = *holding.DateRequested
			}
		}
	}

	return startDate, endDate, nil
}

// CalculateVoucherReturn calculates the return for a single voucher by taking holdings in pairs and applying transactions within the date ranges.
func CalculateVoucherReturn(voucher schemas.Voucher) (schemas.VoucherReturn, error) {
	if len(voucher.Holdings) < 2 {
		return schemas.VoucherReturn{}, fmt.Errorf("insufficient holdings data to calculate return for voucher %s", voucher.ID)
	}

	// Sort holdings by date
	sortedHoldings := sortHoldingsByDate(voucher.Holdings)
	var returnsByDateRange []schemas.ReturnByDate

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
		for _, transaction := range voucher.Transactions {
			if transaction.Date != nil && (transaction.Date.After(startDate) && !transaction.Date.After(endDate)) {
				netTransactions -= transaction.Value
			}
		}

		// Calculate return for this date range
		if startingValue == 0 {
			continue
		}

		returnPercentage := ((endingValue - (startingValue + netTransactions)) / (startingValue + netTransactions)) * 100
		// Append the return for this date range
		returnsByDateRange = append(returnsByDateRange, schemas.ReturnByDate{
			StartDate:        startDate,
			EndDate:          endDate,
			ReturnPercentage: returnPercentage,
		})
	}

	// Return the result
	return schemas.VoucherReturn{
		ID:                 voucher.ID,
		Type:               voucher.Type,
		Denomination:       voucher.Denomination,
		Category:           voucher.Category,
		ReturnsByDateRange: returnsByDateRange,
	}, nil
}

// sortHoldingsByDate sorts the holdings by DateRequested.
func sortHoldingsByDate(holdings []schemas.Holding) []schemas.Holding {
	sort.Slice(holdings, func(i, j int) bool {
		return holdings[i].DateRequested.Before(*holdings[j].DateRequested)
	})
	return holdings
}

// GenerateAccountReports calculates the return for each voucher per category and returns an AccountsReports struct.
func GenerateAccountReports(accountStateByCategory *schemas.AccountStateByCategory) (*schemas.AccountsReports, error) {
	voucherReturnsByCategory := make(map[string][]schemas.VoucherReturn)

	// Iterate through each category and its associated vouchers
	for category, vouchers := range *accountStateByCategory.VouchersByCategory {
		if category == "ARS" {
			continue
		}
		for _, voucher := range vouchers {
			voucherReturn, _ := CalculateVoucherReturn(voucher)
			// if err != nil {
			// 	return &schemas.AccountsReports{}, err
			// }
			voucherReturnsByCategory[category] = append(voucherReturnsByCategory[category], voucherReturn)
		}
	}

	return &schemas.AccountsReports{
		VouchersByCategory:       accountStateByCategory.VouchersByCategory,
		VouchersReturnByCategory: &voucherReturnsByCategory,
	}, nil
}

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
