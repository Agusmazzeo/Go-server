package schemas

import (
	"time"
)

type AccountsReports struct {
	VouchersByCategory       *map[string][]Voucher
	VouchersReturnByCategory *map[string][]VoucherReturn
}

type VoucherReturn struct {
	ID                 string
	Type               string
	Denomination       string
	Category           string
	ReturnsByDateRange []ReturnByDate
}

type ReturnByDate struct {
	StartDate        time.Time
	EndDate          time.Time
	ReturnPercentage float64
}

// CreateReportScheduleRequest represents the request schema for creating a new report schedule.
type CreateReportScheduleRequest struct {
	SenderID                uint   `json:"sender_id" validate:"required"`
	RecipientOrganizationID uint   `json:"recipient_organization_id" validate:"required"`
	ReportTemplateID        uint   `json:"report_template_id" validate:"required"`
	CronTime                string `json:"cron_time" validate:"required"`
}

// UpdateReportScheduleRequest represents the request schema for updating an existing report schedule.
type UpdateReportScheduleRequest struct {
	ID                      uint    `json:"id"`
	SenderID                *uint   `json:"sender_id"`
	RecipientOrganizationID *uint   `json:"recipient_organization_id"`
	ReportTemplateID        *uint   `json:"report_template_id"`
	CronTime                *string `json:"cron_time"`
	Active                  *bool   `json:"active"`
}

// ReportScheduleResponse represents the response schema for report schedule data.
type ReportScheduleResponse struct {
	ID                      uint      `json:"id"`
	SenderID                uint      `json:"sender_id"`
	RecipientOrganizationID uint      `json:"recipient_organization_id"`
	ReportTemplateID        uint      `json:"report_template_id"`
	CronTime                string    `json:"cron_time"`
	LastSentAt              time.Time `json:"last_sent_at"`
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`
	Active                  bool      `json:"active"`
}
