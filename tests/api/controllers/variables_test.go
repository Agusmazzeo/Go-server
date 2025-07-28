package controllers_test

import (
	"context"
	"reflect"
	"server/src/schemas"
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

func TestCompleteValuations(t *testing.T) {
	tests := []struct {
		name      string
		input     *schemas.VariableWithValuationResponse
		startDate time.Time
		endDate   time.Time
		expected  []schemas.VariableValuation
	}{
		{
			name: "No gaps in dates",
			input: &schemas.VariableWithValuationResponse{
				Valuations: []schemas.VariableValuation{
					{Date: "2023-11-01", Value: 10.5},
					{Date: "2023-11-02", Value: 11.0},
					{Date: "2023-11-03", Value: 12.0},
				},
			},
			startDate: time.Date(2023, 11, 1, 0, 0, 0, 0, time.UTC),
			endDate:   time.Date(2023, 11, 3, 0, 0, 0, 0, time.UTC),
			expected: []schemas.VariableValuation{
				{Date: "2023-11-01", Value: 10.5},
				{Date: "2023-11-02", Value: 11.0},
				{Date: "2023-11-03", Value: 12.0},
			},
		},
		{
			name: "Gaps in dates with last known value filling",
			input: &schemas.VariableWithValuationResponse{
				Valuations: []schemas.VariableValuation{
					{Date: "2023-11-01", Value: 10.5},
					{Date: "2023-11-03", Value: 12.0},
				},
			},
			startDate: time.Date(2023, 11, 1, 0, 0, 0, 0, time.UTC),
			endDate:   time.Date(2023, 11, 5, 0, 0, 0, 0, time.UTC),
			expected: []schemas.VariableValuation{
				{Date: "2023-11-01", Value: 10.5},
				{Date: "2023-11-02", Value: 10.5},
				{Date: "2023-11-03", Value: 12.0},
				{Date: "2023-11-04", Value: 12.0},
				{Date: "2023-11-05", Value: 12.0},
			},
		},
		{
			name: "Empty valuations",
			input: &schemas.VariableWithValuationResponse{
				Valuations: []schemas.VariableValuation{},
			},
			startDate: time.Date(2023, 11, 1, 0, 0, 0, 0, time.UTC),
			endDate:   time.Date(2023, 11, 3, 0, 0, 0, 0, time.UTC),
			expected:  []schemas.VariableValuation{},
		},
		{
			name: "Date range outside of valuations",
			input: &schemas.VariableWithValuationResponse{
				Valuations: []schemas.VariableValuation{
					{Date: "2023-11-02", Value: 10.0},
				},
			},
			startDate: time.Date(2023, 11, 1, 0, 0, 0, 0, time.UTC),
			endDate:   time.Date(2023, 11, 3, 0, 0, 0, 0, time.UTC),
			expected: []schemas.VariableValuation{
				{Date: "2023-11-02", Value: 10.0},
				{Date: "2023-11-03", Value: 10.0},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ctrl.CompleteValuations(test.input, test.startDate, test.endDate)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !reflect.DeepEqual(test.input.Valuations, test.expected) && len(test.input.Valuations) > 0 {
				t.Errorf("for test %q, expected %v but got %v", test.name, test.expected, test.input.Valuations)
			}
		})
	}
}
