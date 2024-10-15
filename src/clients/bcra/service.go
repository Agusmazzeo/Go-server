package bcra

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"server/src/config"
	"server/src/utils"
	"server/src/utils/requests"
	"time"
)

type BCRAServiceClientI interface {
	GetDivisas(ctx context.Context) (*GetDivisasResponse, error)
	GetCotizaciones(ctx context.Context, fecha string) (*GetCotizacionesResponse, error)
	GetCotizacionesPorMoneda(ctx context.Context, moneda string, fechaDesde string, fechaHasta string) (*GetCotizacionesByMonedaResponse, error)
	GetVariables(ctx context.Context) (*GetVariablesResponse, error)
	GetVariablePorFecha(ctx context.Context, id string, startDate, endDate string) (*GetVariablesResponse, error)
}

type BCRAServiceClient struct {
	API            *requests.ExternalAPIService
	BaseURL        string
	VariablesCache *utils.Cache[GetVariablesResponse]
}

// NewClient creates a new instance of BCRAServiceClient
func NewClient(cfg *config.Config) (*BCRAServiceClient, error) {
	api := requests.NewExternalAPIService(&tls.Config{InsecureSkipVerify: true})
	cache := utils.NewCache[GetVariablesResponse]()
	return &BCRAServiceClient{
		API:            api,
		BaseURL:        cfg.ExternalClients.BCRA.BaseURL,
		VariablesCache: cache,
	}, nil
}

// GetDivisas fetches the available currencies (Divisas) from BCRA
func (c *BCRAServiceClient) GetDivisas(ctx context.Context) (*GetDivisasResponse, error) {
	endpoint := fmt.Sprintf("%s/estadisticascambiarias/v1.0/Maestros/Divisas", c.BaseURL)

	// Make the GET request
	resp, err := c.API.Get(endpoint, "", nil)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)

	// Save the response and get the response bytes for further processing
	// responseBody, err := utils.SaveResponseToFile(resp.Body, "divisas_response.json")
	if err != nil {
		return nil, err
	}

	// Parse the response into the GetDivisasResponse schema
	var divisasResponse GetDivisasResponse
	err = json.Unmarshal(responseBody, &divisasResponse)
	if err != nil {
		return nil, err
	}

	return &divisasResponse, nil
}

// GetCotizaciones fetches the currency rates (Cotizaciones) for a specific date
func (c *BCRAServiceClient) GetCotizaciones(ctx context.Context, fecha string) (*GetCotizacionesResponse, error) {
	endpoint := fmt.Sprintf("%s/estadisticascambiarias/v1.0/Cotizaciones", c.BaseURL)

	// Add query parameters for the date
	params := url.Values{}
	params.Add("fecha", fecha)

	// Make the GET request
	resp, err := c.API.Get(endpoint, "", params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Save the response and get the response bytes for further processing
	// responseBody, err := utils.SaveResponseToFile(resp.Body, "divisas_by_date_response.json")
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	// Parse the response into the GetCotizacionesResponse schema
	var cotizacionesResponse GetCotizacionesResponse
	err = json.Unmarshal(responseBody, &cotizacionesResponse)
	if err != nil {
		return nil, err
	}

	return &cotizacionesResponse, nil
}

// GetCotizacionesPorMoneda fetches the currency rates for a specific currency (e.g., USD) and date range
func (c *BCRAServiceClient) GetCotizacionesPorMoneda(ctx context.Context, moneda string, fechaDesde string, fechaHasta string) (*GetCotizacionesByMonedaResponse, error) {
	endpoint := fmt.Sprintf("%s/estadisticascambiarias/v1.0/Cotizaciones/%s", c.BaseURL, moneda)

	// Add query parameters for the date range
	params := url.Values{}
	params.Add("fechadesde", fechaDesde)
	params.Add("fechahasta", fechaHasta)

	// Make the GET request
	resp, err := c.API.Get(endpoint, "", params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	// Save the response and get the response bytes for further processing
	// responseBody, err := utils.SaveResponseToFile(resp.Body, fmt.Sprintf("%s_%s_%s_response.json", moneda, fechaDesde, fechaHasta))
	if err != nil {
		return nil, err
	}

	// Parse the response into the GetCotizacionesByMonedaResponse schema
	var cotizacionesByMonedaResponse GetCotizacionesByMonedaResponse
	err = json.Unmarshal(responseBody, &cotizacionesByMonedaResponse)
	if err != nil {
		return nil, err
	}

	return &cotizacionesByMonedaResponse, nil
}

// GetVariables fetches the v2 endpoints for Variables
func (c *BCRAServiceClient) GetVariables(ctx context.Context) (*GetVariablesResponse, error) {
	cachedResponse, found := c.VariablesCache.Get(time.Now())
	if found {
		return &cachedResponse, nil
	}
	endpoint := fmt.Sprintf("%s/estadisticas/v2.0/PrincipalesVariables", c.BaseURL)

	// Make the GET request
	resp, err := c.API.Get(endpoint, "", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Save the response and get the response bytes for further processing
	// responseBody, err := utils.SaveResponseToFile(resp.Body, "variables_response.json")
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	// Parse the response into the GetVariablesResponse schema
	var variablesResponse GetVariablesResponse
	err = json.Unmarshal(responseBody, &variablesResponse)
	if err != nil {
		return nil, err
	}
	c.VariablesCache.Set(variablesResponse, 24*time.Hour)
	return &variablesResponse, nil
}

// GetCotizacionesPorMoneda fetches the currency rates for a specific currency (e.g., USD) and date range
func (c *BCRAServiceClient) GetVariablePorFecha(ctx context.Context, id string, fechaDesde string, fechaHasta string) (*GetVariablesResponse, error) {
	endpoint := fmt.Sprintf("%s/estadisticas/v2.0/DatosVariable/%s/%s/%s", c.BaseURL, id, fechaDesde, fechaHasta)

	// Make the GET request
	resp, err := c.API.Get(endpoint, "", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	// Save the response and get the response bytes for further processing
	// responseBody, err := utils.SaveResponseToFile(resp.Body, fmt.Sprintf("%s_%s_%s_response.json", id, fechaDesde, fechaHasta))
	if err != nil {
		return nil, err
	}

	var variablesPorFecha GetVariablesResponse
	err = json.Unmarshal(responseBody, &variablesPorFecha)
	if err != nil {
		return nil, err
	}

	return &variablesPorFecha, nil
}
