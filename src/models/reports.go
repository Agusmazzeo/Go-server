package models

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type ReportSchedule struct {
	ID                      uint      `gorm:"primaryKey;column:id"`
	SenderID                uint      `gorm:"column:sender_id"`
	RecipientOrganizationID uint      `gorm:"column:recipient_organization_id"`
	ReportTemplateID        uint      `gorm:"column:report_template_id"`
	CronTime                string    `gorm:"column:cron_time"`
	LastSentAt              time.Time `gorm:"column:last_sent_at"`
	CreatedAt               time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt               time.Time `gorm:"column:updated_at;autoUpdateTime"`
	Active                  bool      `gorm:"column:active"`
}

func (ReportSchedule) TableName() string {
	return "report_schedules"
}

// GetReportScheduleByID fetches a ReportSchedule by its ID
func GetReportScheduleByID(ctx context.Context, db *gorm.DB, id uint) (*ReportSchedule, error) {
	var reportSchedule ReportSchedule
	if err := db.WithContext(ctx).First(&reportSchedule, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &reportSchedule, nil
}

// GetAllReportSchedules fetches all ReportSchedule records
func GetAllReportSchedules(ctx context.Context, db *gorm.DB) ([]ReportSchedule, error) {
	var reportSchedules []ReportSchedule
	if err := db.WithContext(ctx).Find(&reportSchedules).Error; err != nil {
		return nil, err
	}
	return reportSchedules, nil
}
