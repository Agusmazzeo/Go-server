package services

import (
	"context"
	"server/src/clients/esco"
	"server/src/services"
	"server/src/utils"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

type mockESCOServiceClient struct {
	esco.ESCOServiceClientI
	buscarCuentasFunc        func(token, id string) ([]esco.CuentaSchema, error)
	getEstadoCuentaFunc      func(token, clientID, fi, n, c string, date time.Time, consolidated bool) ([]esco.EstadoCuentaSchema, error)
	getLiquidacionesFunc     func(token, clientID, fi, n, c string, startDate, endDate time.Time, consolidated bool) ([]esco.Liquidacion, error)
	getBoletosFunc           func(token, clientID, fi, n, c string, startDate, endDate time.Time, consolidated bool) ([]esco.Boleto, error)
	getCtaCteConsolidadoFunc func(token, clientID, fi, n, c string, startDate, endDate time.Time, consolidated bool) ([]esco.Instrumentos, error)
	getCategoryMapFunc       func() map[string]string
}

func (m *mockESCOServiceClient) BuscarCuentas(token, id string) ([]esco.CuentaSchema, error) {
	return m.buscarCuentasFunc(token, id)
}

func (m *mockESCOServiceClient) GetEstadoCuenta(token, clientID, fi, n, c string, date time.Time, consolidated bool) ([]esco.EstadoCuentaSchema, error) {
	return m.getEstadoCuentaFunc(token, clientID, fi, n, c, date, consolidated)
}

func (m *mockESCOServiceClient) GetLiquidaciones(token, clientID, fi, n, c string, startDate, endDate time.Time, consolidated bool) ([]esco.Liquidacion, error) {
	return m.getLiquidacionesFunc(token, clientID, fi, n, c, startDate, endDate, consolidated)
}

func (m *mockESCOServiceClient) GetBoletos(token, clientID, fi, n, c string, startDate, endDate time.Time, consolidated bool) ([]esco.Boleto, error) {
	return m.getBoletosFunc(token, clientID, fi, n, c, startDate, endDate, consolidated)
}

func (m *mockESCOServiceClient) GetCtaCteConsolidado(token, clientID, fi, n, c string, startDate, endDate time.Time, consolidated bool) ([]esco.Instrumentos, error) {
	return m.getCtaCteConsolidadoFunc(token, clientID, fi, n, c, startDate, endDate, consolidated)
}

func (m *mockESCOServiceClient) GetCategoryMap() map[string]string {
	return m.getCategoryMapFunc()
}

func NewESCOService(client esco.ESCOServiceClientI) *services.ESCOService {
	return services.NewESCOService(client)
}

func TestGetAccountByID(t *testing.T) {
	ctx := context.Background()
	logger := utils.NewLogger(logrus.InfoLevel, false, "")
	ctx = utils.WithLogger(ctx, logger)

	mockClient := &mockESCOServiceClient{
		buscarCuentasFunc: func(token, id string) ([]esco.CuentaSchema, error) {
			return []esco.CuentaSchema{
				{
					ID: "test-client",
					FI: "test-fi",
					N:  123,
					D:  "Test Account",
				},
			}, nil
		},
	}

	service := NewESCOService(mockClient)

	account, err := service.GetAccountByID(ctx, "token", "test-id")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if account.ID != "test-client" {
		t.Errorf("Expected client ID 'test-client', got %s", account.ID)
	}
	if account.FI != "test-fi" {
		t.Errorf("Expected FI 'test-fi', got %s", account.FI)
	}
	if account.N != 123 {
		t.Errorf("Expected N 123, got %d", account.N)
	}
}

func TestGetAccountState(t *testing.T) {
	ctx := context.Background()
	logger := utils.NewLogger(logrus.InfoLevel, false, "")
	ctx = utils.WithLogger(ctx, logger)
	date := time.Now()

	mockClient := &mockESCOServiceClient{
		buscarCuentasFunc: func(token, id string) ([]esco.CuentaSchema, error) {
			return []esco.CuentaSchema{
				{
					ID: "test-client",
					FI: "test-fi",
					N:  123,
					D:  "Test Account",
				},
			}, nil
		},
		getEstadoCuentaFunc: func(token, clientID, fi, n, c string, date time.Time, consolidated bool) ([]esco.EstadoCuentaSchema, error) {
			return []esco.EstadoCuentaSchema{
				{
					A:  "test-asset",
					D:  "Test Asset",
					TI: "STOCK",
					M:  "USD",
					MS: "$",
					C:  100,
					N:  1000,
					F:  date.Format(utils.ShortSlashDateLayout),
				},
			}, nil
		},
		getCategoryMapFunc: func() map[string]string {
			return map[string]string{
				"test-asset - Test Asset": "Stocks",
			}
		},
	}

	service := NewESCOService(mockClient)

	state, err := service.GetAccountState(ctx, "token", "test-id", date)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if state.Vouchers == nil {
		t.Fatal("Expected vouchers map to be initialized")
	}

	voucher, exists := (*state.Vouchers)["test-asset"]
	if !exists {
		t.Fatal("Expected voucher for test-asset to exist")
	}

	if len(voucher.Holdings) != 1 {
		t.Errorf("Expected 1 holding, got %d", len(voucher.Holdings))
	}

	holding := voucher.Holdings[0]
	if holding.Currency != "USD" {
		t.Errorf("Expected currency USD, got %s", holding.Currency)
	}
	if holding.Value != 1000 {
		t.Errorf("Expected value 1000, got %f", holding.Value)
	}
	if holding.Units != 100 {
		t.Errorf("Expected units 100, got %f", holding.Units)
	}
}

func TestGetAccountStateWithTransactions(t *testing.T) {
	ctx := context.Background()
	logger := utils.NewLogger(logrus.InfoLevel, false, "")
	ctx = utils.WithLogger(ctx, logger)
	startDate := time.Now()
	endDate := startDate.Add(24 * time.Hour)

	mockClient := &mockESCOServiceClient{
		buscarCuentasFunc: func(token, id string) ([]esco.CuentaSchema, error) {
			return []esco.CuentaSchema{
				{
					ID: "test-client",
					FI: "test-fi",
					N:  123,
					D:  "Test Account",
				},
			}, nil
		},
		getEstadoCuentaFunc: func(token, clientID, fi, n, c string, date time.Time, consolidated bool) ([]esco.EstadoCuentaSchema, error) {
			return []esco.EstadoCuentaSchema{
				{
					A:  "test-asset",
					D:  "Test Asset",
					TI: "STOCK",
					M:  "USD",
					MS: "$",
					C:  100,
					N:  1000,
					F:  date.Format(utils.ShortSlashDateLayout),
				},
			}, nil
		},
		getLiquidacionesFunc: func(token, clientID, fi, n, c string, startDate, endDate time.Time, consolidated bool) ([]esco.Liquidacion, error) {
			return []esco.Liquidacion{
				{
					F:  "test-asset",
					FL: startDate.Format(utils.ShortSlashDateLayout),
					I:  500,
					Q:  50,
					MS: "$",
				},
			}, nil
		},
		getBoletosFunc: func(token, clientID, fi, n, c string, startDate, endDate time.Time, consolidated bool) ([]esco.Boleto, error) {
			return []esco.Boleto{
				{
					I:  "test-asset",
					FL: startDate.Format(utils.ShortSlashDateLayout),
					N:  300,
					C:  30,
					NS: "$",
					T:  "BUY",
				},
			}, nil
		},
		getCtaCteConsolidadoFunc: func(token, clientID, fi, n, c string, startDate, endDate time.Time, consolidated bool) ([]esco.Instrumentos, error) {
			return []esco.Instrumentos{
				{
					I:    "Test Asset - test-asset",
					FL:   startDate.Format(utils.ShortSlashDateLayout),
					C:    -20,
					N:    200,
					PR_S: "$",
					D:    "Retiro de Títulos",
				},
			}, nil
		},
		getCategoryMapFunc: func() map[string]string {
			return map[string]string{
				"Test Asset - test-asset": "Stocks",
			}
		},
	}

	service := NewESCOService(mockClient)

	state, err := service.GetAccountStateWithTransactions(ctx, "token", "test-id", startDate, endDate, 24*time.Hour)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if state.Vouchers == nil {
		t.Fatal("Expected vouchers map to be initialized")
	}

	voucher, exists := (*state.Vouchers)["test-asset"]
	if !exists {
		t.Fatal("Expected voucher for test-asset to exist")
	}

	if len(voucher.Transactions) != 3 {
		t.Errorf("Expected 3 transactions, got %d", len(voucher.Transactions))
	}

	// Verify transaction values
	var totalValue float64
	for _, transaction := range voucher.Transactions {
		totalValue += transaction.Value
	}
	expectedTotal := -500.0 + -300.0 + 200.0
	if totalValue != expectedTotal {
		t.Errorf("Expected total transaction value %f, got %f", expectedTotal, totalValue)
	}
}

func TestGetMultiAccountStateByCategory(t *testing.T) {
	ctx := context.Background()
	logger := utils.NewLogger(logrus.InfoLevel, false, "")
	ctx = utils.WithLogger(ctx, logger)
	startDate := time.Now()
	endDate := startDate.Add(24 * time.Hour)

	mockClient := &mockESCOServiceClient{
		buscarCuentasFunc: func(token, id string) ([]esco.CuentaSchema, error) {
			return []esco.CuentaSchema{
				{
					ID: "test-client",
					FI: "test-fi",
					N:  123,
					D:  "Test Account",
				},
			}, nil
		},
		getEstadoCuentaFunc: func(token, clientID, fi, n, c string, date time.Time, consolidated bool) ([]esco.EstadoCuentaSchema, error) {
			return []esco.EstadoCuentaSchema{
				{
					A:  "test-asset",
					D:  "Test Asset",
					TI: "STOCK",
					M:  "USD",
					MS: "$",
					C:  100,
					N:  1000,
					F:  date.Format(utils.ShortSlashDateLayout),
				},
			}, nil
		},
		getLiquidacionesFunc: func(token, clientID, fi, n, c string, startDate, endDate time.Time, consolidated bool) ([]esco.Liquidacion, error) {
			return []esco.Liquidacion{
				{
					F:  "test-asset",
					FL: startDate.Format(utils.ShortSlashDateLayout),
					I:  500,
					Q:  50,
					MS: "$",
				},
			}, nil
		},
		getBoletosFunc: func(token, clientID, fi, n, c string, startDate, endDate time.Time, consolidated bool) ([]esco.Boleto, error) {
			return []esco.Boleto{
				{
					I:  "test-asset",
					FL: startDate.Format(utils.ShortSlashDateLayout),
					N:  300,
					C:  30,
					NS: "$",
					T:  "BUY",
				},
			}, nil
		},
		getCtaCteConsolidadoFunc: func(token, clientID, fi, n, c string, startDate, endDate time.Time, consolidated bool) ([]esco.Instrumentos, error) {
			return []esco.Instrumentos{
				{
					I:    "Test Asset - test-asset",
					FL:   startDate.Format(utils.ShortSlashDateLayout),
					C:    -20,
					N:    200,
					PR_S: "$",
					D:    "Retiro de Títulos",
				},
			}, nil
		},
		getCategoryMapFunc: func() map[string]string {
			return map[string]string{
				"Test Asset - test-asset": "Stocks",
				"test-asset - Test Asset": "Stocks",
			}
		},
	}

	service := NewESCOService(mockClient)

	state, err := service.GetMultiAccountStateByCategory(ctx, "token", []string{"test-id"}, startDate, endDate, 24*time.Hour)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if state.VouchersByCategory == nil {
		t.Fatal("Expected vouchers by category map to be initialized")
	}

	vouchers, exists := (*state.VouchersByCategory)["Stocks"]
	if !exists {
		t.Fatal("Expected vouchers for Stocks category to exist")
	}

	if len(vouchers) != 1 {
		t.Errorf("Expected 1 voucher in Stocks category, got %d", len(vouchers))
	}

	voucher := vouchers[0]
	if voucher.Category != "Stocks" {
		t.Errorf("Expected category Stocks, got %s", voucher.Category)
	}
	if len(voucher.Holdings) != 2 {
		t.Errorf("Expected 2 holdings, got %d", len(voucher.Holdings))
	}
}
