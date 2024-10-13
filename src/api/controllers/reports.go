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
	"strings"
	"time"

	"github.com/go-gota/gota/dataframe"
	"github.com/go-gota/gota/series"
	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

type ReportsControllerI interface {
	GetXLSXReport(ctx context.Context, accountState *schemas.AccountState, startDate, endDate time.Time, interval time.Duration) (*excelize.File, error)
	GetDataFrameReport(ctx context.Context, accountState *schemas.AccountState, startDate, endDate time.Time, interval time.Duration) (*dataframe.DataFrame, error)
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

func (rc *ReportsController) GetXLSXReport(ctx context.Context, accountState *schemas.AccountState, startDate, endDate time.Time, interval time.Duration) (*excelize.File, error) {
	df, err := rc.GetDataFrameReport(ctx, accountState, startDate, endDate, interval)
	if err != nil {
		return nil, err
	}
	return convertDataframeToExcel(df)
}

// CreateDataFrameWithDatesAndVoucher creates a DataFrame with dates as rows and Voucher IDs as columns
func (rc *ReportsController) GetDataFrameReport(_ context.Context, accountState *schemas.AccountState, startDate, endDate time.Time, interval time.Duration) (*dataframe.DataFrame, error) {
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
	for _, voucher := range *accountState.Vouchers {
		voucherValues := make([]interface{}, len(dates))

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
							voucherValues[i] = holding.Value
						} else {
							voucherValues[i] = "-"
						}
						break
					}
				}
			}
		}

		// Add the new series (column) for this voucher to the DataFrame
		voucherSeries := series.New(voucherValues, series.String, fmt.Sprintf("%s-%s", voucher.Category, voucher.ID))
		df = df.Mutate(voucherSeries)
	}

	return &df, nil
}

func convertDataframeToExcel(df *dataframe.DataFrame) (*excelize.File, error) {
	// Create a new Excel file
	f := excelize.NewFile()
	sheetName := "Tenencia"

	err := f.SetSheetName("Sheet1", sheetName)
	if err != nil {
		return nil, err
	}
	// Define variables for column and row indices
	startRow := 1
	categoryRow := startRow
	idRow := startRow + 1

	// Extract column names from DataFrame
	cols := df.Names()

	// Variables to track categories and column ranges for merging
	categoryStartCol := make(map[string]int)
	categoryEndCol := make(map[string]int)

	// Set the DateRequested in the first column (A) for both the first and second rows
	err = f.SetCellValue(sheetName, "A2", "DateRequested")
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
	for rowIndex, row := range df.Records()[1:] { // Skip the first row (headers)
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
