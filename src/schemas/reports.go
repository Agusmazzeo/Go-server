package schemas

import "time"

type ReportScheduleSchema struct {
	ID                      uint      `json:"id"`
	SenderID                uint      `json:"senderId"`
	RecipientOrganizationID uint      `json:"recipientOrganizationId"`
	ReportTemplateID        uint      `json:"reportTemplateId"`
	CronTime                string    `json:"cronTime"`
	LastSentAt              time.Time `json:"lastSentAt"`
	CreatedAt               time.Time `json:"createdAt"`
	UpdatedAt               time.Time `json:"updatedAt"`
	Active                  bool      `json:"active"`
}
