package controllers_test

import (
	"context"
	"log"
	"os"
	"server/src/api/controllers"
	"server/src/clients/esco"
	"server/src/config"
	"server/src/schemas"
	"testing"
	"time"
)

var token *schemas.TokenResponse
var escoClient *esco.ESCOServiceClient

func TestMain(m *testing.M) {

	cfg, err := config.LoadConfig("../../../settings")
	if err != nil {
		log.Println(err, "Error while loading config")
		os.Exit(1)
	}
	cfg.ExternalClients.ESCO.CategoryMapFile = "../../test_files/utils/denominaciones.csv"

	escoClient, err = esco.NewClient(cfg)
	if err != nil {
		log.Println(err, "Error while creating ESCO Client")
		os.Exit(1)
	}

	token, err = escoClient.PostToken(context.Background(), "icastagno", "Messiusa24!")
	if err != nil {
		log.Println(err, "Error while getting esco token")
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func TestGetAllAccounts(t *testing.T) {

	ctrl := controllers.NewController(nil, escoClient)

	accounts, err := ctrl.GetAllAccounts(context.Background(), token.AccessToken, "DIAGNOSTICO VETERINARIO")
	if err != nil {
		t.Error(err)
	}

	if len(accounts) == 0 {
		t.Errorf("expected GetAllAccounts to return more than 0 accounts")
	}

}

func TestGetAccountByID(t *testing.T) {

	ctrl := controllers.NewController(nil, escoClient)

	account, err := ctrl.GetAccountByID(context.Background(), token.AccessToken, "11170") // Use a valid account ID here
	if err != nil {
		t.Error(err)
	}

	if account == nil {
		t.Errorf("expected account to be returned")
	}
}

func TestGetAccountState(t *testing.T) {

	ctrl := controllers.NewController(nil, escoClient)

	// Use a valid account ID and date here
	date := time.Now()
	accountState, err := ctrl.GetAccountState(context.Background(), token.AccessToken, "11170", date)
	if err != nil {
		t.Error(err)
	}

	if accountState == nil || len(*accountState.Vouchers) == 0 {
		t.Errorf("expected non-empty account state")
	}
}

func TestGetAccountStateDateRange(t *testing.T) {

	ctrl := controllers.NewController(nil, escoClient)

	// Use a valid account ID and date range here
	startDate := time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 0, 6)
	accountState, err := ctrl.GetAccountStateDateRange(context.Background(), token.AccessToken, "11170", startDate, endDate)
	if err != nil {
		t.Error(err)
	}

	if accountState == nil || len(*accountState.Vouchers) == 0 {
		t.Errorf("expected non-empty account state for the date range")
	}
}
