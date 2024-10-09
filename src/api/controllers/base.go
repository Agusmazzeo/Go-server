package controllers

import (
	"context"
	"server/src/clients/bcra"
	"server/src/clients/esco"
	"server/src/schemas"
	"time"

	"gorm.io/gorm"
)

type IController interface {
	GetDBClient() *gorm.DB
	GetESCOClient() esco.ESCOServiceClientI

	// Accounts
	GetAllAccounts(ctx context.Context, token, filter string) ([]*schemas.AccountReponse, error)
	GetAccountState(ctx context.Context, token, id string, date time.Time) (*schemas.AccountState, error)
	GetAccountStateDateRange(ctx context.Context, token, id string, startDate, endDate time.Time) (*schemas.AccountState, error)

	//Report Schedules
	GetAllReportSchedules(ctx context.Context) ([]*schemas.ReportScheduleResponse, error)
	GetReportScheduleByID(ctx context.Context, ID uint) (*schemas.ReportScheduleResponse, error)
	CreateReportSchedule(ctx context.Context, req *schemas.CreateReportScheduleRequest) (*schemas.ReportScheduleResponse, error)
	UpdateReportSchedule(ctx context.Context, req *schemas.UpdateReportScheduleRequest) (*schemas.ReportScheduleResponse, error)
	DeleteReportSchedule(ctx context.Context, id uint) error

	// Currencies
	GetAllCurrencies(ctx context.Context) ([]schemas.Currency, error)
	GetCurrencyWithValuationByID(ctx context.Context, id string, date time.Time) (*schemas.CurrencyWithValuationResponse, error)
	GetCurrencyWithValuationDateRangeByID(ctx context.Context, id string, startDate, endDate time.Time) (*schemas.CurrencyWithValuationResponse, error)

	PostToken(ctx context.Context, username, password string) (*schemas.TokenResponse, error)
}

type Controller struct {
	DB         *gorm.DB
	ESCOClient esco.ESCOServiceClientI
	BCRAClient bcra.BCRAServiceClientI
}

func NewController(db *gorm.DB, escoCLient esco.ESCOServiceClientI, bcraClient bcra.BCRAServiceClientI) *Controller {
	return &Controller{DB: db, ESCOClient: escoCLient, BCRAClient: bcraClient}
}

func (c *Controller) GetDBClient() *gorm.DB {
	return c.DB
}

func (c *Controller) GetESCOClient() esco.ESCOServiceClientI {
	return c.ESCOClient
}

func (c *Controller) GetBCRAClient() bcra.BCRAServiceClientI {
	return c.BCRAClient
}
