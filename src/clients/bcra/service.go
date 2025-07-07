package bcra

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"server/src/config"
	"server/src/utils"
	"server/src/utils/requests"
	"time"
)

type BCRAServiceClientI interface {
	GetVariables(ctx context.Context) (*GetVariablesResponse, error)
	GetVariablesPorFecha(ctx context.Context, id string, startDate, endDate string) (*GetVariablesResponse, error)
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

// GetVariables fetches the v2 endpoints for Variables
func (c *BCRAServiceClient) GetVariables(ctx context.Context) (*GetVariablesResponse, error) {
	cachedResponse, found := c.VariablesCache.Get(time.Now())
	if found {
		return &cachedResponse, nil
	}
	endpoint := fmt.Sprintf("%s/estadisticas/v3.0/monetarias", c.BaseURL)

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
func (c *BCRAServiceClient) GetVariablesPorFecha(ctx context.Context, id string, fechaDesde string, fechaHasta string) (*GetVariablesResponse, error) {
	endpoint := fmt.Sprintf("%s/estadisticas/v3.0/monetarias/%s?desde=%s&hasta=%s", c.BaseURL, id, fechaDesde, fechaHasta)

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
