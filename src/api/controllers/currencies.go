package controllers

import (
	"context"
	"server/src/schemas"
	"time"
)

func (c *Controller) GetAllCurrencies(ctx context.Context) ([]schemas.Currency, error) {
	response, err := c.BCRAClient.GetDivisas(ctx)
	if err != nil {
		return nil, err
	}

	currencies := make([]schemas.Currency, 0, len(response.Results))
	for _, currency := range response.Results {
		currencies = append(currencies, schemas.Currency{ID: currency.Codigo, Description: currency.Denominacion})
	}
	return currencies, nil
}

func (c *Controller) GetCurrencyWithValuationByID(ctx context.Context, id string, date time.Time) (*schemas.CurrencyWithValuationResponse, error) {
	response, err := c.BCRAClient.GetCotizacionesPorMoneda(ctx, id, date.Format("2006-01-02"), date.AddDate(0, 0, 1).Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	if len(response.Results) == 0 {
		return nil, nil
	}
	currencyResponse := &schemas.CurrencyWithValuationResponse{
		ID:          response.Results[0].Detalle[0].CodigoMoneda,
		Description: response.Results[0].Detalle[0].Descripcion,
		Valuations:  make([]schemas.CurrencyValuation, 0, len(response.Results)),
	}
	for _, currency := range response.Results {
		for _, detail := range currency.Detalle {
			currencyResponse.Valuations = append(currencyResponse.Valuations, schemas.CurrencyValuation{
				Date:                currency.Fecha,
				ArgCurrencyRelation: float32(detail.TipoCotizacion),
				UsdCurrencyRelation: float32(detail.TipoPase),
			})
		}

	}
	return currencyResponse, nil
}

func (c *Controller) GetCurrencyWithValuationDateRangeByID(ctx context.Context, id string, startDate, endDate time.Time) (*schemas.CurrencyWithValuationResponse, error) {
	response, err := c.BCRAClient.GetCotizacionesPorMoneda(ctx, id, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	currencyResponse := &schemas.CurrencyWithValuationResponse{
		ID:          response.Results[0].Detalle[0].CodigoMoneda,
		Description: response.Results[0].Detalle[0].Descripcion,
		Valuations:  make([]schemas.CurrencyValuation, 0, len(response.Results)),
	}
	for _, currency := range response.Results {
		for _, detail := range currency.Detalle {
			currencyResponse.Valuations = append(currencyResponse.Valuations, schemas.CurrencyValuation{
				Date:                currency.Fecha,
				ArgCurrencyRelation: float32(detail.TipoCotizacion),
				UsdCurrencyRelation: float32(detail.TipoPase),
			})
		}

	}
	return currencyResponse, nil
}
