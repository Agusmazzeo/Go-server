package requests

import (
	"bytes"
	"encoding/json"
	"net/http"
)

// ExternalAPIService is a struct representing a configurable external service
type ExternalAPIService struct{}

// NewExternalAPIService creates a new instance of ExternalAPIService
func NewExternalAPIService() *ExternalAPIService {
	return &ExternalAPIService{}
}

// makeRequest is a helper function to make HTTP requests
func (s *ExternalAPIService) makeRequest(method, endpoint, token string, body interface{}) (*http.Response, error) {
	var err error
	var jsonBody []byte
	if body != nil {
		jsonBody, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, endpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	return client.Do(req)
}

// Get makes a GET request to the external service
func (s *ExternalAPIService) Get(endpoint, token string) (*http.Response, error) {
	return s.makeRequest("GET", endpoint, token, nil)
}

// Post makes a POST request to the external service
func (s *ExternalAPIService) Post(endpoint, token string, body interface{}) (*http.Response, error) {
	return s.makeRequest("POST", endpoint, token, body)
}

// Put makes a PUT request to the external service
func (s *ExternalAPIService) Put(endpoint, token string, body interface{}) (*http.Response, error) {
	return s.makeRequest("PUT", endpoint, token, body)
}

// Delete makes a DELETE request to the external service
func (s *ExternalAPIService) Delete(endpoint, token string) (*http.Response, error) {
	return s.makeRequest("DELETE", endpoint, token, nil)
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
	return client.Do(req)
}
