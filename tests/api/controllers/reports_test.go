package controllers_test

import (
	"context"
	"encoding/json"
	"server/src/api/controllers"
	"server/src/models"
	"server/src/schemas"
	"server/src/utils"

	"testing"
)

func TestGroupVouchersByCategory(t *testing.T) {
	// Read the vouchers.json file
	accountStateBytes, err := utils.ReadResponseFromFile("../../test_files/controllers/reports/vouchers.json")
	if err != nil {
		t.Fatalf("error while reading vouchers.json file: %v", err)
	}

	// Unmarshal the JSON into the AccountState struct
	var accountState *schemas.AccountState
	err = json.Unmarshal(accountStateBytes, &accountState)
	if err != nil {
		t.Fatalf("error while unmarshalling vouchers.json file: %v", err)
	}

	// Group vouchers by category
	accountGroupedByCategory := controllers.GroupVouchersByCategory(accountState)

	// Validation: Ensure categories are grouped correctly
	if accountGroupedByCategory == nil {
		t.Fatal("Expected accountGroupedByCategory to be non-nil")
	}

	// Expected categories based on the provided JSON
	expectedCategories := []string{
		"ARS",
		"BONOS HARD DOLLAR",
		"MONEY MARKET",
		"TASA FIJA",
		"CER",
		"", // Empty category from the JSON data
	}

	// Check if all expected categories are present in the result
	for _, category := range expectedCategories {
		vouchers, exists := (*accountGroupedByCategory.CategoryVouchers)[category]
		if !exists {
			t.Errorf("Expected category %s not found", category)
		} else if len(vouchers) == 0 {
			t.Errorf("Category %s has no vouchers, expected some vouchers", category)
		}
	}

	// Check specific category: "ARS"
	arsVouchers, arsExists := (*accountGroupedByCategory.CategoryVouchers)["ARS"]
	if !arsExists {
		t.Errorf("Category 'ARS' not found")
	} else if len(arsVouchers) != 1 {
		t.Errorf("Expected 1 voucher in 'ARS' category, got %d", len(arsVouchers))
	}

	// Check specific category: "BONOS HARD DOLLAR"
	bhdVouchers, bhdExists := (*accountGroupedByCategory.CategoryVouchers)["BONOS HARD DOLLAR"]
	if !bhdExists {
		t.Errorf("Category 'BONOS HARD DOLLAR' not found")
	} else if len(bhdVouchers) != 1 {
		t.Errorf("Expected 1 voucher in 'BONOS HARD DOLLAR' category, got %d", len(bhdVouchers))
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
	if len(resp) != 2 {
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
