package bcra

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"server/src/config"
	"server/src/utils/requests"
)

type BCRAServiceClientI interface {
	GetDivisas() (*GetDivisasResponse, error)
	GetCotizaciones(fecha string) (*GetCotizacionesResponse, error)
	GetCotizacionesPorMoneda(moneda string, fechaDesde string, fechaHasta string) (*GetCotizacionesByMonedaResponse, error)
}

type BCRAServiceClient struct {
	API     *requests.ExternalAPIService
	BaseURL string
}

// NewClient creates a new instance of BCRAServiceClient
func NewClient(cfg *config.Config) (*BCRAServiceClient, error) {
	api := requests.NewExternalAPIService()
	return &BCRAServiceClient{
		API:     api,
		BaseURL: cfg.ExternalClients.BCRA.BaseURL,
	}, nil
}

// GetDivisas fetches the available currencies (Divisas) from BCRA
func (c *BCRAServiceClient) GetDivisas() (*GetDivisasResponse, error) {
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
func (c *BCRAServiceClient) GetCotizaciones(fecha string) (*GetCotizacionesResponse, error) {
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
func (c *BCRAServiceClient) GetCotizacionesPorMoneda(moneda string, fechaDesde string, fechaHasta string) (*GetCotizacionesByMonedaResponse, error) {
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
