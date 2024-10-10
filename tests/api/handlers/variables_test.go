package handlers_test

import (
	"net/http"
	"testing"
)

func TestGetAllVariables(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, ts.URL+"/api/variables", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Authorization", "Bearer "+token.AccessToken)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != http.StatusOK {
		t.Errorf("expected status OK; got %v", res.Status)
	}
}

func TestGetVariableWithValuationByID(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, ts.URL+"/api/variables/13?startDate=2024-08-01&endDate=2024-08-02", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Authorization", "Bearer "+token.AccessToken)

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != http.StatusOK {
		t.Errorf("expected status OK; got %v", res.Status)
	}
}
