package services_test

import (
	"context"
	"os"
	"path/filepath"
	"server/src/models"
	"server/src/repositories"
	"server/src/schemas"
	"server/src/services"
	esco_test "server/tests/clients/esco"
	"server/tests/init_test"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testSyncService struct {
	*services.SyncService
	syncLogRepo repositories.SyncLogRepository
}

func setupTest(t *testing.T, holdingRepo repositories.HoldingRepository, transactionRepo repositories.TransactionRepository, assetRepo repositories.AssetRepository, assetCategoryRepo repositories.AssetCategoryRepository) (*testSyncService, *esco_test.ESCOServiceClientMock) {
	db := init_test.SetupTestDB(t)
	syncLogRepo := repositories.NewSyncLogRepository(db)

	// Setup mock ESCO service
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

	escoService := services.NewESCOService(mockClient)
	service := services.NewSyncService(
		holdingRepo,
		transactionRepo,
		assetRepo,
		assetCategoryRepo,
		syncLogRepo,
		escoService,
	)

	return &testSyncService{
		SyncService: service,
		syncLogRepo: syncLogRepo,
	}, mockClient
}

func TestSyncDataFromAccount(t *testing.T) {
	db := init_test.SetupTestDB(t)
	ctx := context.Background()
	accountID := "test-client"
	token := "test-token"
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC)

	// Initialize repositories
	holdingRepo := repositories.NewHoldingRepository(db)
	transactionRepo := repositories.NewTransactionRepository(db)
	assetRepo := repositories.NewAssetRepository(db)
	assetCategoryRepo := repositories.NewAssetCategoryRepository(db)
	syncLogRepo := repositories.NewSyncLogRepository(db)

	t.Run("returns error when ESCO service fails", func(t *testing.T) {
		// Setup mock ESCO service with error
		mockESCO := esco_test.NewMockESCOService(func(ctx context.Context, token, accountID string, startDate, endDate time.Time, interval time.Duration) (*schemas.AccountState, error) {
			return nil, assert.AnError
		})

		// Initialize service with mocked dependencies
		service := services.NewSyncService(
			holdingRepo,
			transactionRepo,
			assetRepo,
			assetCategoryRepo,
			syncLogRepo,
			mockESCO,
		)

		// Execute sync
		err := service.SyncDataFromAccount(ctx, token, accountID, startDate, endDate)
		assert.Error(t, err)
		assert.Equal(t, assert.AnError, err)
	})
	t.Run("successfully syncs and stores account data", func(t *testing.T) {
		// Setup test with mock ESCO service
		testService, _ := setupTest(t, holdingRepo, transactionRepo, assetRepo, assetCategoryRepo)

		// Execute sync
		err := testService.SyncDataFromAccount(ctx, token, accountID, startDate, endDate)
		require.NoError(t, err)

		// Verify asset category was created
		category, err := assetCategoryRepo.GetByName(ctx, "ON HARD DOLLAR")
		require.NoError(t, err)
		assert.NotNil(t, category)
		assert.Equal(t, "ON HARD DOLLAR", category.Name)

		// Verify asset was created
		assets, err := assetRepo.GetAll(ctx)
		require.NoError(t, err)
		var foundAsset *models.Asset
		for _, asset := range assets {
			if asset.ExternalID == "YMCQO" {
				foundAsset = &asset
				break
			}
		}
		assert.NotNil(t, foundAsset)
		assert.Equal(t, "57118 / ON YPF CL. 25 V13/02/26", foundAsset.Name)
		assert.Equal(t, "Letes / Pesos", foundAsset.AssetType)
		assert.Equal(t, category.ID, foundAsset.CategoryID)

		// Verify holdings were created (4 assets × 19 days = 76 holdings)
		holdings, err := holdingRepo.GetByClientID(ctx, accountID, startDate, endDate)
		require.NoError(t, err)
		assert.Len(t, holdings, 76)

		// Verify transactions were created (1 asset has 1 transaction for 1 day = 1 transaction)
		transactions, err := transactionRepo.GetByClientID(ctx, accountID, startDate, endDate)
		require.NoError(t, err)
		assert.Len(t, transactions, 1)

		// Verify sync logs were created
		syncedDates, err := syncLogRepo.GetSyncedDates(ctx, accountID, startDate, endDate)
		require.NoError(t, err)
		assert.Len(t, syncedDates, 19) // Should have all dates from start to end
	})

	t.Run("skips sync when data is already synced", func(t *testing.T) {
		// Mark dates as already synced
		dates := []time.Time{
			time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 3, 2, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 3, 3, 0, 0, 0, 0, time.UTC),
		}
		err := syncLogRepo.MarkClientForDates(ctx, accountID, dates)
		require.NoError(t, err)

		// Cleanup sync logs after test
		defer func() {
			err = syncLogRepo.CleanupSyncLogs(ctx, accountID, startDate, endDate)
			require.NoError(t, err)
		}()

		// Setup mock ESCO service that should not be called
		mockESCO := esco_test.NewMockESCOService(func(ctx context.Context, token, accountID string, startDate, endDate time.Time, interval time.Duration) (*schemas.AccountState, error) {
			t.Error("ESCO service should not be called when data is already synced")
			return nil, nil
		})

		// Initialize service with mocked dependencies
		service := services.NewSyncService(
			holdingRepo,
			transactionRepo,
			assetRepo,
			assetCategoryRepo,
			syncLogRepo,
			mockESCO,
		)

		// Execute sync
		err = service.SyncDataFromAccount(ctx, token, accountID, startDate, endDate)
		require.NoError(t, err)
	})
}

