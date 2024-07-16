package controllers_test

import (
	"context"
	"testing"
	"time"

	"server/src/models"
	"server/src/worker/controllers"
)

var ch = make(chan bool, 1)

// Mock function that sends a value to a channel when called
func mockSendReportByEmail(reportSchedule *models.ReportSchedule) error {
	// Send a value to the channel to indicate the function was called
	ch <- true
	return nil
}

func TestScheduleReport(t *testing.T) {

	c := controllers.NewController(nil)

	reportSchedule := &models.ReportSchedule{
		ID:       1,
		CronTime: "@every 1s", // every second
	}

	// Test scheduling a new task
	err := c.ScheduleReport(context.Background(), reportSchedule, mockSendReportByEmail)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if _, ok := c.Schedulers[reportSchedule.ID]; !ok {
		t.Fatalf("Expected task to be scheduled, but it was not")
	}

	// Test if the task is correctly scheduled and executed
	select {
	case <-ch:
		// Task was called as expected
	case <-time.After(5 * time.Second):
		t.Fatalf("Expected task to be called within 5 seconds, but it was not")
	}

}

func TestScheduleReport_ErrorCreatingTask(t *testing.T) {
	c := controllers.NewController(nil)

	reportSchedule := &models.ReportSchedule{
		ID:       1,
		CronTime: "invalid-cron", // invalid cron time to induce error
	}

	// Test scheduling a new task with an invalid cron expression
	err := c.ScheduleReport(context.Background(), reportSchedule, mockSendReportByEmail)
	if err == nil {
		t.Fatalf("Expected error due to invalid cron expression, got nil")
	}
}
