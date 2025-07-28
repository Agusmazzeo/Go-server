package services_test

import (
	"context"
	"fmt"
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
	// Setup test database connection
	db := init_test.SetupTestDB(t)

	// Create repository instances
	holdingRepo := repositories.NewHoldingRepository(db)
	transactionRepo := repositories.NewTransactionRepository(db)
	assetRepo := repositories.NewAssetRepository(db)
	assetCategoryRepo := repositories.NewAssetCategoryRepository(db)
	syncLogRepo := repositories.NewSyncLogRepository(db)

	ctx := context.Background()
	token := "test-token"
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 20, 0, 0, 0, 0, time.UTC)

	t.Run("returns error when ESCO service fails", func(t *testing.T) {
		// Use unique account ID for this test
		accountID := fmt.Sprintf("test-client-error-%d", time.Now().UnixNano())

		// Cleanup after test
		defer func() {
			init_test.CleanupTestData(t, db, accountID)
		}()

		// Setup mock ESCO service that returns error
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
	})

	t.Run("successfully syncs and stores account data", func(t *testing.T) {
		// Use unique account ID for this test
		accountID := fmt.Sprintf("test-client-success-%d", time.Now().UnixNano())

		// Cleanup after test
		defer func() {
			init_test.CleanupTestData(t, db, accountID)
		}()

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
		assert.Len(t, transactions, 0) // No transactions in 2024 date range in mock data

		// Verify sync logs were created
		syncedDates, err := syncLogRepo.GetSyncedDates(ctx, accountID, startDate, endDate)
		require.NoError(t, err)
		assert.Len(t, syncedDates, 19) // Should have all dates from start to end
	})

	t.Run("skips sync when data is already synced", func(t *testing.T) {
		// Use unique account ID for this test
		accountID := fmt.Sprintf("test-client-skip-%d", time.Now().UnixNano())

		// Cleanup after test
		defer func() {
			init_test.CleanupTestData(t, db, accountID)
		}()

		// Mark all dates in the range as already synced
		dates := []time.Time{}
		for date := startDate; date.Before(endDate); date = date.AddDate(0, 0, 1) {
			dates = append(dates, date)
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
		accountID := fmt.Sprintf("test-account-1-%d", time.Now().UnixNano())

		// Cleanup after test
		defer func() {
			init_test.CleanupTestData(t, db, accountID)
		}()

		// Create test dates
		date1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		date2 := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
		date3 := time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC)

		// Only sync dates 1 and 3
		datesToSync := []time.Time{date1, date3}

		// Create test account state with holdings and transactions for all three dates
		accountState := &schemas.AccountState{
			Assets: &map[string]schemas.Asset{
				"asset1": {
					ID:           "asset1",
					Category:     "Test Category",
					Type:         "STOCK",
					Denomination: "USD",
					Holdings: []schemas.Holding{
						{DateRequested: &date1, Value: 100.0, Units: 10},
						{DateRequested: &date2, Value: 110.0, Units: 10},
						{DateRequested: &date3, Value: 120.0, Units: 10},
					},
					Transactions: []schemas.Transaction{
						{Date: &date1, Value: 10.0, Units: 1},
						{Date: &date2, Value: 11.0, Units: 1},
						{Date: &date3, Value: 12.0, Units: 1},
					},
				},
			},
		}

		// Execute
		err := service.StoreAccountState(ctx, accountID, accountState, datesToSync)
		require.NoError(t, err)

		// Verify only holdings for dates 1 and 3 were stored
		holdings, err := holdingRepo.GetByClientID(ctx, accountID, date1, date3)
		require.NoError(t, err)
		assert.Len(t, holdings, 2)

		// Verify only transactions for dates 1 and 3 were stored
		transactions, err := transactionRepo.GetByClientID(ctx, accountID, date1, date3)
		require.NoError(t, err)
		assert.Len(t, transactions, 2)
	})

	t.Run("handles empty datesToSync", func(t *testing.T) {
		accountID := fmt.Sprintf("test-account-2-%d", time.Now().UnixNano())

		// Cleanup after test
		defer func() {
			init_test.CleanupTestData(t, db, accountID)
		}()

		// Create test account state
		date := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		accountState := &schemas.AccountState{
			Assets: &map[string]schemas.Asset{
				"asset2": {
					ID:           "asset2",
					Category:     "Test Category",
					Type:         "STOCK",
					Denomination: "USD",
					Holdings: []schemas.Holding{
						{DateRequested: &date, Value: 100.0, Units: 10},
					},
					Transactions: []schemas.Transaction{
						{Date: &date, Value: 10.0, Units: 1},
					},
				},
			},
		}

		// Execute with empty datesToSync
		err := service.StoreAccountState(ctx, accountID, accountState, []time.Time{})
		require.NoError(t, err)

		// Verify no data was stored
		holdings, err := holdingRepo.GetByClientID(ctx, accountID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC))
		require.NoError(t, err)
		assert.Len(t, holdings, 0)

		transactions, err := transactionRepo.GetByClientID(ctx, accountID, time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC))
		require.NoError(t, err)
		assert.Len(t, transactions, 0)
	})

	t.Run("handles nil dates in holdings and transactions", func(t *testing.T) {
		accountID := fmt.Sprintf("test-account-3-%d", time.Now().UnixNano())

		// Cleanup after test
		defer func() {
			init_test.CleanupTestData(t, db, accountID)
		}()

		date1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		var nilDate *time.Time

		// Create test account state with some nil dates
		accountState := &schemas.AccountState{
			Assets: &map[string]schemas.Asset{
				"asset3": {
					ID:           "asset3",
					Category:     "Test Category",
					Type:         "STOCK",
					Denomination: "USD",
					Holdings: []schemas.Holding{
						{DateRequested: &date1, Value: 100.0, Units: 10},
						{DateRequested: nilDate, Value: 110.0, Units: 10},
					},
					Transactions: []schemas.Transaction{
						{Date: &date1, Value: 10.0, Units: 1},
						{Date: nilDate, Value: 11.0, Units: 1},
					},
				},
			},
		}

		// Execute
		err := service.StoreAccountState(ctx, accountID, accountState, []time.Time{date1})
		require.NoError(t, err)

		// Verify only data with valid dates was stored
		holdings, err := holdingRepo.GetByClientID(ctx, accountID, date1, date1)
		require.NoError(t, err)
		assert.Len(t, holdings, 1, "Should only have 1 holding with valid date")

		transactions, err := transactionRepo.GetByClientID(ctx, accountID, date1, date1)
		require.NoError(t, err)
		assert.Len(t, transactions, 1, "Should only have 1 transaction with valid date")
	})
}
