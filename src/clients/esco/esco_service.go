package esco

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	requests "server/src/utils/requests"
)

// ESCOServiceClient is a struct that uses ExternalAPIService to interact with the ESCO API
type ESCOServiceClient struct {
	API *requests.ExternalAPIService
}

// NewESCOServiceClient creates a new instance of ESCOServiceClient
func NewESCOServiceClient(baseURL, tokenURL, clientID, clientSecret, username, password string) *ESCOServiceClient {
	api := requests.NewExternalAPIService(baseURL, tokenURL, clientID, clientSecret, username, password)
	return &ESCOServiceClient{API: api}
}

// BuscarCuentas retrieves all accounts matching filter
func (s *ESCOServiceClient) BuscarCuentas(filter string) ([]CuentaSchema, error) {
	userID := s.API.Username // Assuming s.API.Username is the correct way to get the userID
	body := map[string]string{
		"Filtro": filter,
		"USERID": userID,
	}

	headers := map[string]string{}

	resp, err := s.API.PostWithHeaders("/BuscarCuentas", body, headers)
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
func (s *ESCOServiceClient) GetCuentaDetalle(cid string) (*CuentaDetalleSchema, error) {
	body := map[string]string{
		"CID_P": cid,
	}

	headers := map[string]string{}

	resp, err := s.API.PostWithHeaders("/GetCuentaDetalle", body, headers)
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
func (s *ESCOServiceClient) GetEstadoCuenta(cid, fid, nncc string, date time.Time) ([]EstadoCuentaSchema, error) {
	headers := map[string]string{
		"UID":   s.API.Username, // Assuming this is how you get the user ID
		"CID":   cid,
		"FID":   fid,
		"NNCC":  nncc,
		"AUSER": "False",
	}

	url := s.API.BaseURL + "/GetEstadoCuenta"
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
	q.Add("FR", date.Format("2006-01-02T15:04:05.000Z"))
	q.Add("TF", "-1")
	req.URL.RawQuery = q.Encode()

	// Add bearer token
	req.Header.Set("Authorization", "Bearer "+s.API.Token)

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
