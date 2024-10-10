package controllers_test

import (
	"context"
	"testing"
	"time"
)

func TestGetAllVariables(t *testing.T) {

	variables, err := ctrl.GetAllVariables(context.Background())
	if err != nil {
		t.Error(err)
	}

	if len(variables) == 0 {
		t.Errorf("expected GetAllAccounts to return more than 0 accounts")
	}

}

func TestGetVariableWithValuationByID(t *testing.T) {

	variable, err := ctrl.GetVariableWithValuationByID(context.Background(), "USD", time.Date(2024, 10, 4, 0, 0, 0, 0, time.UTC)) // Use a valid account ID here
	if err != nil {
		t.Error(err)
	}

	if variable == nil || variable.Valuations == nil {
		t.Errorf("expected variable to be returned")
	}

	if len(variable.Valuations) == 0 {
		t.Errorf("expected variable valuations to be returned")
	}
}

func TestGetVariableWithValuationDateRangeByID(t *testing.T) {

	// Use a valid account ID and date range here
	startDate := time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 10, 8, 0, 0, 0, 0, time.UTC)
	variable, err := ctrl.GetVariableWithValuationDateRangeByID(context.Background(), "13", startDate, endDate) // Use a valid account ID here
	if err != nil {
		t.Error(err)
	}

	if variable == nil || variable.Valuations == nil {
		t.Errorf("expected variable to be returned")
	}

	if len(variable.Valuations) == 0 {
		t.Errorf("expected variable valuations to be returned")
	}
}
