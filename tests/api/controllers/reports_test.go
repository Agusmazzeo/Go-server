package controllers_test

import (
	"context"
	"os"
	"reflect"
	"server/src/api/controllers"
	"server/src/models"
	"server/src/schemas"
	"server/src/utils"

	"testing"
)

// TestGenerateAccountReports verifies the report generation consistency.
func TestGenerateAccountReports(t *testing.T) {
	const inputFile = "../../test_files/controllers/reports/vouchers_by_category_1w.json"
	const outputFile = "../../test_files/controllers/reports/vouchers_return_by_category_1w.json"

	// Load the input data
	var accountData schemas.AccountStateByCategory
	err := utils.LoadStructFromJSONFile(inputFile, &accountData)
	if err != nil {
		t.Fatalf("Failed to load input data: %v", err)
	}

	for i := 0; i < 5; i++ {
		// Generate the report
		accountReport, err := controllers.GenerateAccountReports(&accountData)
		if err != nil {
			t.Fatalf("Failed to generate account report: %v", err)
		}

		// Check if the output file already exists
		if _, err := os.Stat(outputFile); os.IsNotExist(err) {
			// File does not exist, so create it
			err = utils.SaveStructToJSONFile(accountReport, outputFile)
			if err != nil {
				t.Fatalf("Failed to save output report: %v", err)
			}
			t.Logf("Output file created: %s", outputFile)
		} else {
			// File exists, load the saved report and compare
			var savedReport schemas.AccountsReports
			err = utils.LoadStructFromJSONFile(outputFile, &savedReport)
			if err != nil {
				t.Fatalf("Failed to load saved report: %v", err)
			}

			// Compare the newly generated report with the saved one
			if !compareReports(t, accountReport, &savedReport) {
				t.Errorf("Generated report does not match the saved report on iteration %d", i+1)
			} else {
				t.Logf("Generated report matches the saved report on iteration %d", i+1)
			}
		}
	}
}

// compareReports compares two AccountsReports by checking each field and nested struct.
func compareReports(t *testing.T, newReport, savedReport *schemas.AccountsReports) bool {
	t.Helper()
	if !compareVouchersByCategory(t, *newReport.VouchersByCategory, *savedReport.VouchersByCategory) {
		return false
	}

	if !compareVouchersReturnByCategory(t, *newReport.VouchersReturnByCategory, *savedReport.VouchersReturnByCategory) {
		return false
	}

	return true
}

func compareVouchersByCategory(t *testing.T, v1, v2 map[string][]schemas.Voucher) bool {
	t.Helper()
	if len(v1) != len(v2) {
		return false
	}

	for category, vouchers1 := range v1 {
		vouchers2, exists := v2[category]
		if !exists || !reflect.DeepEqual(vouchers1, vouchers2) {
			return false
		}
	}

	return true
}

func compareVouchersReturnByCategory(t *testing.T, v1, v2 map[string][]schemas.VoucherReturn) bool {
	t.Helper()
	if len(v1) != len(v2) {
		return false
	}

	for category, voucherReturns1 := range v1 {
		voucherReturns2, exists := v2[category]
		if !exists || !compareVoucherReturns(t, voucherReturns1, voucherReturns2) {
			return false
		}
	}

	return true
}

func compareVoucherReturns(t *testing.T, v1, v2 []schemas.VoucherReturn) bool {
	t.Helper()
	if len(v1) != len(v2) {
		return false
	}

	for i := range v1 {
		if v1[i].ID != v2[i].ID || v1[i].Type != v2[i].Type || v1[i].Denomination != v2[i].Denomination || v1[i].Category != v2[i].Category {
			return false
		}
		if !compareReturnsByDateRange(t, v1[i].ReturnsByDateRange, v2[i].ReturnsByDateRange) {
			return false
		}
	}

	return true
}

func compareReturnsByDateRange(t *testing.T, r1, r2 []schemas.ReturnByDate) bool {
	t.Helper()
	if len(r1) != len(r2) {
		return false
	}

	for i := range r1 {
		if !r1[i].StartDate.Equal(r2[i].StartDate) ||
			!r1[i].EndDate.Equal(r2[i].EndDate) ||
			r1[i].ReturnPercentage != r2[i].ReturnPercentage {
			return false
		}
	}

	return true
}

