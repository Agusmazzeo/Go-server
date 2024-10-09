package handlers_test

import (
	"context"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"server/src/api/handlers"
	"server/src/config"
	"server/src/schemas"
	bcra_test "server/tests/clients/bcra"
	esco_test "server/tests/clients/esco"

	"github.com/go-chi/chi/v5"
)

var ts *httptest.Server
var token *schemas.TokenResponse

func TestMain(m *testing.M) {
	cfg, err := config.LoadConfig("../../../settings", os.Getenv("ENV"))
	if err != nil {
		log.Println(err, "Error while loading config")
		os.Exit(1)
	}
	cfg.ExternalClients.ESCO.CategoryMapFile = "../../test_files/clients/esco/denominaciones.csv"

	escoClient, err := esco_test.NewMockClient("../../test_files/clients/esco")
	if err != nil {
		log.Println(err, "Error while creating Mock Esco Client")
		os.Exit(1)
	}
	bcraClient, err := bcra_test.NewMockClient("../../test_files/clients/bcra")
	if err != nil {
		log.Println(err, "Error while creating Mock BCRA Client")
		os.Exit(1)
	}

	r := chi.NewRouter()
	h, err := handlers.NewHandler(nil, escoClient, bcraClient)
	if err != nil {
		log.Println(err, "Error while starting handler")
		os.Exit(1)
	}

	r.Route("/api/accounts", func(r chi.Router) {
		r.Get("/", h.GetAllAccounts)
		r.Get("/{id}", h.GetAccountState)
	})

	ts = httptest.NewServer(r)
	defer ts.Close()

	token, err = h.Controller.PostToken(context.Background(), "user", "pass")
	if err != nil {
		log.Println(err, "Error while getting esco token")
		os.Exit(1)
	}

	os.Exit(m.Run())
}

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
