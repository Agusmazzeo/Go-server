package controllers_test

import (
	"context"
	"os"
	"reflect"
	"server/src/models"
	"server/src/schemas"
	"server/src/services"
	"server/src/utils"
	"time"

	"testing"
)

// TestGenerateAccountReports verifies the report generation consistency.
func TestGenerateAccountReports(t *testing.T) {
	const inputFile = "../../test_files/controllers/reports/assets_by_category_1w.json"
	const outputFile = "../../test_files/controllers/reports/assets_return_by_category_1w.json"

	// Load the input data
	var accountData schemas.AccountStateByCategory
	err := utils.LoadStructFromJSONFile(inputFile, &accountData)
	if err != nil {
		t.Fatalf("Failed to load input data: %v", err)
	}

	for i := 0; i < 5; i++ {
		// Generate the report using the service
		reportService := services.NewReportService()
		accountReport, err := reportService.GenerateReport(
			context.Background(),
			&accountData,
			time.Date(2024, 5, 3, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 6, 3, 0, 0, 0, 0, time.UTC),
			24*time.Hour)
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
	if !compareAssetsByCategory(t, *newReport.AssetsByCategory, *savedReport.AssetsByCategory) {
		return false
	}

	if !compareAssetsReturnByCategory(t, *newReport.AssetsReturnByCategory, *savedReport.AssetsReturnByCategory) {
		return false
	}

	return true
}

func compareAssetsByCategory(t *testing.T, v1, v2 map[string][]schemas.Asset) bool {
	t.Helper()
	if len(v1) != len(v2) {
		return false
	}

	for category, assets1 := range v1 {
		assets2, exists := v2[category]
		if !exists || !reflect.DeepEqual(assets1, assets2) {
			return false
		}
	}

	return true
}

func compareAssetsReturnByCategory(t *testing.T, v1, v2 map[string][]schemas.AssetReturn) bool {
	t.Helper()
	if len(v1) != len(v2) {
		return false
	}

	for category, assetReturns1 := range v1 {
		assetReturns2, exists := v2[category]
		if !exists || !compareAssetReturns(t, assetReturns1, assetReturns2) {
			return false
		}
	}

	return true
}

func compareAssetReturns(t *testing.T, v1, v2 []schemas.AssetReturn) bool {
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

func TestCalculateHoldingsReturn(t *testing.T) {
	// Read saved response from file
	var totalHoldingsByDate []schemas.Holding
	err := utils.LoadStructFromJSONFile("../../test_files/controllers/reports/total_holdings_by_date.json", &totalHoldingsByDate)
	if err != nil {
		t.Fatalf("error loading file")
	}

	var totalTransactionsByDate []schemas.Transaction
	err = utils.LoadStructFromJSONFile("../../test_files/controllers/reports/total_transactions_by_date.json", &totalTransactionsByDate)
	if err != nil {
		t.Fatalf("error loading file")
	}

	reportService := services.NewReportService()
	totalReturns := reportService.CalculateHoldingsReturn(totalHoldingsByDate, totalTransactionsByDate, 24*time.Hour, false)

	for _, total := range totalReturns {
		if total.ReturnPercentage > 200 {
			t.Errorf("expected return to not be higher than 50 but got %2.f", total.ReturnPercentage)
		}
	}

}

func TestCreateReportSchedule(t *testing.T) {

	req := &schemas.CreateReportScheduleRequest{
		SenderID:                1,
		RecipientOrganizationID: 2,
		ReportTemplateID:        3,
		CronTime:                "0 0 * * *",
	}

	ctx := context.Background()
	resp, err := reportsScheduleController.CreateReportSchedule(ctx, req)

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
	var err error

	// Create some test data
	_, err = testDB.Exec(ctx,
		"INSERT INTO report_schedules (sender_id, recipient_organization_id, report_template_id, cron_time, active, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, NOW(), NOW())",
		1, 2, 3, "0 0 * * *", true)
	if err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}
	_, err = testDB.Exec(ctx,
		"INSERT INTO report_schedules (sender_id, recipient_organization_id, report_template_id, cron_time, active, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, NOW(), NOW())",
		4, 5, 6, "0 0 * * *", true)
	if err != nil {
		t.Fatalf("Failed to create test data: %v", err)
	}

	resp, err := reportsScheduleController.GetAllReportSchedules(ctx)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(resp) < 2 {
		t.Fatalf("Expected 2 report schedules, got %d", len(resp))
	}
}

func TestGetReportScheduleByID(t *testing.T) {
	ctx := context.Background()
	var err error
	var rs models.ReportSchedule

	// Create a test record
	err = testDB.QueryRow(ctx,
		"INSERT INTO report_schedules (sender_id, recipient_organization_id, report_template_id, cron_time, active, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, NOW(), NOW()) RETURNING id",
		1, 2, 3, "0 0 * * *", true).Scan(&rs.ID)
	if err != nil {
		t.Fatalf("Failed to create test record: %v", err)
	}

	resp, err := reportsScheduleController.GetReportScheduleByID(ctx, rs.ID)
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
	ctx := context.Background()
	var err error
	var rs models.ReportSchedule

	// Create a test record
	err = testDB.QueryRow(ctx,
		"INSERT INTO report_schedules (sender_id, recipient_organization_id, report_template_id, cron_time, active, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, NOW(), NOW()) RETURNING id",
		1, 2, 3, "0 0 * * *", true).Scan(&rs.ID)
	if err != nil {
		t.Fatalf("Failed to create test record: %v", err)
	}

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

	resp, err := reportsScheduleController.UpdateReportSchedule(ctx, req)
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
	ctx := context.Background()
	var err error
	var rs models.ReportSchedule

	// Create a test record
	err = testDB.QueryRow(ctx,
		"INSERT INTO report_schedules (sender_id, recipient_organization_id, report_template_id, cron_time, active, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, NOW(), NOW()) RETURNING id",
		1, 2, 3, "0 0 * * *", true).Scan(&rs.ID)
	if err != nil {
		t.Fatalf("Failed to create test record: %v", err)
	}

	err = reportsScheduleController.DeleteReportSchedule(ctx, rs.ID)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	// Verify deletion
	var count int
	err = testDB.QueryRow(ctx, "SELECT COUNT(*) FROM report_schedules WHERE id = $1", rs.ID).Scan(&count)
	if err != nil {
		t.Fatalf("Failed to verify deletion: %v", err)
	}
	if count != 0 {
		t.Fatalf("Expected count 0, got %d", count)
	}
}
