package bcra_test

import (
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

// GetDivisas reads the saved Divisas response from a file and returns the data.
func (c *BCRAServiceClientMock) GetDivisas() (*bcra.GetDivisasResponse, error) {
	filePath := fmt.Sprintf("%s/divisas_response.json", c.mockDataDir)

	// Read saved response from file
	responseBytes, err := utils.ReadResponseFromFile(filePath)
	if err != nil {
		return nil, err
	}

	// Unmarshal the response into the GetDivisasResponse struct
	var divisasResponse bcra.GetDivisasResponse
	err = json.Unmarshal(responseBytes, &divisasResponse)
	if err != nil {
		return nil, err
	}

	return &divisasResponse, nil
}

// GetCotizaciones reads the saved Cotizaciones response from a file and returns the data.
func (c *BCRAServiceClientMock) GetCotizaciones(fecha string) (*bcra.GetCotizacionesResponse, error) {
	filePath := fmt.Sprintf("%s/cotizaciones_response.json", c.mockDataDir)

	// Read saved response from file
	responseBytes, err := utils.ReadResponseFromFile(filePath)
	if err != nil {
		return nil, err
	}

	// Unmarshal the response into the GetCotizacionesResponse struct
	var cotizacionesResponse bcra.GetCotizacionesResponse
	err = json.Unmarshal(responseBytes, &cotizacionesResponse)
	if err != nil {
		return nil, err
	}

	return &cotizacionesResponse, nil
}

// GetCotizacionesPorMoneda reads the saved CotizacionesPorMoneda response from a file and returns the data.
func (c *BCRAServiceClientMock) GetCotizacionesPorMoneda(moneda string, fechaDesde string, fechaHasta string) (*bcra.GetCotizacionesByMonedaResponse, error) {
	filePath := fmt.Sprintf("%s/cotizaciones_por_moneda_response.json", c.mockDataDir)

	// Read saved response from file
	responseBytes, err := utils.ReadResponseFromFile(filePath)
	if err != nil {
		return nil, err
	}

	// Unmarshal the response into the GetCotizacionesByMonedaResponse struct
	var cotizacionesByMonedaResponse bcra.GetCotizacionesByMonedaResponse
	err = json.Unmarshal(responseBytes, &cotizacionesByMonedaResponse)
	if err != nil {
		return nil, err
	}

	return &cotizacionesByMonedaResponse, nil
}
