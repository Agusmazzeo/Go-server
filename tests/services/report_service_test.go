package services_test

import (
	"context"
	"encoding/json"
	"fmt"
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

	// Create repository instances
	holdingRepo := repositories.NewHoldingRepository(db)
	assetRepo := repositories.NewAssetRepository(db)
	assetCategoryRepo := repositories.NewAssetCategoryRepository(db)
	transactionRepo := repositories.NewTransactionRepository(db)

	service := services.NewReportService()

	ctx := context.Background()
	startDate := time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC)
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

	// Make category names unique to avoid conflicts
	for i := range mockCategories {
		mockCategories[i].Name = fmt.Sprintf("Test_%s_%d", mockCategories[i].Name, time.Now().UnixNano())
	}

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
		mockHoldings[i].ClientID = "test-client"
		err := holdingRepo.Create(ctx, &mockHoldings[i], nil)
		require.NoError(t, err)
	}
	for i := range mockTransactions {
		mockTransactions[i].ClientID = "test-client"
		mockTransactions[i].AssetID = mockAssets[0].ID
		err := transactionRepo.Create(ctx, &mockTransactions[i], nil)
		require.NoError(t, err)
	}

	// Create AccountStateByCategory from the test data
	assetsByCategory := make(map[string][]schemas.Asset)
	categoryAssets := make(map[string]schemas.Asset)
	totalHoldingsByDate := make(map[string]schemas.Holding)
	totalTransactionsByDate := make(map[string]schemas.Transaction)

	// Create schema assets from the test data
	schemaAsset := schemas.Asset{
		ID:           fmt.Sprintf("%d", mockAssets[0].ID),
		Category:     mockCategories[0].Name,
		Type:         mockAssets[0].AssetType,
		Denomination: mockAssets[0].Currency,
		Holdings:     make([]schemas.Holding, 0),
		Transactions: make([]schemas.Transaction, 0),
	}

	// Add holdings to the asset
	for _, holding := range mockHoldings {
		schemaAsset.Holdings = append(schemaAsset.Holdings, schemas.Holding{
			DateRequested: &holding.Date,
			Value:         holding.Value,
			Units:         holding.Units,
		})
	}

	// Add transactions to the asset
	for _, transaction := range mockTransactions {
		schemaAsset.Transactions = append(schemaAsset.Transactions, schemas.Transaction{
			Date:  &transaction.Date,
			Value: transaction.TotalValue,
			Units: transaction.Units,
		})
	}

	assetsByCategory[mockCategories[0].Name] = append(assetsByCategory[mockCategories[0].Name], schemaAsset)

	// Create category asset
	categoryAssets[mockCategories[0].Name] = schemas.Asset{
		ID:           mockCategories[0].Name,
		Category:     mockCategories[0].Name,
		Type:         "CATEGORY",
		Denomination: mockAssets[0].Currency,
		Holdings:     make([]schemas.Holding, 0),
		Transactions: make([]schemas.Transaction, 0),
	}

	// Create total holdings by date
	for _, holding := range mockHoldings {
		dateStr := holding.Date.Format("2006-01-02")
		totalHoldingsByDate[dateStr] = schemas.Holding{
			DateRequested: &holding.Date,
			Value:         holding.Value,
			Units:         holding.Units,
		}
	}

	// Create total transactions by date
	for _, transaction := range mockTransactions {
		dateStr := transaction.Date.Format("2006-01-02")
		totalTransactionsByDate[dateStr] = schemas.Transaction{
			Date:  &transaction.Date,
			Value: transaction.TotalValue,
			Units: transaction.Units,
		}
	}

	accountStateByCategory := &schemas.AccountStateByCategory{
		AssetsByCategory:        &assetsByCategory,
		CategoryAssets:          &categoryAssets,
		TotalHoldingsByDate:     &totalHoldingsByDate,
		TotalTransactionsByDate: &totalTransactionsByDate,
	}

	// Execute
	report, err := service.GenerateReport(ctx, accountStateByCategory, startDate, endDate, interval)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, report)
	assert.NotNil(t, report.AssetsByCategory)
	assert.NotNil(t, report.CategoryAssets)
	assert.NotNil(t, report.TotalHoldingsByDate)

	// Verify the report structure - use the actual category name
	categoryName := mockCategories[0].Name
	stocks, exists := (*report.AssetsByCategory)[categoryName]
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

	// Cleanup after test
	init_test.CleanupTestData(t, db, "test-client")
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

