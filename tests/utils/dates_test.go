package utils_test

import (
	"server/src/utils"
	"testing"
	"time"
)

func TestGenerateDates(t *testing.T) {
	tests := []struct {
		startDate   time.Time
		endDate     time.Time
		interval    time.Duration
		expected    []time.Time
		expectError bool
	}{
		{
			startDate: time.Date(2024, 8, 1, 0, 0, 0, 0, time.UTC),
			endDate:   time.Date(2024, 8, 31, 0, 0, 0, 0, time.UTC),
			interval:  7 * 24 * time.Hour, // Every 7 days (1 week)
			expected: []time.Time{
				time.Date(2024, 8, 1, 0, 0, 0, 0, time.UTC),
				time.Date(2024, 8, 8, 0, 0, 0, 0, time.UTC),
				time.Date(2024, 8, 15, 0, 0, 0, 0, time.UTC),
				time.Date(2024, 8, 22, 0, 0, 0, 0, time.UTC),
				time.Date(2024, 8, 29, 0, 0, 0, 0, time.UTC),
			},
			expectError: false,
		},
		{
			startDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			endDate:   time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC),
			interval:  2 * 24 * time.Hour, // Every 2 days
			expected: []time.Time{
				time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC),
				time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC),
				time.Date(2024, 1, 7, 0, 0, 0, 0, time.UTC),
				time.Date(2024, 1, 9, 0, 0, 0, 0, time.UTC),
			},
			expectError: false,
		},
		{
			startDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			endDate:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			interval:  1 * 24 * time.Hour, // Interval of 1 day, but start and end are the same
			expected: []time.Time{
				time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			expectError: false,
		},
		{
			startDate:   time.Date(2024, 8, 31, 0, 0, 0, 0, time.UTC),
			endDate:     time.Date(2024, 8, 1, 0, 0, 0, 0, time.UTC),
			interval:    7 * 24 * time.Hour, // Invalid case: endDate is before startDate
			expected:    nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		result, err := utils.GenerateDates(tt.startDate, tt.endDate, tt.interval)

		// Check if we expect an error and if one occurred
		if tt.expectError && err == nil {
			t.Errorf("Expected error, but got none")
		}
		if !tt.expectError && err != nil {
			t.Errorf("Did not expect an error, but got one: %v", err)
		}

		// If there's no error, validate the generated dates
		if !tt.expectError {
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d dates, but got %d", len(tt.expected), len(result))
			}

			// Check that the generated dates match the expected dates
			for i := range tt.expected {
				if !result[i].Equal(tt.expected[i]) {
					t.Errorf("Expected date %v, but got %v", tt.expected[i], result[i])
				}
			}
		}
	}
}
