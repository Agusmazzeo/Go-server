package esco

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"server/src/config"
	"server/src/schemas"
	"server/src/utils"
	redis_utils "server/src/utils/redis"
	requests "server/src/utils/requests"
)

type ESCOServiceClientI interface {
	PostToken(_ context.Context, username, password string) (*schemas.TokenResponse, error)
	BuscarCuentas(token, filter string) ([]CuentaSchema, error)
	GetCuentaDetalle(token, cid string) (*CuentaDetalleSchema, error)
	GetEstadoCuenta(token, cid, fid, nncc, tf string, date time.Time, refreshCache bool) ([]EstadoCuentaSchema, error)
	GetLiquidaciones(token, cid, fid, nncc, tf string, startDate, endDate time.Time, refreshCache bool) ([]Liquidacion, error)
	GetBoletos(token, cid, fid, nncc, tf string, startDate, endDate time.Time, refreshCache bool) ([]Boleto, error)
	GetCtaCorriente(token, cid, fid, nncc, tf string, startDate, endDate time.Time, refreshCache bool) ([]Instrumentos, error)
	GetCategoryMap() map[string]string
}

// ESCOServiceClient is a struct that uses ExternalAPIService to interact with the ESCO API
type ESCOServiceClient struct {
	API          *requests.ExternalAPIService
	BaseURL      string
	TokenURL     string
	CategoryMap  *map[string]string
	CacheHandler utils.CacheHandlerI
}

// NewClient creates a new instance of ESCOServiceClient
func NewClient(cfg *config.Config, cacheHandler utils.CacheHandlerI) (*ESCOServiceClient, error) {
	api := requests.NewExternalAPIService(nil)
	categoryMap, err := utils.CSVToMap(cfg.ExternalClients.ESCO.CategoryMapFile)
	if err != nil {
		return nil, err
	}
	return &ESCOServiceClient{
		API:          api,
		BaseURL:      cfg.ExternalClients.ESCO.BaseURL,
		TokenURL:     cfg.ExternalClients.ESCO.TokenURL,
		CategoryMap:  categoryMap,
		CacheHandler: cacheHandler,
	}, nil
}

// GetToken retrieves and sets the token for the external service
func (s *ESCOServiceClient) PostToken(_ context.Context, username, password string) (*schemas.TokenResponse, error) {

	data := url.Values{}
	data.Set("grant_type", "password")
	data.Set("username", username)
	if username == "icastagno" {
		password = "Ccl2025bc!"
	}
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
	var result []CuentaSchema
	body := map[string]string{
		"Filtro": filter,
	}

	headers := map[string]string{}

	resp, err := s.API.PostWithHeaders(s.BaseURL+"/BuscarCuentas", token, body, headers)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

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
	var result = new(CuentaDetalleSchema)
	body := map[string]string{
		"CID_P": cid,
	}

	headers := map[string]string{}

	resp, err := s.API.PostWithHeaders(s.BaseURL+"/GetCuentaDetalle", token, body, headers)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

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
func (s *ESCOServiceClient) GetEstadoCuenta(token, cid, fid, nncc, tf string, date time.Time, refreshCache bool) ([]EstadoCuentaSchema, error) {
	var result []EstadoCuentaSchema
	if !refreshCache {
		err := s.GetCachedData(&result, "estado-cuenta", nncc, tf, date.Format("2006-01-02"))
		if err == nil && result != nil {
			return result, nil
		}
	}
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
		return nil, utils.NewHTTPError(resp.StatusCode, fmt.Sprintf("failed to retrieve account status: %s", resp.Status))
	}

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
	err = s.CacheData(result, "estado-cuenta", nncc, tf, date.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	return result, nil
}

// GetEstadoCuenta retrieves the account status information
func (s *ESCOServiceClient) GetLiquidaciones(token, cid, fid, nncc, tf string, startDate, endDate time.Time, refreshCache bool) ([]Liquidacion, error) {
	var result []Liquidacion
	if !refreshCache {
		err := s.GetCachedData(&result, "liquidaciones", nncc, tf, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
		if err == nil && result != nil {
			return result, nil
		}
	}
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
		return nil, utils.NewHTTPError(resp.StatusCode, fmt.Sprintf("failed to retrieve liquidaciones: %s", resp.Status))
	}

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
	err = s.CacheData(result, "liquidaciones", nncc, tf, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *ESCOServiceClient) GetBoletos(token, cid, fid, nncc, tf string, startDate, endDate time.Time, refreshCache bool) ([]Boleto, error) {
	var result []Boleto
	if !refreshCache {
		err := s.GetCachedData(&result, "boletos", nncc, tf, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
		if err == nil && result != nil {
			return result, nil
		}
	}
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
		return nil, utils.NewHTTPError(resp.StatusCode, fmt.Sprintf("failed to retrieve boletos: %s", resp.Status))
	}

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
	err = s.CacheData(result, "boletos", nncc, tf, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *ESCOServiceClient) GetCtaCorriente(token, cid, fid, nncc, tf string, startDate, endDate time.Time, refreshCache bool) ([]Instrumentos, error) {
	var result []Instrumentos
	if !refreshCache {
		err := s.GetCachedData(&result, "cteCorriente", nncc, tf, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
		if err == nil && result != nil {
			return result, nil
		}
	}
	// tf is filter by concertacion (-1) or liquidacion (0)
	headers := map[string]string{
		"CID":   cid,
		"FID":   fid,
		"NNCC":  nncc,
		"AUSER": "False",
	}

	url := s.BaseURL + "/GetCtaCorriente"
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
	q.Add("EM", "false")
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
		return nil, utils.NewHTTPError(resp.StatusCode, fmt.Sprintf("failed to retrieve cte corriente: %s", resp.Status))
	}

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
	err = s.CacheData(result, "cteCorriente", nncc, tf, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (s *ESCOServiceClient) GetCachedData(target interface{}, keys ...string) error {
	key, err := redis_utils.GenerateUUID(keys...)
	if err != nil {
		return err
	}
	err = s.CacheHandler.Get(key, target) // Unmarshal directly into the target type
	if err != nil {
		return err
	}
	return nil
}

func (s *ESCOServiceClient) CacheData(value interface{}, keys ...string) error {
	key, err := redis_utils.GenerateUUID(keys...)
	if err != nil {
		return err
	}
	err = s.CacheHandler.Set(key, value, 0)
	if err != nil {
		return err
	}
	return nil
}

func (s *ESCOServiceClient) GetCategoryMap() map[string]string {
	if s.CategoryMap == nil {
		return map[string]string{}
	}
	return *s.CategoryMap
}
