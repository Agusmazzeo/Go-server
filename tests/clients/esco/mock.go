package esco_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"server/src/clients/esco"
	"server/src/schemas"
	"server/src/utils"
	"time"
)

type ESCOServiceClientMock struct {
	mockDataDir string
	categoryMap *map[string]string
}

// NewMockClient creates a new instance of ESCOServiceClientMock.
func NewMockClient(mockDataDir string) (*ESCOServiceClientMock, error) {
	categoryMap, err := utils.CSVToMap(fmt.Sprintf("%s/denominaciones.csv", mockDataDir))
	if err != nil {
		return nil, err
	}
	return &ESCOServiceClientMock{
		mockDataDir: mockDataDir, categoryMap: categoryMap,
	}, nil
}

// ReadMockResponse reads a mock response from a JSON file.
func (c *ESCOServiceClientMock) ReadMockResponse(fileName string, v interface{}) error {
	filePath := fmt.Sprintf("%s/%s", c.mockDataDir, fileName)
	responseBytes, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	return json.Unmarshal(responseBytes, &v)
}

// PostToken reads a saved token response from a mock file.
func (c *ESCOServiceClientMock) PostToken(_ context.Context, _, _ string) (*schemas.TokenResponse, error) {
	var tokenResponse schemas.TokenResponse
	err := c.ReadMockResponse("token_response.json", &tokenResponse)
	if err != nil {
		return nil, err
	}
	return &tokenResponse, nil
}

// BuscarCuentas reads saved account data from a mock file.
func (c *ESCOServiceClientMock) BuscarCuentas(_, filter string) ([]esco.CuentaSchema, error) {
	var cuentas []esco.CuentaSchema
	err := c.ReadMockResponse("cuentas_response.json", &cuentas)
	if err != nil {
		return nil, err
	}
	if filter != "*" {
		return cuentas[0:1], nil
	}
	return cuentas, nil
}

// GetCuentaDetalle reads detailed account information from a mock file.
func (c *ESCOServiceClientMock) GetCuentaDetalle(_, cid string) (*esco.CuentaDetalleSchema, error) {
	var cuentaDetalle esco.CuentaDetalleSchema
	err := c.ReadMockResponse(fmt.Sprintf("cuenta_detalle_%s_response.json", cid), &cuentaDetalle)
	if err != nil {
		return nil, err
	}
	return &cuentaDetalle, nil
}

// GetEstadoCuenta reads account status information from a mock file.
func (c *ESCOServiceClientMock) GetEstadoCuenta(_, cid, _, _, _ string, _ time.Time) ([]esco.EstadoCuentaSchema, error) {
	var estadoCuenta []esco.EstadoCuentaSchema
	err := c.ReadMockResponse(fmt.Sprintf("estado_cuenta_%s_date_response.json", cid), &estadoCuenta)
	if err != nil {
		return nil, err
	}
	return estadoCuenta, nil
}

// GetCategoryMap returns a mocked category map.
func (c *ESCOServiceClientMock) GetCategoryMap() map[string]string {
	return *c.categoryMap
}
