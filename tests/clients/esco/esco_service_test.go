package esco_test

import (
	"log"
	"os"
	"server/src/clients/esco"
	"server/src/config"
	"strconv"
	"testing"
	"time"
)

// LoadEnv loads environment variables for testing
func LoadEnv() {
	// Set your environment variables here or use a .env file
	os.Setenv("ESCO_BASE_URL", "https://clientes.criteria.com.ar/uniwa/api")
	os.Setenv("ESCO_TOKEN_URL", "https://clientes.criteria.com.ar/uniwa/api/token")
	os.Setenv("ESCO_CLIENT_ID", "Unisync")
	os.Setenv("ESCO_CLIENT_SECRET", "your_client_secret")
	os.Setenv("ESCO_USERNAME", "icastagno")
	os.Setenv("ESCO_PASSWORD", "Messiusa24!")
}

func TestESCOService(t *testing.T) {
	cfg, err := config.LoadConfig("../../../")
	if err != nil {
		log.Println(err, "Error while loading config")
		return
	}
	baseURL := cfg.ExternalClients.ESCO.BaseURL
	tokenURL := cfg.ExternalClients.ESCO.TokenURL
	clientID := cfg.ExternalClients.ESCO.ClientID
	clientSecret := ""
	username := cfg.ExternalClients.ESCO.Username
	password := cfg.ExternalClients.ESCO.Password

	escoService := esco.NewESCOServiceClient(baseURL, tokenURL, clientID, clientSecret, username, password)

	t.Run("BuscarCuentas with filter * works correctly", func(t *testing.T) {
		result, err := escoService.BuscarCuentas("*")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(result) == 0 {
			t.Errorf("expected some results, got none")
		}
	})

	t.Run("GetCuentaDetalle with defined account works correctly", func(t *testing.T) {
		accounts, err := escoService.BuscarCuentas("DIAGNOSTICO VETERINARIO")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(accounts) == 0 {
			t.Errorf("expected some results, got none")
		}

		result, err := escoService.GetCuentaDetalle(accounts[0].ID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(result.MERC) == 0 {
			t.Fatal("expected MERC len more than 0")
		}
	})

	t.Run("GetEstadoCuenta with defined account works correctly", func(t *testing.T) {
		accounts, err := escoService.BuscarCuentas("DIAGNOSTICO VETERINARIO")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(accounts) == 0 {
			t.Errorf("expected some results, got none")
		}

		date := time.Now()
		result, err := escoService.GetEstadoCuenta(accounts[0].ID, accounts[0].FI, strconv.Itoa(accounts[0].N), date)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(result) == 0 {
			t.Errorf("expected some results, got none")
		}
	})
}
