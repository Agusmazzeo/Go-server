package handlers_test

import (
	"net/http"
	"testing"
)

func TestGetAllCurrencies(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, ts.URL+"/api/currencies", nil)
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

func TestGetCurrencyWithValuationByID(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, ts.URL+"/api/currencies/USD?date=2024-08-01", nil)
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
