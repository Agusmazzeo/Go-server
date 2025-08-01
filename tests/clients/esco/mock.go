package esco_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"server/src/clients/esco"
	"server/src/schemas"
	"server/src/services"
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
func (c *ESCOServiceClientMock) GetEstadoCuenta(_, cid, _, _, _ string, _ time.Time, _ bool) ([]esco.EstadoCuentaSchema, error) {
	var estadoCuenta []esco.EstadoCuentaSchema
	err := c.ReadMockResponse(fmt.Sprintf("estado_cuenta_%s_date_response.json", cid), &estadoCuenta)
	if err != nil {
		return nil, err
	}
	return estadoCuenta, nil
}

// GetLiquidaciones reads account liquidaciones information from a mock file.
func (c *ESCOServiceClientMock) GetLiquidaciones(token, cid, fid, nncc, tf string, startDate, endDate time.Time, _ bool) ([]esco.Liquidacion, error) {
	var liquidaciones []esco.Liquidacion
	err := c.ReadMockResponse("liquidaciones_response.json", &liquidaciones)
	if err != nil {
		return nil, err
	}
	return liquidaciones, nil
}

// GetBoletos reads account boletos information from a mock file.
func (c *ESCOServiceClientMock) GetBoletos(token, cid, fid, nncc, tf string, startDate, endDate time.Time, _ bool) ([]esco.Boleto, error) {
	var boletos []esco.Boleto
	err := c.ReadMockResponse("boletos_response.json", &boletos)
	if err != nil {
		return nil, err
	}
	return boletos, nil
}

// GetCtaCteConsolidado reads account cte corriente information from a mock file.
func (c *ESCOServiceClientMock) GetCtaCteConsolidado(token, cid, fid, nncc, tf string, startDate, endDate time.Time, _ bool) ([]esco.Instrumentos, error) {
	var instrumentos []esco.Instrumentos
	err := c.ReadMockResponse("cte_corriente_response.json", &instrumentos)
	if err != nil {
		return nil, err
	}
	return instrumentos, nil
}

// GetCategoryMap returns a mocked category map.
func (c *ESCOServiceClientMock) GetCategoryMap() map[string]string {
	return *c.categoryMap
}

// MockESCOService is a mock implementation of services.ESCOServiceI
type MockESCOService struct {
	services.ESCOServiceI
	getAccountStateFunc func(ctx context.Context, token, accountID string, startDate, endDate time.Time, interval time.Duration) (*schemas.AccountState, error)
}

// NewMockESCOService creates a new instance of MockESCOService
func NewMockESCOService(getAccountStateFunc func(ctx context.Context, token, accountID string, startDate, endDate time.Time, interval time.Duration) (*schemas.AccountState, error)) *MockESCOService {
	return &MockESCOService{
		getAccountStateFunc: getAccountStateFunc,
	}
}

// GetAccountStateWithTransactions implements the ESCOServiceI interface
func (m *MockESCOService) GetAccountStateWithTransactions(ctx context.Context, token, accountID string, startDate, endDate time.Time, interval time.Duration) (*schemas.AccountState, error) {
	return m.getAccountStateFunc(ctx, token, accountID, startDate, endDate, interval)
}
