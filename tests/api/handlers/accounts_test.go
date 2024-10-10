package handlers_test

import (
	"net/http"
	"testing"
)

func TestGetAllAccounts(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, ts.URL+"/api/accounts?filter=DIAGNOSTICO", nil)
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

func TestGetAccountState(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, ts.URL+"/api/accounts/11170?date=2024-08-01", nil)
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

func TestGetAccountStateDateRange(t *testing.T) {
	req, err := http.NewRequest(http.MethodGet, ts.URL+"/api/accounts/11170?startDate=2024-08-01&endDate=2024-08-03", nil)
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
