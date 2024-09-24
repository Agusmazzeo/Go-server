package esco_test

import (
	"context"
	"log"
	"server/src/clients/esco"
	"server/src/config"
	"strconv"
	"testing"
	"time"
)

func TestESCOService(t *testing.T) {
	cfg, err := config.LoadConfig("../../../settings")
	if err != nil {
		log.Println(err, "Error while loading config")
		return
	}

	escoService, err := esco.NewClient(cfg)
	if err != nil {
		t.Errorf("an error ocurred while creating the escoService: %s", err.Error())
	}
	token, err := escoService.PostToken(context.Background(), "icastagno", "Messiusa24!")
	if err != nil {
		t.Errorf("an error ocurred while retrieving the token: %s", err.Error())
	}

	t.Run("BuscarCuentas with filter * works correctly", func(t *testing.T) {
		result, err := escoService.BuscarCuentas(token.AccessToken, "*")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(result) == 0 {
			t.Errorf("expected some results, got none")
		}
	})

	t.Run("GetCuentaDetalle with defined account works correctly", func(t *testing.T) {
		accounts, err := escoService.BuscarCuentas(token.AccessToken, "DIAGNOSTICO VETERINARIO")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(accounts) == 0 {
			t.Errorf("expected some results, got none")
		}

		result, err := escoService.GetCuentaDetalle(token.AccessToken, accounts[0].ID)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(result.MERC) == 0 {
			t.Fatal("expected MERC len more than 0")
		}
	})

	t.Run("GetEstadoCuenta with defined account works correctly", func(t *testing.T) {
		accounts, err := escoService.BuscarCuentas(token.AccessToken, "11170")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(accounts) == 0 {
			t.Errorf("expected some results, got none")
		}

		date := time.Now()
		result, err := escoService.GetEstadoCuenta(token.AccessToken, accounts[0].ID, accounts[0].FI, strconv.Itoa(accounts[0].N), "-1", date)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(result) == 0 {
			t.Errorf("expected some results, got none")
		}
	})
}
