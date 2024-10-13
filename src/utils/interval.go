package utils

import (
	"fmt"
	"regexp"
	"strconv"
	"time"
)

// TimeInterval represents a parsed interval with years, months, weeks, and days.
type TimeInterval struct {
	Years  int
	Months int
	Weeks  int
	Days   int
}

// ParseTimeInterval parses a string in the format "1y:2m:1w:3d" and returns a TimeInterval struct.
// It returns an error if the format is invalid.
func ParseTimeInterval(intervalStr string) (*TimeInterval, error) {
	// Define the regex to capture months, weeks, and days with optional colons
	regex := regexp.MustCompile(`^(\d+m)?(:?\d+w)?(:?\d+d)?$`)

	match := regex.FindStringSubmatch(intervalStr)
	if match == nil {
		return nil, fmt.Errorf("invalid format")
	}

	// Initialize values to 0
	months, weeks, days := 0, 0, 0

	// Convert matched values to integers
	if match[1] != "" {
		months, _ = strconv.Atoi(match[1][:len(match[1])-1]) // Remove 'm' and convert
	}
	if match[2] != "" {
		weeks, _ = strconv.Atoi(match[2][1 : len(match[2])-1]) // Remove colon and 'w' and convert
	}
	if match[3] != "" {
		days, _ = strconv.Atoi(match[3][1 : len(match[3])-1]) // Remove colon and 'd' and convert
	}

	return &TimeInterval{Months: months, Weeks: weeks, Days: days}, nil
}

// ToDuration converts the TimeInterval to a time.Duration, ignoring years and months since they vary.
func (ti *TimeInterval) ToDuration() time.Duration {
	// Calculate weeks and days in terms of duration
	totalDays := ti.Days + (ti.Weeks * 7)
	duration := time.Duration(totalDays) * 24 * time.Hour
	return duration
}
