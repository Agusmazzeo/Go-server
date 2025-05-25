package services_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"server/src/models"
	"server/src/repositories"
	"server/src/schemas"
	"server/src/services"
	"testing"
	"time"

	"server/tests/init_test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func loadTestData(t *testing.T, filename string, v interface{}) {
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
	filePath := filepath.Join(workspaceRoot, "tests", "test_files", "services", "report_service", filename)
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read test data file %s: %v", filename, err)
	}

	err = json.Unmarshal(data, v)
	if err != nil {
		t.Fatalf("Failed to unmarshal test data from %s: %v", filename, err)
	}
}

func TestGenerateReport(t *testing.T) {
	// Setup test database connection
	db := init_test.SetupTestDB(t)
	defer init_test.TruncateTables(t, db)

	// Create repository instances
	holdingRepo := repositories.NewHoldingRepository(db)
	assetRepo := repositories.NewAssetRepository(db)
	assetCategoryRepo := repositories.NewAssetCategoryRepository(db)
	transactionRepo := repositories.NewTransactionRepository(db)

	service := services.NewReportService(
		holdingRepo,
		assetRepo,
		assetCategoryRepo,
		transactionRepo,
	)

	ctx := context.Background()
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	interval := 24 * time.Hour

	// Load test data from JSON files
	var mockAssets []models.Asset
	var mockCategories []models.AssetCategory
	var mockHoldings []models.Holding
	var mockTransactions []models.Transaction

	loadTestData(t, "assets.json", &mockAssets)
	loadTestData(t, "categories.json", &mockCategories)
	loadTestData(t, "holdings.json", &mockHoldings)
	loadTestData(t, "transactions.json", &mockTransactions)

	// Insert test data into the real database
	for i := range mockCategories {
		err := assetCategoryRepo.Create(ctx, &mockCategories[i], nil)
		require.NoError(t, err)
	}
	for i := range mockAssets {
		mockAssets[i].CategoryID = mockCategories[i].ID
		err := assetRepo.Create(ctx, &mockAssets[i], nil)
		require.NoError(t, err)
	}
	for i := range mockHoldings {
		mockHoldings[i].AssetID = mockAssets[0].ID
		err := holdingRepo.Create(ctx, &mockHoldings[i], nil)
		require.NoError(t, err)
	}
	for i := range mockTransactions {
		mockTransactions[i].AssetID = mockAssets[0].ID
		err := transactionRepo.Create(ctx, &mockTransactions[i], nil)
		require.NoError(t, err)
	}

	// Execute
	report, err := service.GenerateReport(ctx, startDate, endDate, interval)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, report)
	assert.NotNil(t, report.AssetsByCategory)
	assert.NotNil(t, report.CategoryAssets)
	assert.NotNil(t, report.TotalHoldingsByDate)
	assert.NotNil(t, report.TotalTransactionsByDate)

	// Verify the report structure
	stocks, exists := (*report.AssetsByCategory)["STOCKS"]
	assert.True(t, exists)
	assert.Len(t, stocks, 1)
	assert.Equal(t, "STOCK", stocks[0].Type)
	assert.Equal(t, "USD", stocks[0].Denomination)

	// Verify holdings
	assert.Len(t, stocks[0].Holdings, 2)
	assert.Equal(t, 1000.0, stocks[0].Holdings[0].Value)
	assert.Equal(t, 1100.0, stocks[0].Holdings[1].Value)

	// Verify transactions
	assert.Len(t, stocks[0].Transactions, 1)
	assert.Equal(t, 100.0, stocks[0].Transactions[0].Value)
}

func TestCalculateAssetReturn(t *testing.T) {
	service := &services.ReportService{}
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)

	asset := schemas.Asset{
		ID:           "1",
		Category:     "STOCKS",
		Type:         "STOCK",
		Denomination: "USD",
		Holdings: []schemas.Holding{
			{
				DateRequested: &startDate,
				Value:         1000.0,
				Units:         10,
			},
			{
				DateRequested: &endDate,
				Value:         1100.0,
				Units:         10,
			},
		},
		Transactions: []schemas.Transaction{
			{
				Date:  &startDate,
				Value: 100.0,
				Units: 1,
			},
		},
	}

	// Test successful calculation
	returns, err := service.CalculateAssetReturn(asset, 24*time.Hour)
	assert.NoError(t, err)
	assert.NotNil(t, returns)
	assert.Equal(t, "1", returns.ID)
	assert.Equal(t, "STOCK", returns.Type)
	assert.Equal(t, "USD", returns.Denomination)
	assert.Equal(t, "STOCKS", returns.Category)
	assert.Len(t, returns.ReturnsByDateRange, 2)
	assert.InDelta(t, 10.0, returns.ReturnsByDateRange[0].ReturnPercentage, 0.1)

	// Test insufficient holdings
	asset.Holdings = asset.Holdings[:1]
	returns, err = service.CalculateAssetReturn(asset, 24*time.Hour)
	assert.Error(t, err)
	assert.Empty(t, returns)
}

func TestFilterHoldingsByInterval(t *testing.T) {
	service := &services.ReportService{}
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC)
	interval := 24 * time.Hour

	holdings := []schemas.Holding{
		{
			DateRequested: &startDate,
			Value:         1000.0,
			Units:         10,
		},
		{
			DateRequested: &endDate,
			Value:         1100.0,
			Units:         10,
		},
	}

	// Test filtering
	filtered := service.FilterHoldingsByInterval(holdings, startDate, endDate, interval)
	assert.Len(t, filtered, 2)
	assert.Equal(t, startDate, *filtered[0].DateRequested)
	assert.Equal(t, endDate, *filtered[1].DateRequested)

	// Test with different interval
	midDate := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	holdings = append(holdings, schemas.Holding{
		DateRequested: &midDate,
		Value:         1050.0,
		Units:         10,
	})
	filtered = service.FilterHoldingsByInterval(holdings, startDate, endDate, interval)
	assert.Len(t, filtered, 3)
}

func TestCalculateFinalIntervalReturn(t *testing.T) {
	service := &services.ReportService{}
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)

	returns := []schemas.ReturnByDate{
		{
			StartDate:        startDate,
			EndDate:          endDate,
			ReturnPercentage: 10.0,
		},
		{
			StartDate:        endDate,
			EndDate:          endDate.Add(24 * time.Hour),
			ReturnPercentage: 5.0,
		},
	}

	// Test calculation
	finalReturn := service.CalculateFinalIntervalReturn(returns)
	expectedReturn := 1.155 // (1 + 0.1) * (1 + 0.05)
	assert.InDelta(t, expectedReturn, finalReturn, 0.001)
}