func TestCreateReportSchedule(t *testing.T) {

	req := &schemas.CreateReportScheduleRequest{
		SenderID:                1,
		RecipientOrganizationID: 2,
		ReportTemplateID:        3,
		CronTime:                "0 0 * * *",
	}

	ctx := context.Background()
	resp, err := reportsController.CreateReportSchedule(ctx, req)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if resp == nil {
		t.Fatalf("Expected response, got nil")
	}
	if resp.SenderID != req.SenderID {
		t.Errorf("Expected SenderID %d, got %d", req.SenderID, resp.SenderID)
	}
	if resp.RecipientOrganizationID != req.RecipientOrganizationID {
		t.Errorf("Expected RecipientOrganizationID %d, got %d", req.RecipientOrganizationID, resp.RecipientOrganizationID)
	}
	if resp.ReportTemplateID != req.ReportTemplateID {
		t.Errorf("Expected ReportTemplateID %d, got %d", req.ReportTemplateID, resp.ReportTemplateID)
	}
	if resp.CronTime != req.CronTime {
		t.Errorf("Expected CronTime %s, got %s", req.CronTime, resp.CronTime)
	}
}

func TestGetAllReportSchedules(t *testing.T) {
	ctx := context.Background()

	// Create some test data
	testDB.Create(&models.ReportSchedule{SenderID: 1, RecipientOrganizationID: 2, ReportTemplateID: 3, CronTime: "0 0 * * *", Active: true})
	testDB.Create(&models.ReportSchedule{SenderID: 4, RecipientOrganizationID: 5, ReportTemplateID: 6, CronTime: "0 0 * * *", Active: true})

	resp, err := reportsController.GetAllReportSchedules(ctx)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(resp) < 2 {
		t.Fatalf("Expected 2 report schedules, got %d", len(resp))
	}
}

func TestGetReportScheduleByID(t *testing.T) {

	// Create a test record
	rs := &models.ReportSchedule{SenderID: 1, RecipientOrganizationID: 2, ReportTemplateID: 3, CronTime: "0 0 * * *", Active: true}
	testDB.Create(rs)

	ctx := context.Background()
	resp, err := reportsController.GetReportScheduleByID(ctx, rs.ID)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if resp == nil {
		t.Fatalf("Expected response, got nil")
	}
	if resp.ID != rs.ID {
		t.Errorf("Expected ID %d, got %d", rs.ID, resp.ID)
	}
}

func TestUpdateReportSchedule(t *testing.T) {

	// Create a test record
	rs := &models.ReportSchedule{SenderID: 1, RecipientOrganizationID: 2, ReportTemplateID: 3, CronTime: "0 0 * * *", Active: true}
	testDB.Create(rs)

	// Update the record
	req := &schemas.UpdateReportScheduleRequest{
		ID:                      rs.ID,
		SenderID:                new(uint),
		RecipientOrganizationID: new(uint),
		ReportTemplateID:        new(uint),
		CronTime:                new(string),
		Active:                  new(bool),
	}

	*req.SenderID = 10
	*req.RecipientOrganizationID = 20
	*req.ReportTemplateID = 30
	*req.CronTime = "0 1 * * *"
	*req.Active = false

	ctx := context.Background()
	resp, err := reportsController.UpdateReportSchedule(ctx, req)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if resp == nil {
		t.Fatalf("Expected response, got nil")
	}
	if resp.SenderID != *req.SenderID {
		t.Errorf("Expected SenderID %d, got %d", *req.SenderID, resp.SenderID)
	}
	if resp.RecipientOrganizationID != *req.RecipientOrganizationID {
		t.Errorf("Expected RecipientOrganizationID %d, got %d", *req.RecipientOrganizationID, resp.RecipientOrganizationID)
	}
	if resp.ReportTemplateID != *req.ReportTemplateID {
		t.Errorf("Expected ReportTemplateID %d, got %d", *req.ReportTemplateID, resp.ReportTemplateID)
	}
	if resp.CronTime != *req.CronTime {
		t.Errorf("Expected CronTime %s, got %s", *req.CronTime, resp.CronTime)
	}
	if resp.Active != *req.Active {
		t.Errorf("Expected Active %v, got %v", *req.Active, resp.Active)
	}
}

func TestDeleteReportSchedule(t *testing.T) {

	// Create a test record
	rs := &models.ReportSchedule{SenderID: 1, RecipientOrganizationID: 2, ReportTemplateID: 3, CronTime: "0 0 * * *", Active: true}
	testDB.Create(rs)

	ctx := context.Background()
	err := reportsController.DeleteReportSchedule(ctx, rs.ID)

	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify deletion
	var count int64
	testDB.Model(&models.ReportSchedule{}).Where("id = ?", rs.ID).Count(&count)
	if count != 0 {
		t.Fatalf("Expected count 0, got %d", count)
	}
}
