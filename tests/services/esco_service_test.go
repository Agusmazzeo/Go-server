package services

import (
	"context"
	"os"
	"path/filepath"
	"server/src/services"
	"server/src/utils"
	esco_test "server/tests/clients/esco"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func setupMockClient(t *testing.T) *esco_test.ESCOServiceClientMock {
	t.Helper()
	workspaceRoot, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current working directory: %v", err)
	}
	// Navigate up to the workspace root if we're in a subdirectory
	for {
		if _, err := os.Stat(filepath.Join(workspaceRoot, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(workspaceRoot)
		if parent == workspaceRoot {
			t.Fatalf("Could not find workspace root directory")
		}
		workspaceRoot = parent
	}

	mockDataDir := filepath.Join(workspaceRoot, "tests", "test_files", "clients", "esco")
	mockClient, err := esco_test.NewMockClient(mockDataDir)
	if err != nil {
		t.Fatalf("Failed to create mock client: %v", err)
	}
	return mockClient
}

func TestGetAccountByID(t *testing.T) {
	ctx := context.Background()
	logger := utils.NewLogger(logrus.InfoLevel, false, "")
	ctx = utils.WithLogger(ctx, logger)

	mockClient := setupMockClient(t)
	service := services.NewESCOService(mockClient)

	account, err := service.GetAccountByID(ctx, "token", "4014D4EFDD5DE27B")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if account.ID != "4014D4EFDD5DE27B" {
		t.Errorf("Expected client ID '4014D4EFDD5DE27B', got %s", account.ID)
	}
	if account.FI != "38B198EB5EB4A638" {
		t.Errorf("Expected FI '38B198EB5EB4A638', got %s", account.FI)
	}
	if account.N != 10569 {
		t.Errorf("Expected N 10569, got %d", account.N)
	}
}

func TestGetAccountState(t *testing.T) {
	ctx := context.Background()
	logger := utils.NewLogger(logrus.InfoLevel, false, "")
	ctx = utils.WithLogger(ctx, logger)
	date := time.Date(2024, 10, 8, 0, 0, 0, 0, time.UTC)

	mockClient := setupMockClient(t)
	service := services.NewESCOService(mockClient)

	state, err := service.GetAccountState(ctx, "token", "4014D4EFDD5DE27B", date)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if state.Assets == nil {
		t.Fatal("Expected assets map to be initialized")
	}

	// Verify YPF asset
	asset, exists := (*state.Assets)["YMCQO"]
	if !exists {
		t.Fatal("Expected asset for YMCQO to exist")
	}

	if len(asset.Holdings) != 1 {
		t.Errorf("Expected 1 holding, got %d", len(asset.Holdings))
	}

	holding := asset.Holdings[0]
	if holding.Currency != "Pesos" {
		t.Errorf("Expected currency Pesos, got %s", holding.Currency)
	}
	if holding.Value != 222222222.64 {
		t.Errorf("Expected value 222222222.64, got %f", holding.Value)
	}
	if holding.Units != 509694310.0 {
		t.Errorf("Expected units 509694310.0, got %f", holding.Units)
	}
}

func TestGetAccountStateWithTransactions(t *testing.T) {
	ctx := context.Background()
	logger := utils.NewLogger(logrus.InfoLevel, false, "")
	ctx = utils.WithLogger(ctx, logger)
	startDate := time.Date(2024, 10, 8, 0, 0, 0, 0, time.UTC)
	endDate := startDate.Add(24 * time.Hour)

	mockClient := setupMockClient(t)
	service := services.NewESCOService(mockClient)

	state, err := service.GetAccountStateWithTransactions(ctx, "token", "4014D4EFDD5DE27B", startDate, endDate, 24*time.Hour)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if state.Assets == nil {
		t.Fatal("Expected assets map to be initialized")
	}

	// Verify YPF asset
	asset, exists := (*state.Assets)["YMCQO"]
	if !exists {
		t.Fatal("Expected asset for YMCQO to exist")
	}

	if len(asset.Holdings) != 2 {
		t.Errorf("Expected 2 holdings, got %d", len(asset.Holdings))
	}

	holding := asset.Holdings[0]
	if holding.Currency != "Pesos" {
		t.Errorf("Expected currency Pesos, got %s", holding.Currency)
	}
	if holding.Value != 222222222.64 {
		t.Errorf("Expected value 222222222.64, got %f", holding.Value)
	}
	if holding.Units != 509694310.0 {
		t.Errorf("Expected units 509694310.0, got %f", holding.Units)
	}

	// Verify transactions
	if len(asset.Transactions) != 0 {
		t.Errorf("Expected transactions to not exist")
	}
}

func TestGetMultiAccountStateByCategory(t *testing.T) {
	ctx := context.Background()
	logger := utils.NewLogger(logrus.InfoLevel, false, "")
	ctx = utils.WithLogger(ctx, logger)
	startDate := time.Date(2024, 10, 8, 0, 0, 0, 0, time.UTC)
	endDate := startDate.Add(24 * time.Hour)

	mockClient := setupMockClient(t)
	service := services.NewESCOService(mockClient)

	state, err := service.GetMultiAccountStateByCategory(ctx, "token", []string{"4014D4EFDD5DE27B"}, startDate, endDate, 24*time.Hour)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if state.AssetsByCategory == nil {
		t.Fatal("Expected assets by category map to be initialized")
	}

	// Verify tasa fija category
	tasaFija, exists := (*state.AssetsByCategory)["TASA FIJA"]
	if !exists {
		t.Fatal("Expected assets for Tasa Fija category to exist")
	}

	if len(tasaFija) != 1 {
		t.Errorf("Expected 1 asset in Tasa Fija category, got %d", len(tasaFija))
	}

}
