package controllers_test

import (
	"context"
	"testing"
	"time"
)

func TestGetAllCurrencies(t *testing.T) {

	currencies, err := ctrl.GetAllCurrencies(context.Background())
	if err != nil {
		t.Error(err)
	}

	if len(currencies) == 0 {
		t.Errorf("expected GetAllAccounts to return more than 0 accounts")
	}

}

func TestGetCurrencyWithValuationByID(t *testing.T) {

	currency, err := ctrl.GetCurrencyWithValuationByID(context.Background(), "USD", time.Date(2024, 10, 4, 0, 0, 0, 0, time.UTC)) // Use a valid account ID here
	if err != nil {
		t.Error(err)
	}

	if currency == nil || currency.Valuations == nil {
		t.Errorf("expected currency to be returned")
	}

	if len(currency.Valuations) == 0 {
		t.Errorf("expected currency valuations to be returned")
	}
}

func TestGetCurrencyWithValuationDateRangeByID(t *testing.T) {

	// Use a valid account ID and date range here
	startDate := time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC)
	endDate := startDate.AddDate(0, 0, 6)
	currency, err := ctrl.GetCurrencyWithValuationDateRangeByID(context.Background(), "USD", startDate, endDate) // Use a valid account ID here
	if err != nil {
		t.Error(err)
	}

	if currency == nil || currency.Valuations == nil {
		t.Errorf("expected currency to be returned")
	}

	if len(currency.Valuations) == 0 {
		t.Errorf("expected currency valuations to be returned")
	}
}
