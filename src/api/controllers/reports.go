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
	"time"

	"github.com/xuri/excelize/v2"
	"gorm.io/gorm"
)

type ReportsControllerI interface {
	GenerateXLSX(ctx context.Context, accountState *schemas.AccountState, startDate, endDate time.Time, interval time.Duration) (*excelize.File, error)
	GetAllReportSchedules(ctx context.Context) ([]*schemas.ReportScheduleResponse, error)
	GetReportScheduleByID(ctx context.Context, ID uint) (*schemas.ReportScheduleResponse, error)
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

func (rc *ReportsController) GenerateXLSX(ctx context.Context, accountState *schemas.AccountState, startDate, endDate time.Time, interval time.Duration) (*excelize.File, error) {

	dates, err := utils.GenerateDates(startDate, endDate, interval)
	if err != nil {
		return nil, err
	}
	groupedVouchers := GroupVouchersByCategory(accountState)

	// Create a new Excel file
	f := excelize.NewFile()

	// Create a new sheet
	sheetName := "Tenencia"
	index, err := f.NewSheet(sheetName)
	if err != nil {
		return nil, err
	}

	datesIndex, err := setDateRequestedInFile(f, sheetName, 'A', dates)
	if err != nil {
		return nil, err
	}

	err = setVouchersInFile(f, sheetName, 'B', datesIndex, groupedVouchers)
	if err != nil {
		return nil, err
	}

	// // Collect all unique DateRequested values across all vouchers
	// datesMap := map[string]struct{}{}
	// for _, voucher := range *accountState.Vouchers {
	// 	for _, holding := range voucher.Holdings {
	// 		formattedDate := holding.DateRequested.Format("2006-01-02")
	// 		datesMap[formattedDate] = struct{}{}
	// 	}
	// }

	// // Convert the dates map keys into a sorted slice
	// dates := make([]string, 0, len(datesMap))
	// for date := range datesMap {
	// 	dates = append(dates, date)
	// }
	// sort.Strings(dates)

	// // Write the header row with categories in the first row
	// _ = f.SetCellValue(sheetName, "A2", "Requested Date")
	// col := 2
	// categoryMap := make(map[string][]string)

	// // Collect vouchers under their respective categories
	// for voucherID, voucher := range *accountState.Vouchers {
	// 	categoryMap[voucher.Category] = append(categoryMap[voucher.Category], voucherID)
	// }

	// // Write categories in the first row and voucher IDs in the second row
	// for category, vouchers := range categoryMap {
	// 	startCol := col
	// 	for _, voucherID := range vouchers {
	// 		// Write the voucher ID in the second row
	// 		_ = f.SetCellValue(sheetName, fmt.Sprintf("%s2", string('A'+col-1)), voucherID)
	// 		col++
	// 	}
	// 	// Merge the category header over the voucher columns
	// 	endCol := col - 1
	// 	_ = f.MergeCell(sheetName, fmt.Sprintf("%s1", string('A'+startCol-1)), fmt.Sprintf("%s1", string('A'+endCol-1)))
	// 	_ = f.SetCellValue(sheetName, fmt.Sprintf("%s1", string('A'+startCol-1)), category)
	// }

	// // Write the dates in the first column for each data row
	// for i, date := range dates {
	// 	row := i + 3
	// 	_ = f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), date)
	// }

	// // Now, populate the values by iterating over the vouchers and the holdings
	// col = 2
	// for _, vouchers := range categoryMap {
	// 	for _, voucherID := range vouchers {
	// 		voucher := (*accountState.Vouchers)[voucherID]

	// 		// Iterate over each date and populate values for the corresponding voucher ID
	// 		for i, date := range dates {
	// 			row := i + 3
	// 			valueSet := false

	// 			for _, holding := range voucher.Holdings {
	// 				if holding.DateRequested.Format("2006-01-02") == date {
	// 					if holding.Value < 1.0 {
	// 						_ = f.SetCellValue(sheetName, fmt.Sprintf("%s%d", string('A'+col-1), row), "-")
	// 					} else {
	// 						_ = f.SetCellValue(sheetName, fmt.Sprintf("%s%d", string('A'+col-1), row), holding.Value)
	// 					}
	// 					valueSet = true
	// 					break
	// 				}
	// 			}

	// 			// If no value was set for this date, set it to "-"
	// 			if !valueSet {
	// 				_ = f.SetCellValue(sheetName, fmt.Sprintf("%s%d", string('A'+col-1), row), "-")
	// 			}
	// 		}
	// 		col++
	// 	}
	// }

	// Set the active sheet and return the file
	f.SetActiveSheet(index)
	return f, nil
}

func setDateRequestedInFile(f *excelize.File, sheetName string, column rune, dates []time.Time) (map[string]int, error) {
	var err error
	i := 2
	var dateStr string
	dateIndex := map[string]int{}
	cell := fmt.Sprintf("%c%d", column, i)
	err = f.SetCellStr(sheetName, cell, "Fecha")
	if err != nil {
		return nil, err
	}
	for _, date := range dates {
		i++
		cell = fmt.Sprintf("%c%d", column, i)
		dateStr = date.Format("2006-01-02")
		err = f.SetCellStr(sheetName, cell, dateStr)
		if err != nil {
			return nil, err
		}
		dateIndex[dateStr] = i
	}
	return dateIndex, nil
}

func setVouchersInFile(f *excelize.File, sheetName string, startColumn rune, datesIndex map[string]int, groupedVouchers *schemas.AccountStateByCategory) error {
	var err error
	var column rune
	var cell string
	columnIndex := 0
	for category, vouchers := range *groupedVouchers.CategoryVouchers {
		rowIndex := 1
		column = startColumn + rune(columnIndex)
		cell = fmt.Sprintf("%c%d", column, rowIndex)
		err = f.SetCellStr(sheetName, cell, category)
		if err != nil {
			return err
		}
		if len(vouchers) > 1 {
			columnToMerge := column + rune(len(vouchers)-1)
			cellToMerge := fmt.Sprintf("%c%d", columnToMerge, rowIndex)
			err = f.MergeCell(sheetName, cell, cellToMerge)
			if err != nil {
				return err
			}
		}
		rowIndex++
		for _, voucher := range vouchers {
			cell = fmt.Sprintf("%c%d", column, rowIndex)
			err = f.SetCellStr(sheetName, cell, voucher.ID)
			if err != nil {
				return err
			}
			for _, holding := range voucher.Holdings {
				holdingRowIndex := datesIndex[holding.DateRequested.Format("2006-01-02")]
				cell = fmt.Sprintf("%c%d", column, holdingRowIndex)
				err = f.SetCellFloat(sheetName, cell, holding.Value, 2, 32)
				if err != nil {
					return err
				}
			}
			columnIndex++
			column = startColumn + rune(columnIndex)
		}
	}
	return nil
}

// OrganizeVouchersByCategory takes an AccountState and returns an AccountStateByCategory
// which groups Vouchers by their Category.
func GroupVouchersByCategory(accountState *schemas.AccountState) *schemas.AccountStateByCategory {
	// Initialize a map to store vouchers by category
	categoryVouchers := make(map[string][]schemas.Voucher)

	// Iterate over the vouchers in AccountState and group them by Category
	for _, voucher := range *accountState.Vouchers {
		category := voucher.Category
		categoryVouchers[category] = append(categoryVouchers[category], voucher)
	}

	// Return the result as AccountStateByCategory
	return &schemas.AccountStateByCategory{
		CategoryVouchers: &categoryVouchers,
	}
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
