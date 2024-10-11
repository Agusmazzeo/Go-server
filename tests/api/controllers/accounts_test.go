package controllers_test

import (
	"context"
	"testing"
	"time"
)

func TestGetAllAccounts(t *testing.T) {

	accounts, err := accountsController.GetAllAccounts(context.Background(), token.AccessToken, "DIAGNOSTICO VETERINARIO")
	if err != nil {
		t.Error(err)
	}

	if len(accounts) == 0 {
		t.Errorf("expected GetAllAccounts to return more than 0 accounts")
	}

}

func TestGetAccountByID(t *testing.T) {

	account, err := accountsController.GetAccountByID(context.Background(), token.AccessToken, "11170") // Use a valid account ID here
	if err != nil {
		t.Error(err)
	}

	if account == nil {
		t.Errorf("expected account to be returned")
	}
}

func TestGetAccountState(t *testing.T) {

	// Use a valid account ID and date here
	date := time.Now()
	accountState, err := accountsController.GetAccountState(context.Background(), token.AccessToken, "11170", date)
	if err != nil {
		t.Error(err)
	}

	if accountState == nil || len(*accountState.Vouchers) == 0 {
		t.Errorf("expected non-empty account state")
	}
}

func TestGetAccountStateDateRange(t *testing.T) {

	// Use a valid account ID and date range here
	startDate := time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 0, 6)
	accountState, err := accountsController.GetAccountStateDateRange(context.Background(), token.AccessToken, "11170", startDate, endDate)
	if err != nil {
		t.Error(err)
	}

	if accountState == nil || len(*accountState.Vouchers) == 0 {
		t.Errorf("expected non-empty account state for the date range")
	}
}
