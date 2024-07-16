package models

import (
	"time"
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
