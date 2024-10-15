package controllers_test

import (
	"context"
	"server/src/models"
	"server/src/schemas"

	"testing"
)

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
