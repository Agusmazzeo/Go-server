package bcra_test

import (
	"context"
	"encoding/json"
	"fmt"
	"server/src/clients/bcra"
	"server/src/utils"
)

// BCRAServiceClientMock is a mock implementation of BCRAServiceClientI
// that reads from saved JSON files instead of making actual API calls.
type BCRAServiceClientMock struct {
	mockDataDir string
}

// NewMockClient creates a new instance of BCRAServiceClientMock.
func NewMockClient(mockDataDir string) (*BCRAServiceClientMock, error) {
	return &BCRAServiceClientMock{
		mockDataDir: mockDataDir,
	}, nil
}

// GetCotizaciones reads the saved Cotizaciones response from a file and returns the data.
func (c *BCRAServiceClientMock) GetVariables(_ context.Context) (*bcra.GetVariablesResponse, error) {
	filePath := fmt.Sprintf("%s/variables_response.json", c.mockDataDir)

	// Read saved response from file
	responseBytes, err := utils.ReadResponseFromFile(filePath)
	if err != nil {
		return nil, err
	}

	// Unmarshal the response into the GetCotizacionesResponse struct
	var variablesResponse bcra.GetVariablesResponse
	err = json.Unmarshal(responseBytes, &variablesResponse)
	if err != nil {
		return nil, err
	}

	return &variablesResponse, nil
}

// GetCotizacionesPorMoneda reads the saved CotizacionesPorMoneda response from a file and returns the data.
func (c *BCRAServiceClientMock) GetVariablesPorFecha(_ context.Context, _ string, _ string, _ string) (*bcra.GetVariablesResponse, error) {
	filePath := fmt.Sprintf("%s/variable_por_fecha_response.json", c.mockDataDir)

	// Read saved response from file
	responseBytes, err := utils.ReadResponseFromFile(filePath)
	if err != nil {
		return nil, err
	}

	// Unmarshal the response into the GetCotizacionesByMonedaResponse struct
	var variablesPorFecha bcra.GetVariablesResponse
	err = json.Unmarshal(responseBytes, &variablesPorFecha)
	if err != nil {
		return nil, err
	}

	return &variablesPorFecha, nil
}
