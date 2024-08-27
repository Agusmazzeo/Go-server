package controllers

import (
	"context"
	"server/src/clients/esco"
	"server/src/schemas"
	"time"

	"gorm.io/gorm"
)

type IController interface {
	GetDBClient() *gorm.DB
	GetESCOClient() *esco.ESCOServiceClient
	GetAllAccounts(ctx context.Context, token, filter string) ([]*schemas.AccountReponse, error)
	GetAccountState(ctx context.Context, token, id string, date time.Time) (*schemas.AccountState, error)
	GetAccountStateDateRange(ctx context.Context, token, id string, startDate, endDate time.Time) (*schemas.AccountState, error)
	GetAllReportSchedules(ctx context.Context) ([]*schemas.ReportScheduleResponse, error)
	GetReportScheduleByID(ctx context.Context, ID uint) (*schemas.ReportScheduleResponse, error)

	PostToken(ctx context.Context, username, password string) (*schemas.TokenResponse, error)
	CreateReportSchedule(ctx context.Context, req *schemas.CreateReportScheduleRequest) (*schemas.ReportScheduleResponse, error)
	UpdateReportSchedule(ctx context.Context, req *schemas.UpdateReportScheduleRequest) (*schemas.ReportScheduleResponse, error)
	DeleteReportSchedule(ctx context.Context, id uint) error
}

type Controller struct {
	DB         *gorm.DB
	ESCOClient *esco.ESCOServiceClient
}

func NewController(db *gorm.DB, escoCLient *esco.ESCOServiceClient) *Controller {
	return &Controller{DB: db, ESCOClient: escoCLient}
}

func (c *Controller) GetDBClient() *gorm.DB {
	return c.DB
}

func (c *Controller) GetESCOClient() *esco.ESCOServiceClient {
	return c.ESCOClient
}