func TestCollapseReturnsByInterval(t *testing.T) {
	service := &services.ReportService{}

	t.Run("Empty returns", func(t *testing.T) {
		result := service.CollapseReturnsByInterval([]schemas.ReturnByDate{}, 24*time.Hour)
		assert.Empty(t, result)
	})

	t.Run("Single daily return with 1-day interval", func(t *testing.T) {
		startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)

		dailyReturns := []schemas.ReturnByDate{
			{
				StartDate:        startDate,
				EndDate:          endDate,
				ReturnPercentage: 5.0,
			},
		}

		result := service.CollapseReturnsByInterval(dailyReturns, 24*time.Hour)
		assert.Len(t, result, 1)
		assert.Equal(t, startDate, result[0].StartDate)
		assert.Equal(t, endDate, result[0].EndDate)
		assert.InDelta(t, 5.0, result[0].ReturnPercentage, 0.001)
	})

	t.Run("Multiple daily returns with 1-day interval", func(t *testing.T) {
		startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

		dailyReturns := []schemas.ReturnByDate{
			{
				StartDate:        startDate,
				EndDate:          startDate.Add(24 * time.Hour),
				ReturnPercentage: 5.0,
			},
			{
				StartDate:        startDate.Add(24 * time.Hour),
				EndDate:          startDate.Add(48 * time.Hour),
				ReturnPercentage: 3.0,
			},
			{
				StartDate:        startDate.Add(48 * time.Hour),
				EndDate:          startDate.Add(72 * time.Hour),
				ReturnPercentage: 2.0,
			},
		}

		result := service.CollapseReturnsByInterval(dailyReturns, 24*time.Hour)
		assert.Len(t, result, 3)

		// Each daily return should be in its own interval
		assert.InDelta(t, 5.0, result[0].ReturnPercentage, 0.001)
		assert.InDelta(t, 3.0, result[1].ReturnPercentage, 0.001)
		assert.InDelta(t, 2.0, result[2].ReturnPercentage, 0.001)
	})

	t.Run("Multiple daily returns with 2-day interval", func(t *testing.T) {
		startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

		dailyReturns := []schemas.ReturnByDate{
			{
				StartDate:        startDate,
				EndDate:          startDate.Add(24 * time.Hour),
				ReturnPercentage: 5.0,
			},
			{
				StartDate:        startDate.Add(24 * time.Hour),
				EndDate:          startDate.Add(48 * time.Hour),
				ReturnPercentage: 3.0,
			},
			{
				StartDate:        startDate.Add(48 * time.Hour),
				EndDate:          startDate.Add(72 * time.Hour),
				ReturnPercentage: 2.0,
			},
			{
				StartDate:        startDate.Add(72 * time.Hour),
				EndDate:          startDate.Add(96 * time.Hour),
				ReturnPercentage: 4.0,
			},
		}

		result := service.CollapseReturnsByInterval(dailyReturns, 48*time.Hour)
		assert.Len(t, result, 2)

		// First interval: (1 + 0.05) * (1 + 0.03) - 1 = 0.0815 = 8.15%
		expectedFirstInterval := (1.05*1.03 - 1) * 100
		assert.InDelta(t, expectedFirstInterval, result[0].ReturnPercentage, 0.001)
		assert.Equal(t, startDate, result[0].StartDate)
		assert.Equal(t, startDate.Add(48*time.Hour), result[0].EndDate)

		// Second interval: (1 + 0.02) * (1 + 0.04) - 1 = 0.0608 = 6.08%
		expectedSecondInterval := (1.02*1.04 - 1) * 100
		assert.InDelta(t, expectedSecondInterval, result[1].ReturnPercentage, 0.001)
		assert.Equal(t, startDate.Add(48*time.Hour), result[1].StartDate)
		assert.Equal(t, startDate.Add(96*time.Hour), result[1].EndDate)
	})

	t.Run("Multiple daily returns with 3-day interval", func(t *testing.T) {
		startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

		dailyReturns := []schemas.ReturnByDate{
			{
				StartDate:        startDate,
				EndDate:          startDate.Add(24 * time.Hour),
				ReturnPercentage: 5.0,
			},
			{
				StartDate:        startDate.Add(24 * time.Hour),
				EndDate:          startDate.Add(48 * time.Hour),
				ReturnPercentage: 3.0,
			},
			{
				StartDate:        startDate.Add(48 * time.Hour),
				EndDate:          startDate.Add(72 * time.Hour),
				ReturnPercentage: 2.0,
			},
			{
				StartDate:        startDate.Add(72 * time.Hour),
				EndDate:          startDate.Add(96 * time.Hour),
				ReturnPercentage: 4.0,
			},
			{
				StartDate:        startDate.Add(96 * time.Hour),
				EndDate:          startDate.Add(120 * time.Hour),
				ReturnPercentage: 1.0,
			},
		}

		result := service.CollapseReturnsByInterval(dailyReturns, 72*time.Hour)
		assert.Len(t, result, 2)

		// First interval: (1 + 0.05) * (1 + 0.03) * (1 + 0.02) - 1 = 0.1031 = 10.31%
		expectedFirstInterval := (1.05*1.03*1.02 - 1) * 100
		assert.InDelta(t, expectedFirstInterval, result[0].ReturnPercentage, 0.001)
		assert.Equal(t, startDate, result[0].StartDate)
		assert.Equal(t, startDate.Add(72*time.Hour), result[0].EndDate)

		// Second interval: (1 + 0.04) * (1 + 0.01) - 1 = 0.0504 = 5.04%
		expectedSecondInterval := (1.04*1.01 - 1) * 100
		assert.InDelta(t, expectedSecondInterval, result[1].ReturnPercentage, 0.001)
		assert.Equal(t, startDate.Add(72*time.Hour), result[1].StartDate)
		assert.Equal(t, startDate.Add(144*time.Hour), result[1].EndDate)
	})

	t.Run("Multiple daily returns with 7-day interval (weekly)", func(t *testing.T) {
		startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

		dailyReturns := []schemas.ReturnByDate{
			// Week 1: Monday to Sunday
			{
				StartDate:        startDate,
				EndDate:          startDate.Add(24 * time.Hour),
				ReturnPercentage: 1.0,
			},
			{
				StartDate:        startDate.Add(24 * time.Hour),
				EndDate:          startDate.Add(48 * time.Hour),
				ReturnPercentage: 2.0,
			},
			{
				StartDate:        startDate.Add(48 * time.Hour),
				EndDate:          startDate.Add(72 * time.Hour),
				ReturnPercentage: -1.0,
			},
			{
				StartDate:        startDate.Add(72 * time.Hour),
				EndDate:          startDate.Add(96 * time.Hour),
				ReturnPercentage: 3.0,
			},
			{
				StartDate:        startDate.Add(96 * time.Hour),
				EndDate:          startDate.Add(120 * time.Hour),
				ReturnPercentage: 1.5,
			},
			{
				StartDate:        startDate.Add(120 * time.Hour),
				EndDate:          startDate.Add(144 * time.Hour),
				ReturnPercentage: -0.5,
			},
			{
				StartDate:        startDate.Add(144 * time.Hour),
				EndDate:          startDate.Add(168 * time.Hour),
				ReturnPercentage: 2.5,
			},
			// Week 2: Monday to Sunday
			{
				StartDate:        startDate.Add(168 * time.Hour),
				EndDate:          startDate.Add(192 * time.Hour),
				ReturnPercentage: 0.5,
			},
			{
				StartDate:        startDate.Add(192 * time.Hour),
				EndDate:          startDate.Add(216 * time.Hour),
				ReturnPercentage: 1.0,
			},
		}

		result := service.CollapseReturnsByInterval(dailyReturns, 168*time.Hour) // 7 days
		assert.Len(t, result, 2)

		// First week: compound return of all 7 daily returns
		expectedFirstWeek := (1.01*1.02*0.99*1.03*1.015*0.995*1.025 - 1) * 100
		assert.InDelta(t, expectedFirstWeek, result[0].ReturnPercentage, 0.001)
		assert.Equal(t, startDate, result[0].StartDate)
		assert.Equal(t, startDate.Add(168*time.Hour), result[0].EndDate)

		// Second week: compound return of the remaining 2 daily returns
		expectedSecondWeek := (1.005*1.01 - 1) * 100
		assert.InDelta(t, expectedSecondWeek, result[1].ReturnPercentage, 0.001)
		assert.Equal(t, startDate.Add(168*time.Hour), result[1].StartDate)
		assert.Equal(t, startDate.Add(336*time.Hour), result[1].EndDate)
	})

	t.Run("Negative returns", func(t *testing.T) {
		startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

		dailyReturns := []schemas.ReturnByDate{
			{
				StartDate:        startDate,
				EndDate:          startDate.Add(24 * time.Hour),
				ReturnPercentage: -5.0,
			},
			{
				StartDate:        startDate.Add(24 * time.Hour),
				EndDate:          startDate.Add(48 * time.Hour),
				ReturnPercentage: -3.0,
			},
			{
				StartDate:        startDate.Add(48 * time.Hour),
				EndDate:          startDate.Add(72 * time.Hour),
				ReturnPercentage: 2.0,
			},
		}

		result := service.CollapseReturnsByInterval(dailyReturns, 48*time.Hour)
		assert.Len(t, result, 2)

		// First interval: (1 - 0.05) * (1 - 0.03) - 1 = -0.0785 = -7.85%
		expectedFirstInterval := (0.95*0.97 - 1) * 100
		assert.InDelta(t, expectedFirstInterval, result[0].ReturnPercentage, 0.001)

		// Second interval: just the last return
		assert.InDelta(t, 2.0, result[1].ReturnPercentage, 0.001)
	})

	t.Run("Zero returns", func(t *testing.T) {
		startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

		dailyReturns := []schemas.ReturnByDate{
			{
				StartDate:        startDate,
				EndDate:          startDate.Add(24 * time.Hour),
				ReturnPercentage: 0.0,
			},
			{
				StartDate:        startDate.Add(24 * time.Hour),
				EndDate:          startDate.Add(48 * time.Hour),
				ReturnPercentage: 5.0,
			},
		}

		result := service.CollapseReturnsByInterval(dailyReturns, 48*time.Hour)
		assert.Len(t, result, 1)

		// Should be 5% since 0% return doesn't affect the compound calculation
		assert.InDelta(t, 5.0, result[0].ReturnPercentage, 0.001)
	})

	t.Run("Large number of daily returns", func(t *testing.T) {
		startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

		// Create 30 daily returns
		dailyReturns := make([]schemas.ReturnByDate, 30)
		for i := 0; i < 30; i++ {
			dailyReturns[i] = schemas.ReturnByDate{
				StartDate:        startDate.Add(time.Duration(i) * 24 * time.Hour),
				EndDate:          startDate.Add(time.Duration(i+1) * 24 * time.Hour),
				ReturnPercentage: 1.0, // 1% daily return
			}
		}

		result := service.CollapseReturnsByInterval(dailyReturns, 168*time.Hour) // 7-day intervals
		assert.Len(t, result, 5)                                                 // 30 days / 7 days = 4 full weeks + 1 partial week

		// Each week should have the same compound return: (1.01)^7 - 1
		expectedWeeklyReturn := (1.01*1.01*1.01*1.01*1.01*1.01*1.01 - 1) * 100
		for i := 0; i < 4; i++ {
			assert.InDelta(t, expectedWeeklyReturn, result[i].ReturnPercentage, 0.001)
		}

		// Last partial week should have 2 days: (1.01)^2 - 1
		expectedPartialWeekReturn := (1.01*1.01 - 1) * 100
		assert.InDelta(t, expectedPartialWeekReturn, result[4].ReturnPercentage, 0.001)
	})

	t.Run("Unsorted daily returns", func(t *testing.T) {
		startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

		// Create returns in reverse order (latest first)
		dailyReturns := []schemas.ReturnByDate{
			{
				StartDate:        startDate.Add(48 * time.Hour), // Jan 3
				EndDate:          startDate.Add(72 * time.Hour),
				ReturnPercentage: 2.0,
			},
			{
				StartDate:        startDate.Add(24 * time.Hour), // Jan 2
				EndDate:          startDate.Add(48 * time.Hour),
				ReturnPercentage: 3.0,
			},
			{
				StartDate:        startDate, // Jan 1
				EndDate:          startDate.Add(24 * time.Hour),
				ReturnPercentage: 5.0,
			},
			{
				StartDate:        startDate.Add(72 * time.Hour), // Jan 4
				EndDate:          startDate.Add(96 * time.Hour),
				ReturnPercentage: 4.0,
			},
		}

		result := service.CollapseReturnsByInterval(dailyReturns, 48*time.Hour)
		assert.Len(t, result, 2)

		// First interval: (1 + 0.05) * (1 + 0.03) - 1 = 0.0815 = 8.15%
		expectedFirstInterval := (1.05*1.03 - 1) * 100
		assert.InDelta(t, expectedFirstInterval, result[0].ReturnPercentage, 0.001)
		assert.Equal(t, startDate, result[0].StartDate)
		assert.Equal(t, startDate.Add(48*time.Hour), result[0].EndDate)

		// Second interval: (1 + 0.02) * (1 + 0.04) - 1 = 0.0608 = 6.08%
		expectedSecondInterval := (1.02*1.04 - 1) * 100
		assert.InDelta(t, expectedSecondInterval, result[1].ReturnPercentage, 0.001)
		assert.Equal(t, startDate.Add(48*time.Hour), result[1].StartDate)
		assert.Equal(t, startDate.Add(96*time.Hour), result[1].EndDate)
	})

	t.Run("Randomly ordered daily returns", func(t *testing.T) {
		startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

		// Create returns in random order
		dailyReturns := []schemas.ReturnByDate{
			{
				StartDate:        startDate.Add(72 * time.Hour), // Jan 4
				EndDate:          startDate.Add(96 * time.Hour),
				ReturnPercentage: 4.0,
			},
			{
				StartDate:        startDate, // Jan 1
				EndDate:          startDate.Add(24 * time.Hour),
				ReturnPercentage: 5.0,
			},
			{
				StartDate:        startDate.Add(48 * time.Hour), // Jan 3
				EndDate:          startDate.Add(72 * time.Hour),
				ReturnPercentage: 2.0,
			},
			{
				StartDate:        startDate.Add(24 * time.Hour), // Jan 2
				EndDate:          startDate.Add(48 * time.Hour),
				ReturnPercentage: 3.0,
			},
			{
				StartDate:        startDate.Add(96 * time.Hour), // Jan 5
				EndDate:          startDate.Add(120 * time.Hour),
				ReturnPercentage: 1.0,
			},
		}

		result := service.CollapseReturnsByInterval(dailyReturns, 72*time.Hour)
		assert.Len(t, result, 2)

		// First interval: (1 + 0.05) * (1 + 0.03) * (1 + 0.02) - 1 = 0.1031 = 10.31%
		expectedFirstInterval := (1.05*1.03*1.02 - 1) * 100
		assert.InDelta(t, expectedFirstInterval, result[0].ReturnPercentage, 0.001)
		assert.Equal(t, startDate, result[0].StartDate)
		assert.Equal(t, startDate.Add(72*time.Hour), result[0].EndDate)

		// Second interval: (1 + 0.04) * (1 + 0.01) - 1 = 0.0504 = 5.04%
		expectedSecondInterval := (1.04*1.01 - 1) * 100
		assert.InDelta(t, expectedSecondInterval, result[1].ReturnPercentage, 0.001)
		assert.Equal(t, startDate.Add(72*time.Hour), result[1].StartDate)
		assert.Equal(t, startDate.Add(144*time.Hour), result[1].EndDate)
	})
}
