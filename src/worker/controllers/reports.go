package controllers

import (
	"context"
	"fmt"
	"server/src/models"
	"server/src/scheduler"
	"server/src/tasks"
)

// LoadAllReportSchedule loads all report schedules and schedules them
func (c *Controller) LoadAllReportSchedule(ctx context.Context) error {
	var reportSchedules []*models.ReportSchedule
	if err := c.DB.QueryRow(ctx, "SELECT * FROM report_schedules").Scan(&reportSchedules); err != nil {
		return err
	}

	for _, reportSchedule := range reportSchedules {
		if err := c.ScheduleReport(ctx, reportSchedule, tasks.SendReportByEmail); err != nil {
			return err
		}
	}

	return nil
}

// LoadReportScheduleByID loads a report schedule by ID and schedules it
func (c *Controller) LoadReportScheduleByID(ctx context.Context, ID uint) error {
	var reportSchedule *models.ReportSchedule
	if err := c.DB.QueryRow(ctx, "SELECT * FROM report_schedules WHERE id = $1", ID).Scan(&reportSchedule); err != nil {
		return err
	}
	if err := c.ScheduleReport(ctx, reportSchedule, tasks.SendReportByEmail); err != nil {
		return err
	}

	return nil
}

// scheduleReport handles the scheduling and re-scheduling of report tasks
func (c *Controller) ScheduleReport(_ context.Context, reportSchedule *models.ReportSchedule, taskFunc func(*models.ReportSchedule) error) error {
	// Delete the existing scheduled goroutine
	c.SchedulerMutex.Lock()
	if existingTask, exists := c.Schedulers[reportSchedule.ID]; exists {
		existingTask.Cancel()
		delete(c.Schedulers, reportSchedule.ID)
	}
	c.SchedulerMutex.Unlock()

	// Create a new scheduled goroutine
	newTask, err := scheduler.NewScheduledTask(reportSchedule.CronTime, func() {
		err := taskFunc(reportSchedule)
		if err != nil {
			fmt.Println(err.Error())
		}
	})
	if err != nil {
		return err
	}

	// Add the new task to the scheduler map
	c.SchedulerMutex.Lock()
	c.Schedulers[reportSchedule.ID] = newTask
	c.SchedulerMutex.Unlock()

	return nil
}
