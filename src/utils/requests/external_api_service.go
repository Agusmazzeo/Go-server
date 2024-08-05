package requests

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"time"
)

// ExternalAPIService is a struct representing a configurable external service
type ExternalAPIService struct {
	BaseURL      string
	TokenURL     string
	ClientID     string
	ClientSecret string
	Token        string
	UserID       string
	TokenExpiry  time.Time
	Username     string
	Password     string
}

// NewExternalAPIService creates a new instance of ExternalAPIService
func NewExternalAPIService(baseURL, tokenURL, clientID, clientSecret, username, password string) *ExternalAPIService {
	return &ExternalAPIService{
		BaseURL:      baseURL,
		TokenURL:     tokenURL,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Username:     username,
		Password:     password,
	}
}

// GetToken retrieves and sets the token for the external service
func (s *ExternalAPIService) GetToken() error {
	if s.Token != "" && time.Now().Before(s.TokenExpiry) {
		return nil // Token is still valid
	}

	data := url.Values{}
	data.Set("grant_type", "password")
	data.Set("username", s.Username)
	data.Set("password", s.Password)
	data.Set("client_id", s.ClientID)

	req, err := http.NewRequest("POST", s.TokenURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// if resp.StatusCode != http.StatusOK {
	// 	return errors.New("failed to retrieve token")
	// }

	var result map[string]interface{}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	err = json.Unmarshal(body, &result)
	if err != nil {
		return err
	}

	token, ok := result["access_token"].(string)
	if !ok {
		return errors.New("failed to parse access token")
	}
	expiresIn, ok := result["expires_in"].(float64)
	if !ok {
		return errors.New("failed to parse expires_in")
	}
	userID, ok := result["userID"].(string)
	if !ok {
		return errors.New("failed to parse userID")
	}

	s.Token = token
	s.TokenExpiry = time.Now().Add(time.Duration(expiresIn) * time.Second)
	s.UserID = userID

	return nil
}

// makeRequest is a helper function to make HTTP requests
func (s *ExternalAPIService) makeRequest(method, endpoint string, body interface{}) (*http.Response, error) {
	err := s.GetToken()
	if err != nil {
		return nil, err
	}

	var jsonBody []byte
	if body != nil {
		jsonBody, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, s.BaseURL+endpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+s.Token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	return client.Do(req)
}

// Get makes a GET request to the external service
func (s *ExternalAPIService) Get(endpoint string) (*http.Response, error) {
	return s.makeRequest("GET", endpoint, nil)
}

// Post makes a POST request to the external service
func (s *ExternalAPIService) Post(endpoint string, body interface{}) (*http.Response, error) {
	return s.makeRequest("POST", endpoint, body)
}

// Put makes a PUT request to the external service
func (s *ExternalAPIService) Put(endpoint string, body interface{}) (*http.Response, error) {
	return s.makeRequest("PUT", endpoint, body)
}

// Delete makes a DELETE request to the external service
func (s *ExternalAPIService) Delete(endpoint string) (*http.Response, error) {
	return s.makeRequest("DELETE", endpoint, nil)
}

// PostWithHeaders makes a POST request with custom headers
func (s *ExternalAPIService) PostWithHeaders(endpoint string, body interface{}, headers map[string]string) (*http.Response, error) {
	err := s.GetToken()
	if err != nil {
		return nil, err
	}

	var jsonBody []byte
	if body != nil {
		jsonBody, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest("POST", s.BaseURL+endpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+s.Token)
	req.Header.Set("Content-Type", "application/json")
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := &http.Client{}
	return client.Do(req)
}