func TestStoreAccountStateWithDateFiltering(t *testing.T) {
	// Setup test database connection
	db := init_test.SetupTestDB(t)
	defer init_test.TruncateTables(t, db)

	// Create repository instances
	holdingRepo := repositories.NewHoldingRepository(db)
	transactionRepo := repositories.NewTransactionRepository(db)
	assetRepo := repositories.NewAssetRepository(db)
	assetCategoryRepo := repositories.NewAssetCategoryRepository(db)
	syncLogRepo := repositories.NewSyncLogRepository(db)

	// Setup mock ESCO service
	mockESCO := esco_test.NewMockESCOService(func(ctx context.Context, token, accountID string, startDate, endDate time.Time, interval time.Duration) (*schemas.AccountState, error) {
		return nil, nil
	})

	service := services.NewSyncService(
		holdingRepo,
		transactionRepo,
		assetRepo,
		assetCategoryRepo,
		syncLogRepo,
		mockESCO,
	)

	ctx := context.Background()

	t.Run("only stores data for dates in datesToSync", func(t *testing.T) {
		accountID := "test-account-1"
		// Create test dates
		date1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		date2 := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
		date3 := time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC)

		// Only sync dates 1 and 3
		datesToSync := []time.Time{date1, date3}

		// Create test account state with holdings and transactions for all three dates
		assets := map[string]schemas.Asset{
			"asset1": {
				ID:           "asset1",
				Type:         "STOCK",
				Denomination: "USD",
				Category:     "STOCKS",
				Holdings: []schemas.Holding{
					{
						DateRequested: &date1,
						Value:         1000.0,
						Units:         10,
					},
					{
						DateRequested: &date2,
						Value:         1100.0,
						Units:         10,
					},
					{
						DateRequested: &date3,
						Value:         1200.0,
						Units:         10,
					},
				},
				Transactions: []schemas.Transaction{
					{
						Date:  &date1,
						Value: 100.0,
						Units: 1,
					},
					{
						Date:  &date2,
						Value: 50.0,
						Units: 0.5,
					},
					{
						Date:  &date3,
						Value: 75.0,
						Units: 0.75,
					},
				},
			},
		}
		accountState := &schemas.AccountState{
			Assets: &assets,
		}

		// Store account state with date filtering
		err := service.StoreAccountState(ctx, accountID, accountState, datesToSync)
		require.NoError(t, err)

		// Verify only holdings for dates 1 and 3 were stored
		holdings, err := holdingRepo.GetByClientID(ctx, accountID, date1, date3)
		require.NoError(t, err)
		assert.Len(t, holdings, 2, "Should only have 2 holdings for dates 1 and 3")

		// Verify the holdings are for the correct dates
		holdingDates := make(map[time.Time]bool)
		for _, holding := range holdings {
			holdingDates[holding.Date] = true
		}
		assert.True(t, holdingDates[date1], "Should have holding for date1")
		assert.False(t, holdingDates[date2], "Should not have holding for date2")
		assert.True(t, holdingDates[date3], "Should have holding for date3")

		// Verify only transactions for dates 1 and 3 were stored
		transactions, err := transactionRepo.GetByClientID(ctx, accountID, date1, date3)
		require.NoError(t, err)
		assert.Len(t, transactions, 2, "Should only have 2 transactions for dates 1 and 3")

		// Verify the transactions are for the correct dates
		transactionDates := make(map[time.Time]bool)
		for _, transaction := range transactions {
			transactionDates[transaction.Date] = true
		}
		assert.True(t, transactionDates[date1], "Should have transaction for date1")
		assert.False(t, transactionDates[date2], "Should not have transaction for date2")
		assert.True(t, transactionDates[date3], "Should have transaction for date3")

		// Verify only dates 1 and 3 were marked as synced
		syncedDates, err := syncLogRepo.GetSyncedDates(ctx, accountID, date1, date3.Add(24*time.Hour))
		require.NoError(t, err)
		assert.Len(t, syncedDates, 2, "Should only have 2 synced dates")

		// Verify the synced dates are correct
		syncedDatesMap := make(map[time.Time]bool)
		for _, date := range syncedDates {
			syncedDatesMap[date] = true
		}
		assert.True(t, syncedDatesMap[date1], "Should have date1 marked as synced")
		assert.False(t, syncedDatesMap[date2], "Should not have date2 marked as synced")
		assert.True(t, syncedDatesMap[date3], "Should have date3 marked as synced")

		// Clean up test data
		err = syncLogRepo.CleanupSyncLogs(ctx, accountID, date1, date3)
		require.NoError(t, err)
	})

	t.Run("handles empty datesToSync", func(t *testing.T) {
		accountID := "test-account-2"
		date1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		datesToSync := []time.Time{}

		assets := map[string]schemas.Asset{
			"asset2": {
				ID:           "asset2",
				Type:         "BOND",
				Denomination: "ARS",
				Category:     "BONDS",
				Holdings: []schemas.Holding{
					{
						DateRequested: &date1,
						Value:         5000.0,
						Units:         50,
					},
				},
				Transactions: []schemas.Transaction{
					{
						Date:  &date1,
						Value: 200.0,
						Units: 2,
					},
				},
			},
		}
		accountState := &schemas.AccountState{
			Assets: &assets,
		}

		// Store account state with empty datesToSync
		err := service.StoreAccountState(ctx, accountID, accountState, datesToSync)
		require.NoError(t, err)

		// Verify no holdings were stored
		holdings, err := holdingRepo.GetByClientID(ctx, accountID, date1, date1)
		require.NoError(t, err)
		assert.Len(t, holdings, 0, "Should not have any holdings when datesToSync is empty")

		// Verify no transactions were stored
		transactions, err := transactionRepo.GetByClientID(ctx, accountID, date1, date1)
		require.NoError(t, err)
		assert.Len(t, transactions, 0, "Should not have any transactions when datesToSync is empty")

		// Verify no dates were marked as synced
		syncedDates, err := syncLogRepo.GetSyncedDates(ctx, accountID, date1, date1)
		require.NoError(t, err)
		assert.Len(t, syncedDates, 0, "Should not have any synced dates when datesToSync is empty")

		// Clean up test data (even though none should have been created)
		err = syncLogRepo.CleanupSyncLogs(ctx, accountID, date1, date1)
		require.NoError(t, err)
	})

	t.Run("handles nil dates in holdings and transactions", func(t *testing.T) {
		accountID := "test-account-3"
		date1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		datesToSync := []time.Time{date1}

		assets := map[string]schemas.Asset{
			"asset3": {
				ID:           "asset3",
				Type:         "CASH",
				Denomination: "USD",
				Category:     "CASH",
				Holdings: []schemas.Holding{
					{
						DateRequested: nil, // Nil date
						Value:         100.0,
						Units:         1,
					},
					{
						DateRequested: &date1,
						Value:         200.0,
						Units:         2,
					},
				},
				Transactions: []schemas.Transaction{
					{
						Date:  nil, // Nil date
						Value: 50.0,
						Units: 0.5,
					},
					{
						Date:  &date1,
						Value: 75.0,
						Units: 0.75,
					},
				},
			},
		}
		accountState := &schemas.AccountState{
			Assets: &assets,
		}

		// Store account state with nil dates
		err := service.StoreAccountState(ctx, accountID, accountState, datesToSync)
		require.NoError(t, err)

		// Verify only holdings with valid dates were stored
		holdings, err := holdingRepo.GetByClientID(ctx, accountID, date1, date1)
		require.NoError(t, err)
		assert.Len(t, holdings, 1, "Should only have 1 holding with valid date")

		// Verify only transactions with valid dates were stored
		transactions, err := transactionRepo.GetByClientID(ctx, accountID, date1, date1)
		require.NoError(t, err)
		assert.Len(t, transactions, 1, "Should only have 1 transaction with valid date")

		// Clean up test data
		err = syncLogRepo.CleanupSyncLogs(ctx, accountID, date1, date1)
		require.NoError(t, err)
	})
}
