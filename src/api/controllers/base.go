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

	// Currencies
	GetAllCurrencies(ctx context.Context) ([]schemas.Currency, error)
	GetCurrencyWithValuationByID(ctx context.Context, id string, date time.Time) (*schemas.CurrencyWithValuationResponse, error)
	GetCurrencyWithValuationDateRangeByID(ctx context.Context, id string, startDate, endDate time.Time) (*schemas.CurrencyWithValuationResponse, error)

	// Variables
	GetAllVariables(ctx context.Context) ([]schemas.Variable, error)
	GetVariableWithValuationByID(ctx context.Context, id string, date time.Time) (*schemas.VariableWithValuationResponse, error)
	GetVariableWithValuationDateRangeByID(ctx context.Context, id string, startDate, endDate time.Time) (*schemas.VariableWithValuationResponse, error)
	GetReferenceVariablesWithValuationDateRange(ctx context.Context, startDate, endDate time.Time, interval time.Duration) (map[string]*schemas.VariableWithValuationResponse, error)

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
