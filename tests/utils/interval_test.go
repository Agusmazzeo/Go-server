package utils_test

import (
	"server/src/utils"
	"testing"
	"time"
)

// Helper function to compare TimeIntervals
func compareTimeIntervals(ti1, ti2 *utils.TimeInterval) bool {
	return ti1.Years == ti2.Years &&
		ti1.Months == ti2.Months &&
		ti1.Weeks == ti2.Weeks &&
		ti1.Days == ti2.Days
}

func TestParseTimeInterval(t *testing.T) {
	tests := []struct {
		input    string
		expected utils.TimeInterval
		hasError bool
	}{
		// Valid cases
		{"0w:0d", utils.TimeInterval{Years: 0, Weeks: 0, Days: 0}, false},
		{"1w:3d", utils.TimeInterval{Years: 0, Weeks: 1, Days: 3}, false},
		{"5w:4d", utils.TimeInterval{Years: 0, Weeks: 5, Days: 4}, false},
		{"0w:10d", utils.TimeInterval{Years: 0, Weeks: 0, Days: 10}, false},
		{"", utils.TimeInterval{Years: 0, Weeks: 0, Days: 0}, false}, // Empty string

		// Invalid cases
		{"1y", utils.TimeInterval{Years: 1, Weeks: 0, Days: 0}, true},
		{"invalid", utils.TimeInterval{}, true},
		{"1y:2m:abc", utils.TimeInterval{}, true},
		{"3x:4d", utils.TimeInterval{}, true},
	}

	for _, tt := range tests {
		result, err := utils.ParseTimeInterval(tt.input)
		if (err != nil) != tt.hasError {
			t.Errorf("ParseTimeInterval(%q) unexpected error: %v, want error: %v", tt.input, err, tt.hasError)
		}

		if err == nil && !compareTimeIntervals(result, &tt.expected) {
			t.Errorf("ParseTimeInterval(%q) = %+v, want %+v", tt.input, result, tt.expected)
		}
	}
}

func TestToDuration(t *testing.T) {
	tests := []struct {
		input    utils.TimeInterval
		expected time.Duration
	}{
		// Valid durations
		{utils.TimeInterval{Years: 0, Months: 0, Weeks: 1, Days: 3}, time.Duration(10*24) * time.Hour}, // 1 week + 3 days = 10 days
		{utils.TimeInterval{Years: 0, Months: 0, Weeks: 0, Days: 7}, time.Duration(7*24) * time.Hour},  // 7 days
		{utils.TimeInterval{Years: 0, Months: 0, Weeks: 0, Days: 0}, time.Duration(0)},                 // 0 days
		{utils.TimeInterval{Years: 0, Months: 0, Weeks: 5, Days: 2}, time.Duration(37*24) * time.Hour}, // 5 weeks + 2 days = 37 days
	}

	for _, tt := range tests {
		result := tt.input.ToDuration()
		if result != tt.expected {
			t.Errorf("ToDuration(%+v) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}
