package utils

import (
	"encoding/csv"
	"fmt"
	"os"
)

// CSVToMap reads a CSV file and returns a map where the Denomination is the key and Classification is the value.
func CSVToMap(filePath string) (*map[string]string, error) {
	// Open the CSV file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open the file: %v", err)
	}
	defer file.Close()

	// Create a CSV reader
	reader := csv.NewReader(file)

	// Read all rows from the CSV
	rows, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read the file: %v", err)
	}

	// Create a map to store the Denomination as key and Classification as value
	data := make(map[string]string)

	// Iterate through the rows and populate the map
	for i, row := range rows {
		if i == 0 {
			// Skip header row
			continue
		}
		// Ensure that the row has at least two columns
		if len(row) < 2 {
			continue
		}

		// Add the key-value pair to the map
		denomination := row[0]
		classification := row[1]
		if classification == "" {
			continue
		}
		data[denomination] = classification
	}

	return &data, nil
}
