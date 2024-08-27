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
	requests "server/src/utils/requests"
)

// ESCOServiceClient is a struct that uses ExternalAPIService to interact with the ESCO API
type ESCOServiceClient struct {
	API      *requests.ExternalAPIService
	BaseURL  string
	TokenURL string
}

// NewClient creates a new instance of ESCOServiceClient
func NewClient(cfg *config.Config) *ESCOServiceClient {
	api := requests.NewExternalAPIService()
	return &ESCOServiceClient{
		API:      api,
		BaseURL:  cfg.ExternalClients.ESCO.BaseURL,
		TokenURL: cfg.ExternalClients.ESCO.TokenURL,
	}
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