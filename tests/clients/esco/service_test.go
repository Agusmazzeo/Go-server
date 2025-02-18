package esco_test

import (
	"context"
	"strconv"
	"testing"
	"time"
)

func TestESCOService(t *testing.T) {
	// cfg, err := config.LoadConfig("../../../settings")
	// if err != nil {
	// 	log.Println(err, "Error while loading config")
	// 	return
	// }

	// escoService, err := esco.NewClient(cfg)
	escoService, err := NewMockClient("../../test_files/clients/esco")
	if err != nil {
		t.Errorf("an error ocurred while creating the escoService: %s", err.Error())
	}
	token, err := escoService.PostToken(context.Background(), "user", "pass")
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
		result, err := escoService.GetEstadoCuenta(token.AccessToken, accounts[0].ID, accounts[0].FI, strconv.Itoa(accounts[0].N), "0", date, false)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(result) == 0 {
			t.Errorf("expected some results, got none")
		}
	})

	t.Run("GetBoletos with defined account works correctly", func(t *testing.T) {
		accounts, err := escoService.BuscarCuentas(token.AccessToken, "11170")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(accounts) == 0 {
			t.Errorf("expected some results, got none")
		}

		startDate := time.Now()
		endDate := startDate.AddDate(0, 0, 1)
		result, err := escoService.GetBoletos(token.AccessToken, accounts[0].ID, accounts[0].FI, strconv.Itoa(accounts[0].N), "0", startDate, endDate, false)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(result) == 0 {
			t.Errorf("expected some results, got none")
		}
	})

	t.Run("GetLiquidaciones with defined account works correctly", func(t *testing.T) {
		accounts, err := escoService.BuscarCuentas(token.AccessToken, "11170")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(accounts) == 0 {
			t.Errorf("expected some results, got none")
		}

		startDate := time.Now()
		endDate := startDate.AddDate(0, 0, 1)
		result, err := escoService.GetLiquidaciones(token.AccessToken, accounts[0].ID, accounts[0].FI, strconv.Itoa(accounts[0].N), "0", startDate, endDate, false)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(result) == 0 {
			t.Errorf("expected some results, got none")
		}
	})

	t.Run("GetCteCorriente with defined account works correctly", func(t *testing.T) {
		accounts, err := escoService.BuscarCuentas(token.AccessToken, "11170")
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(accounts) == 0 {
			t.Errorf("expected some results, got none")
		}

		startDate := time.Now()
		endDate := startDate.AddDate(0, 0, 1)
		result, err := escoService.GetCteCorriente(token.AccessToken, accounts[0].ID, accounts[0].FI, strconv.Itoa(accounts[0].N), "0", startDate, endDate, false)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if len(result) == 0 {
			t.Errorf("expected some results, got none")
		}
	})
}
