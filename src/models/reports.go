package models

import (
	"time"
)

type ReportSchedule struct {
	ID                      uint      `db:"id"`
	SenderID                uint      `db:"sender_id"`
	RecipientOrganizationID uint      `db:"recipient_organization_id"`
	ReportTemplateID        uint      `db:"report_template_id"`
	CronTime                string    `db:"cron_time"`
	LastSentAt              time.Time `db:"last_sent_at"`
	CreatedAt               time.Time `db:"created_at"`
	UpdatedAt               time.Time `db:"updated_at"`
	Active                  bool      `db:"active"`
}

func (ReportSchedule) TableName() string {
	return "report_schedules"
}
