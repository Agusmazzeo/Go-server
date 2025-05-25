package controllers

import (
	"context"
	"server/src/models"
	"server/src/schemas"

	"github.com/jackc/pgx/v5/pgxpool"
	"gorm.io/gorm"
)

type ReportScheduleControllerI interface {
	GetAllReportSchedules(ctx context.Context) ([]*schemas.ReportScheduleResponse, error)
	GetReportScheduleByID(ctx context.Context, ID uint) (*schemas.ReportScheduleResponse, error)
	CreateReportSchedule(ctx context.Context, req *schemas.CreateReportScheduleRequest) (*schemas.ReportScheduleResponse, error)
	UpdateReportSchedule(ctx context.Context, req *schemas.UpdateReportScheduleRequest) (*schemas.ReportScheduleResponse, error)
	DeleteReportSchedule(ctx context.Context, id uint) error
}

type ReportScheduleController struct {
	DB *pgxpool.Pool
}

func NewReportScheduleController(db *pgxpool.Pool) *ReportScheduleController {
	return &ReportScheduleController{DB: db}
}

// GetAllReportSchedules loads all report schedules and schedules them
func (rc *ReportScheduleController) GetAllReportSchedules(ctx context.Context) ([]*schemas.ReportScheduleResponse, error) {
	rows, err := rc.DB.Query(ctx, `
		SELECT
			id,
			sender_id,
			recipient_organization_id,
			report_template_id,
			cron_time,
			COALESCE(last_sent_at, '1970-01-01'::timestamp) as last_sent_at,
			COALESCE(created_at, NOW()) as created_at,
			COALESCE(updated_at, NOW()) as updated_at,
			active
		FROM report_schedules`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var responses []*schemas.ReportScheduleResponse
	for rows.Next() {
		var rs models.ReportSchedule
		err := rows.Scan(
			&rs.ID,
			&rs.SenderID,
			&rs.RecipientOrganizationID,
			&rs.ReportTemplateID,
			&rs.CronTime,
			&rs.LastSentAt,
			&rs.CreatedAt,
			&rs.UpdatedAt,
			&rs.Active,
		)
		if err != nil {
			return nil, err
		}
		responses = append(responses, &schemas.ReportScheduleResponse{
			ID:                      rs.ID,
			SenderID:                rs.SenderID,
			RecipientOrganizationID: rs.RecipientOrganizationID,
			ReportTemplateID:        rs.ReportTemplateID,
			CronTime:                rs.CronTime,
			LastSentAt:              rs.LastSentAt,
			CreatedAt:               rs.CreatedAt,
			UpdatedAt:               rs.UpdatedAt,
			Active:                  rs.Active,
		})
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return responses, nil
}

// GetReportScheduleByID loads a report schedule by ID and schedules it
func (rc *ReportScheduleController) GetReportScheduleByID(ctx context.Context, ID uint) (*schemas.ReportScheduleResponse, error) {
	var reportSchedule models.ReportSchedule
	err := rc.DB.QueryRow(ctx, `
		SELECT
			id,
			sender_id,
			recipient_organization_id,
			report_template_id,
			cron_time,
			COALESCE(last_sent_at, '1970-01-01'::timestamp) as last_sent_at,
			COALESCE(created_at, NOW()) as created_at,
			COALESCE(updated_at, NOW()) as updated_at,
			active
		FROM report_schedules WHERE id = $1`, ID).Scan(
		&reportSchedule.ID,
		&reportSchedule.SenderID,
		&reportSchedule.RecipientOrganizationID,
		&reportSchedule.ReportTemplateID,
		&reportSchedule.CronTime,
		&reportSchedule.LastSentAt,
		&reportSchedule.CreatedAt,
		&reportSchedule.UpdatedAt,
		&reportSchedule.Active,
	)
	if err != nil {
		return nil, err
	}

	response := &schemas.ReportScheduleResponse{
		ID:                      reportSchedule.ID,
		SenderID:                reportSchedule.SenderID,
		RecipientOrganizationID: reportSchedule.RecipientOrganizationID,
		ReportTemplateID:        reportSchedule.ReportTemplateID,
		CronTime:                reportSchedule.CronTime,
		LastSentAt:              reportSchedule.LastSentAt,
		CreatedAt:               reportSchedule.CreatedAt,
		UpdatedAt:               reportSchedule.UpdatedAt,
		Active:                  reportSchedule.Active,
	}

	return response, nil
}

func (rc *ReportScheduleController) CreateReportSchedule(ctx context.Context, req *schemas.CreateReportScheduleRequest) (*schemas.ReportScheduleResponse, error) {
	reportSchedule := models.ReportSchedule{
		SenderID:                req.SenderID,
		RecipientOrganizationID: req.RecipientOrganizationID,
		ReportTemplateID:        req.ReportTemplateID,
		CronTime:                req.CronTime,
	}

	if err := rc.DB.QueryRow(ctx, "INSERT INTO report_schedules (sender_id, recipient_organization_id, report_template_id, cron_time) VALUES ($1, $2, $3, $4) RETURNING id", reportSchedule.SenderID, reportSchedule.RecipientOrganizationID, reportSchedule.ReportTemplateID, reportSchedule.CronTime).Scan(&reportSchedule.ID); err != nil {
		return nil, err
	}

	response := &schemas.ReportScheduleResponse{
		ID:                      reportSchedule.ID,
		SenderID:                reportSchedule.SenderID,
		RecipientOrganizationID: reportSchedule.RecipientOrganizationID,
		ReportTemplateID:        reportSchedule.ReportTemplateID,
		CronTime:                reportSchedule.CronTime,
		LastSentAt:              reportSchedule.LastSentAt,
		CreatedAt:               reportSchedule.CreatedAt,
		UpdatedAt:               reportSchedule.UpdatedAt,
		Active:                  reportSchedule.Active,
	}

	return response, nil
}

func (rc *ReportScheduleController) UpdateReportSchedule(ctx context.Context, req *schemas.UpdateReportScheduleRequest) (*schemas.ReportScheduleResponse, error) {
	var reportSchedule models.ReportSchedule
	err := rc.DB.QueryRow(ctx, `
		SELECT
			id,
			sender_id,
			recipient_organization_id,
			report_template_id,
			cron_time,
			COALESCE(last_sent_at, '1970-01-01'::timestamp) as last_sent_at,
			COALESCE(created_at, NOW()) as created_at,
			COALESCE(updated_at, NOW()) as updated_at,
			active
		FROM report_schedules WHERE id = $1`, req.ID).Scan(
		&reportSchedule.ID,
		&reportSchedule.SenderID,
		&reportSchedule.RecipientOrganizationID,
		&reportSchedule.ReportTemplateID,
		&reportSchedule.CronTime,
		&reportSchedule.LastSentAt,
		&reportSchedule.CreatedAt,
		&reportSchedule.UpdatedAt,
		&reportSchedule.Active,
	)
	if err != nil {
		return nil, err
	}

	// Update fields only if they are provided
	if req.SenderID != nil {
		reportSchedule.SenderID = *req.SenderID
	}
	if req.RecipientOrganizationID != nil {
		reportSchedule.RecipientOrganizationID = *req.RecipientOrganizationID
	}
	if req.ReportTemplateID != nil {
		reportSchedule.ReportTemplateID = *req.ReportTemplateID
	}
	if req.CronTime != nil {
		reportSchedule.CronTime = *req.CronTime
	}
	if req.Active != nil {
		reportSchedule.Active = *req.Active
	}

	err = rc.DB.QueryRow(ctx, `
		UPDATE report_schedules
		SET
			sender_id = $1,
			recipient_organization_id = $2,
			report_template_id = $3,
			cron_time = $4,
			active = $5,
			updated_at = NOW()
		WHERE id = $6
		RETURNING
			id,
			sender_id,
			recipient_organization_id,
			report_template_id,
			cron_time,
			COALESCE(last_sent_at, '1970-01-01'::timestamp) as last_sent_at,
			COALESCE(created_at, NOW()) as created_at,
			COALESCE(updated_at, NOW()) as updated_at,
			active`,
		reportSchedule.SenderID,
		reportSchedule.RecipientOrganizationID,
		reportSchedule.ReportTemplateID,
		reportSchedule.CronTime,
		reportSchedule.Active,
		reportSchedule.ID,
	).Scan(
		&reportSchedule.ID,
		&reportSchedule.SenderID,
		&reportSchedule.RecipientOrganizationID,
		&reportSchedule.ReportTemplateID,
		&reportSchedule.CronTime,
		&reportSchedule.LastSentAt,
		&reportSchedule.CreatedAt,
		&reportSchedule.UpdatedAt,
		&reportSchedule.Active,
	)
	if err != nil {
		return nil, err
	}

	response := &schemas.ReportScheduleResponse{
		ID:                      reportSchedule.ID,
		SenderID:                reportSchedule.SenderID,
		RecipientOrganizationID: reportSchedule.RecipientOrganizationID,
		ReportTemplateID:        reportSchedule.ReportTemplateID,
		CronTime:                reportSchedule.CronTime,
		LastSentAt:              reportSchedule.LastSentAt,
		CreatedAt:               reportSchedule.CreatedAt,
		UpdatedAt:               reportSchedule.UpdatedAt,
		Active:                  reportSchedule.Active,
	}

	return response, nil
}

func (rc *ReportScheduleController) DeleteReportSchedule(ctx context.Context, id uint) error {
	// First check if the record exists
	var exists bool
	err := rc.DB.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM report_schedules WHERE id = $1)", id).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return gorm.ErrRecordNotFound
	}

	// If record exists, delete it
	_, err = rc.DB.Exec(ctx, "DELETE FROM report_schedules WHERE id = $1", id)
	return err
}
