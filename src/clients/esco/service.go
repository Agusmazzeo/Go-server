package esco

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"server/src/config"
	"server/src/schemas"
	"server/src/utils"
	requests "server/src/utils/requests"
)

type ESCOServiceClientI interface {
	PostToken(_ context.Context, username, password string) (*schemas.TokenResponse, error)
	BuscarCuentas(token, filter string) ([]CuentaSchema, error)
	GetCuentaDetalle(token, cid string) (*CuentaDetalleSchema, error)
	GetEstadoCuenta(token, cid, fid, nncc, tf string, date time.Time) ([]EstadoCuentaSchema, error)
	GetLiquidaciones(token, cid, fid, nncc, tf string, startDate, endDate time.Time) ([]Liquidacion, error)
	GetBoletos(token, cid, fid, nncc, tf string, startDate, endDate time.Time) ([]Boleto, error)
	GetCategoryMap() map[string]string
}

// ESCOServiceClient is a struct that uses ExternalAPIService to interact with the ESCO API
type ESCOServiceClient struct {
	API         *requests.ExternalAPIService
	BaseURL     string
	TokenURL    string
	CategoryMap *map[string]string
}

// NewClient creates a new instance of ESCOServiceClient
func NewClient(cfg *config.Config) (*ESCOServiceClient, error) {
	api := requests.NewExternalAPIService()
	categoryMap, err := utils.CSVToMap(cfg.ExternalClients.ESCO.CategoryMapFile)
	if err != nil {
		return nil, err
	}
	return &ESCOServiceClient{
		API:         api,
		BaseURL:     cfg.ExternalClients.ESCO.BaseURL,
		TokenURL:    cfg.ExternalClients.ESCO.TokenURL,
		CategoryMap: categoryMap,
	}, nil
}

// GetToken retrieves and sets the token for the external service
func (s *ESCOServiceClient) PostToken(_ context.Context, username, password string) (*schemas.TokenResponse, error) {

	data := url.Values{}
	data.Set("grant_type", "password")
	data.Set("username", username)
	data.Set("password", password)
	data.Set("client_id", "Unisync")

	req, err := http.NewRequest("POST", s.TokenURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to retrieve token | Status Code: %d | Response: %v", resp.StatusCode, resp.Body)
	}

	var tokenResponse = new(schemas.TokenResponse)
	// Save the response and get the response bytes for further processing
	// body, err := utils.SaveResponseToFile(resp.Body, "token_response.json")
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &tokenResponse)
	if err != nil {
		return nil, err
	}

	return tokenResponse, nil
}

// BuscarCuentas retrieves all accounts matching filter
func (s *ESCOServiceClient) BuscarCuentas(token, filter string) ([]CuentaSchema, error) {
	body := map[string]string{
		"Filtro": filter,
	}

	headers := map[string]string{}

	resp, err := s.API.PostWithHeaders(s.BaseURL+"/BuscarCuentas", token, body, headers)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result []CuentaSchema

	// Save the response and get the response bytes for further processing
	// responseBody, err := utils.SaveResponseToFile(resp.Body, "cuentas_response.json")
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(responseBody, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// GetCuentaDetalle retrieves detailed account information
func (s *ESCOServiceClient) GetCuentaDetalle(token, cid string) (*CuentaDetalleSchema, error) {
	body := map[string]string{
		"CID_P": cid,
	}

	headers := map[string]string{}

	resp, err := s.API.PostWithHeaders(s.BaseURL+"/GetCuentaDetalle", token, body, headers)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result = new(CuentaDetalleSchema)

	// Save the response and get the response bytes for further processing
	// responseBody, err := utils.SaveResponseToFile(resp.Body, fmt.Sprintf("cuenta_detalle_%s_response.json", cid))
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(responseBody, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// GetEstadoCuenta retrieves the account status information
func (s *ESCOServiceClient) GetEstadoCuenta(token, cid, fid, nncc, tf string, date time.Time) ([]EstadoCuentaSchema, error) {
	// tf is filter by concertacion (-1) or liquidacion (0)
	headers := map[string]string{
		"CID":   cid,
		"FID":   fid,
		"NNCC":  nncc,
		"AUSER": "False",
	}

	url := s.BaseURL + "/GetEstadoCuenta"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Add headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Add query parameters
	q := req.URL.Query()
	q.Add("AG", "true")
	q.Add("CODCLI", "cliente-CRITERIA")
	q.Add("FR", date.Format("2006-01-02"))
	q.Add("TF", tf)
	req.URL.RawQuery = q.Encode()

	// Add bearer token
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("failed to retrieve account status")
	}

	var result []EstadoCuentaSchema

	// Save the response and get the response bytes for further processing
	// body, err := utils.SaveResponseToFile(resp.Body, fmt.Sprintf("estado_cuenta_%s_date_response.json", cid))
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// GetEstadoCuenta retrieves the account status information
func (s *ESCOServiceClient) GetLiquidaciones(token, cid, fid, nncc, tf string, startDate, endDate time.Time) ([]Liquidacion, error) {
	// tf is filter by concertacion (-1) or liquidacion (0)
	headers := map[string]string{
		"CID":   cid,
		"FID":   fid,
		"NNCC":  nncc,
		"AUSER": "False",
	}

	url := s.BaseURL + "/GetLiquidaciones"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Add headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Add query parameters
	q := req.URL.Query()
	q.Add("AG", "true")
	q.Add("CODCLI", "cliente-CRITERIA")
	q.Add("FD", startDate.Format("2006-01-02"))
	q.Add("FH", endDate.Format("2006-01-02"))
	q.Add("TF", tf)
	req.URL.RawQuery = q.Encode()

	// Add bearer token
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("failed to retrieve liquidaciones")
	}

	var result []Liquidacion

	// Save the response and get the response bytes for further processing
	// body, err := utils.SaveResponseToFile(resp.Body, "liquidaciones_response.json")
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *ESCOServiceClient) GetBoletos(token, cid, fid, nncc, tf string, startDate, endDate time.Time) ([]Boleto, error) {
	// tf is filter by concertacion (-1) or liquidacion (0)
	headers := map[string]string{
		"CID":   cid,
		"FID":   fid,
		"NNCC":  nncc,
		"AUSER": "False",
	}

	url := s.BaseURL + "/GetBoletos"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	// Add headers
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// Add query parameters
	q := req.URL.Query()
	q.Add("AG", "true")
	q.Add("CODCLI", "cliente-CRITERIA")
	q.Add("FD", startDate.Format("2006-01-02"))
	q.Add("FH", endDate.Format("2006-01-02"))
	q.Add("TF", tf)
	req.URL.RawQuery = q.Encode()

	// Add bearer token
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("failed to retrieve boletos")
	}

	var result []Boleto

	// Save the response and get the response bytes for further processing
	// body, err := utils.SaveResponseToFile(resp.Body, "boletos_response.json")
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}

func (s *ESCOServiceClient) GetCategoryMap() map[string]string {
	if s.CategoryMap == nil {
		return map[string]string{}
	}
	return *s.CategoryMap
}
