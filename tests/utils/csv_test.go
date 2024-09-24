package utils_test

import (
	"server/src/utils"
	"testing"
)

func TestCSVToMap(t *testing.T) {
	file := "../test_files/utils/denominaciones.csv"

	denominationMap, err := utils.CSVToMap(file)

	if err != nil {
		t.Fatalf("expected error to be nil: %s", err.Error())
	}

	if len(*denominationMap) != 268 {
		t.Fatalf("expected denomationMap to have at least one value")
	}
}
