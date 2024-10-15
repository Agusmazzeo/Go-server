package requests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"server/src/utils"
)

// ExternalAPIService is a struct representing a configurable external service
type ExternalAPIService struct{}

// NewExternalAPIService creates a new instance of ExternalAPIService
func NewExternalAPIService() *ExternalAPIService {
	return &ExternalAPIService{}
}

// makeRequest is a helper function to make HTTP requests, supporting optional query parameters
func (s *ExternalAPIService) makeRequest(method, endpoint, token string, params url.Values, body interface{}) (*http.Response, error) {
	// Convert params to query string
	if params != nil {
		endpoint = endpoint + "?" + params.Encode()
	}

	// Marshal the body to JSON if it's provided
	var err error
	var jsonBody []byte
	if body != nil {
		jsonBody, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}

	// Create the request
	req, err := http.NewRequest(method, endpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	// Add headers
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	// Execute the request
	client := &http.Client{}
	return client.Do(req)
}

// Get makes a GET request to the external service, accepting optional query parameters
func (s *ExternalAPIService) Get(endpoint, token string, params url.Values) (*http.Response, error) {
	return s.makeRequest("GET", endpoint, token, params, nil)
}

// Post makes a POST request to the external service, accepting optional query parameters
func (s *ExternalAPIService) Post(endpoint, token string, params url.Values, body interface{}) (*http.Response, error) {
	return s.makeRequest("POST", endpoint, token, params, body)
}

// Put makes a PUT request to the external service, accepting optional query parameters
func (s *ExternalAPIService) Put(endpoint, token string, params url.Values, body interface{}) (*http.Response, error) {
	return s.makeRequest("PUT", endpoint, token, params, body)
}

// Delete makes a DELETE request to the external service, accepting optional query parameters
func (s *ExternalAPIService) Delete(endpoint, token string, params url.Values) (*http.Response, error) {
	return s.makeRequest("DELETE", endpoint, token, params, nil)
}

// PostWithHeaders makes a POST request with custom headers
func (s *ExternalAPIService) PostWithHeaders(endpoint, token string, body interface{}, headers map[string]string) (*http.Response, error) {
	var err error
	var jsonBody []byte
	if body != nil {
		jsonBody, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode > http.StatusCreated {
		return nil, utils.NewHTTPError(resp.StatusCode, resp.Status)
	}
	return resp, nil
}
