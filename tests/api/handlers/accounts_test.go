package handlers_test

import (
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"server/src/api/handlers"
	"server/src/config"

	"github.com/go-chi/chi/v5"
)

var ts *httptest.Server

func TestMain(m *testing.M) {
	cfg, err := config.LoadConfig("../../../settings")
	if err != nil {
		log.Println(err, "Error while loading config")
		os.Exit(1)
	}

	r := chi.NewRouter()
	h, err := handlers.NewHandler(cfg)
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

	os.Exit(m.Run())
}

func TestGetAllAccounts(t *testing.T) {
	res, err := http.Get(ts.URL + "/api/accounts?filter=DIAGNOSTICO")
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != http.StatusOK {
		t.Errorf("expected status OK; got %v", res.Status)
	}
}

func TestGetAccountState(t *testing.T) {
	res, err := http.Get(ts.URL + "/api/accounts/11170?date=2024-08-01")
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != http.StatusOK {
		t.Errorf("expected status OK; got %v", res.Status)
	}
}

func TestGetAccountStateDateRange(t *testing.T) {
	res, err := http.Get(ts.URL + "/api/accounts/11170?startDate=2024-08-01&endDate=2024-08-03")
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode != http.StatusOK {
		t.Errorf("expected status OK; got %v", res.Status)
	}
}
