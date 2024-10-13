package utils

import (
	"fmt"
	"time"
)

func GenerateDates(startDate, endDate time.Time, interval time.Duration) ([]time.Time, error) {
	// Ensure the start date is before the end date
	if endDate.Before(startDate) {
		return nil, fmt.Errorf("endDate must be after startDate")
	}

	// Initialize the array to hold the dates
	var dates []time.Time

	// Loop and add intervals to the start date until we surpass the end date
	for currentDate := startDate; currentDate.Before(endDate) || currentDate.Equal(endDate); currentDate = currentDate.Add(interval) {
		dates = append(dates, currentDate)
	}

	return dates, nil
}
