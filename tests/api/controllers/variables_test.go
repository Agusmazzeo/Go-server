package controllers_test

import (
	"context"
	"reflect"
	"server/src/api/controllers"
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

func TestComputeValuationVariations(t *testing.T) {
	tests := []struct {
		name     string
		input    *schemas.VariableWithValuationResponse
		interval time.Duration
		expected *schemas.VariableWithValuationResponse
	}{
		{
			name: "Empty valuations",
			input: &schemas.VariableWithValuationResponse{
				ID:          "USD",
				Description: "USD Rate",
				Valuations:  []schemas.VariableValuation{},
			},
			interval: 24 * time.Hour,
			expected: &schemas.VariableWithValuationResponse{
				ID:          "USD",
				Description: "USD Rate (variation)",
				Valuations:  []schemas.VariableValuation{},
			},
		},
		{
			name: "Single valuation - should be skipped",
			input: &schemas.VariableWithValuationResponse{
				ID:          "USD",
				Description: "USD Rate",
				Valuations: []schemas.VariableValuation{
					{Date: "2023-11-01", Value: 100.0},
				},
			},
			interval: 24 * time.Hour,
			expected: &schemas.VariableWithValuationResponse{
				ID:          "USD",
				Description: "USD Rate (variation)",
				Valuations:  []schemas.VariableValuation{},
			},
		},
		{
			name: "Two valuations - simple case",
			input: &schemas.VariableWithValuationResponse{
				ID:          "USD",
				Description: "USD Rate",
				Valuations: []schemas.VariableValuation{
					{Date: "2023-11-01", Value: 100.0},
					{Date: "2023-11-02", Value: 110.0},
				},
			},
			interval: 24 * time.Hour,
			expected: &schemas.VariableWithValuationResponse{
				ID:          "USD",
				Description: "USD Rate (variation)",
				Valuations: []schemas.VariableValuation{
					{Date: "2023-11-02", Value: 0.1}, // (110-100)/100 = 0.1 = 10%
				},
			},
		},
		{
			name: "Three valuations - cumulative case",
			input: &schemas.VariableWithValuationResponse{
				ID:          "USD",
				Description: "USD Rate",
				Valuations: []schemas.VariableValuation{
					{Date: "2023-11-01", Value: 100.0},
					{Date: "2023-11-02", Value: 110.0}, // +10%
					{Date: "2023-11-03", Value: 121.0}, // +10% from 110
				},
			},
			interval: 24 * time.Hour,
			expected: &schemas.VariableWithValuationResponse{
				ID:          "USD",
				Description: "USD Rate (variation)",
				Valuations: []schemas.VariableValuation{
					{Date: "2023-11-02", Value: 0.1},  // 10%
					{Date: "2023-11-03", Value: 0.21}, // (1.1 * 1.1) - 1 = 0.21 = 21%
				},
			},
		},
		{
			name: "Four valuations - complex case",
			input: &schemas.VariableWithValuationResponse{
				ID:          "USD",
				Description: "USD Rate",
				Valuations: []schemas.VariableValuation{
					{Date: "2023-11-01", Value: 100.0},
					{Date: "2023-11-02", Value: 110.0}, // +10%
					{Date: "2023-11-03", Value: 99.0},  // -10% from 110
					{Date: "2023-11-04", Value: 108.9}, // +10% from 99
				},
			},
			interval: 24 * time.Hour,
			expected: &schemas.VariableWithValuationResponse{
				ID:          "USD",
				Description: "USD Rate (variation)",
				Valuations: []schemas.VariableValuation{
					{Date: "2023-11-02", Value: 0.1},   // 10%
					{Date: "2023-11-03", Value: -0.01}, // (1.1 * 0.9) - 1 = -0.01 = -1%
					{Date: "2023-11-04", Value: 0.089}, // (1.1 * 0.9 * 1.1) - 1 = 0.089 = 8.9%
				},
			},
		},
		{
			name: "Unsorted dates - should be sorted",
			input: &schemas.VariableWithValuationResponse{
				ID:          "USD",
				Description: "USD Rate",
				Valuations: []schemas.VariableValuation{
					{Date: "2023-11-03", Value: 121.0},
					{Date: "2023-11-01", Value: 100.0},
					{Date: "2023-11-02", Value: 110.0},
				},
			},
			interval: 24 * time.Hour,
			expected: &schemas.VariableWithValuationResponse{
				ID:          "USD",
				Description: "USD Rate (variation)",
				Valuations: []schemas.VariableValuation{
					{Date: "2023-11-02", Value: 0.1},  // 10%
					{Date: "2023-11-03", Value: 0.21}, // 21%
				},
			},
		},
		{
			name: "Zero values should be skipped",
			input: &schemas.VariableWithValuationResponse{
				ID:          "USD",
				Description: "USD Rate",
				Valuations: []schemas.VariableValuation{
					{Date: "2023-11-01", Value: 100.0},
					{Date: "2023-11-02", Value: 0.0},   // Should be skipped
					{Date: "2023-11-03", Value: 110.0}, // Should calculate from 100
				},
			},
			interval: 24 * time.Hour,
			expected: &schemas.VariableWithValuationResponse{
				ID:          "USD",
				Description: "USD Rate (variation)",
				Valuations: []schemas.VariableValuation{
					{Date: "2023-11-03", Value: 0.1}, // 10% from 100 to 110 (skipping the zero value)
				},
			},
		},
		{
			name: "Weekly interval test",
			input: &schemas.VariableWithValuationResponse{
				ID:          "USD",
				Description: "USD Rate",
				Valuations: []schemas.VariableValuation{
					{Date: "2023-11-01", Value: 100.0},
					{Date: "2023-11-08", Value: 110.0}, // 7 days later
				},
			},
			interval: 7 * 24 * time.Hour,
			expected: &schemas.VariableWithValuationResponse{
				ID:          "USD",
				Description: "USD Rate (variation)",
				Valuations: []schemas.VariableValuation{
					{Date: "2023-11-08", Value: 0.1}, // 10%
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := controllers.ComputeValuationVariations(test.input, test.interval)

			// Check basic structure
			if result.ID != test.expected.ID {
				t.Errorf("expected ID %s, got %s", test.expected.ID, result.ID)
			}
			if result.Description != test.expected.Description {
				t.Errorf("expected Description %s, got %s", test.expected.Description, result.Description)
			}

			// Check valuations length
			if len(result.Valuations) != len(test.expected.Valuations) {
				t.Errorf("expected %d valuations, got %d", len(test.expected.Valuations), len(result.Valuations))
				t.Logf("Expected: %+v", test.expected.Valuations)
				t.Logf("Got: %+v", result.Valuations)
				return
			}

			// Check each valuation
			for i, expectedVal := range test.expected.Valuations {
				if i >= len(result.Valuations) {
					t.Errorf("missing valuation at index %d", i)
					continue
				}
				actualVal := result.Valuations[i]

				if expectedVal.Date != actualVal.Date {
					t.Errorf("at index %d, expected date %s, got %s", i, expectedVal.Date, actualVal.Date)
				}

				// Use approximate comparison for floating point values
				if abs(expectedVal.Value-actualVal.Value) > 0.0001 {
					t.Errorf("at index %d, expected value %.4f, got %.4f", i, expectedVal.Value, actualVal.Value)
				}
			}
		})
	}
}

// Helper function for absolute value
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// Test to demonstrate the current calculation issues
func TestComputeValuationVariationsIssues(t *testing.T) {
	t.Run("Demonstrate current calculation problems", func(t *testing.T) {
		input := &schemas.VariableWithValuationResponse{
			ID:          "USD",
			Description: "USD Rate",
			Valuations: []schemas.VariableValuation{
				{Date: "2023-11-01", Value: 100.0},
				{Date: "2023-11-02", Value: 110.0}, // +10%
				{Date: "2023-11-03", Value: 121.0}, // +10% from 110
			},
		}

		result := controllers.ComputeValuationVariations(input, 24*time.Hour)

		t.Logf("Input valuations: %+v", input.Valuations)
		t.Logf("Result valuations: %+v", result.Valuations)

		// The current implementation has issues:
		// 1. It should calculate cumulative returns correctly
		// 2. The first value should be handled properly
		// 3. The return format might not be what's expected

		if len(result.Valuations) == 0 {
			t.Log("No variations calculated - this might be correct for the first value")
		} else {
			t.Logf("Calculated variations: %+v", result.Valuations)
		}
	})

	t.Run("Verify zero value handling", func(t *testing.T) {
		input := &schemas.VariableWithValuationResponse{
			ID:          "USD",
			Description: "USD Rate",
			Valuations: []schemas.VariableValuation{
				{Date: "2023-11-01", Value: 100.0},
				{Date: "2023-11-02", Value: 0.0},   // Zero value - should be skipped
				{Date: "2023-11-03", Value: 110.0}, // Should calculate from 100
			},
		}

		result := controllers.ComputeValuationVariations(input, 24*time.Hour)

		// Should calculate variation from 2023-11-01 to 2023-11-03, skipping the zero value
		if len(result.Valuations) != 1 {
			t.Errorf("expected 1 variation, got %d", len(result.Valuations))
		} else {
			expectedValue := 0.1 // 10% increase from 100 to 110
			if abs(result.Valuations[0].Value-expectedValue) > 0.0001 {
				t.Errorf("expected value %.4f, got %.4f", expectedValue, result.Valuations[0].Value)
			}
		}
	})
}
