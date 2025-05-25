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

func TestIsDataSynced(t *testing.T) {
	service, _ := setupTest(t, nil, nil, nil, nil)
	ctx := context.Background()
	accountID := "4014D4EFDD5DE27B"
	token := "test-token"

	t.Run("returns true when all dates are synced", func(t *testing.T) {
		// Setup test data
		startDate := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2024, 3, 3, 0, 0, 0, 0, time.UTC)

		// Mark all dates as synced
		dates := []time.Time{
			time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 3, 2, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 3, 3, 0, 0, 0, 0, time.UTC),
		}
		err := service.syncLogRepo.MarkClientForDates(ctx, accountID, dates)
		require.NoError(t, err)

		// Cleanup sync logs
		defer func() {
			err = service.syncLogRepo.CleanupSyncLogs(ctx, accountID, startDate, endDate)
			require.NoError(t, err)
		}()

		// Test
		isSynced, err := service.IsDataSynced(ctx, token, accountID, startDate, endDate)
		require.NoError(t, err)
		assert.True(t, isSynced)
	})

	t.Run("returns false when some dates are missing", func(t *testing.T) {
		// Setup test data
		startDate := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2024, 3, 3, 0, 0, 0, 0, time.UTC)

		// Mark only some dates as synced
		dates := []time.Time{
			time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 3, 3, 0, 0, 0, 0, time.UTC),
		}
		err := service.syncLogRepo.MarkClientForDates(ctx, accountID, dates)
		require.NoError(t, err)

		// Cleanup sync logs
		defer func() {
			err = service.syncLogRepo.CleanupSyncLogs(ctx, accountID, startDate, endDate)
			require.NoError(t, err)
		}()

		// Test
		isSynced, err := service.IsDataSynced(ctx, token, accountID, startDate, endDate)
		require.NoError(t, err)
		assert.False(t, isSynced)
	})

	t.Run("returns false when no dates are synced", func(t *testing.T) {
		// Setup test data
		startDate := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2024, 3, 3, 0, 0, 0, 0, time.UTC)

		err := service.syncLogRepo.CleanupSyncLogs(ctx, accountID, startDate, endDate)
		require.NoError(t, err)

		// Test with no synced dates
		isSynced, err := service.IsDataSynced(ctx, token, accountID, startDate, endDate)
		require.NoError(t, err)
		assert.False(t, isSynced)
	})

	t.Run("handles single day range", func(t *testing.T) {
		// Setup test data
		date := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)

		// Mark the date as synced
		err := service.syncLogRepo.MarkClientForDates(ctx, accountID, []time.Time{date})
		require.NoError(t, err)

		// Cleanup sync logs
		defer func() {
			err = service.syncLogRepo.CleanupSyncLogs(ctx, accountID, date, date)
			require.NoError(t, err)
		}()

		// Test
		isSynced, err := service.IsDataSynced(ctx, token, accountID, date, date)
		require.NoError(t, err)
		assert.True(t, isSynced)
	})

	t.Run("handles non-existent account", func(t *testing.T) {
		// Setup test data
		startDate := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2024, 3, 3, 0, 0, 0, 0, time.UTC)

		// Test with non-existent account
		isSynced, err := service.IsDataSynced(ctx, token, "non-existent-account", startDate, endDate)
		require.NoError(t, err)
		assert.False(t, isSynced)
	})
}

func TestSyncDataFromAccount(t *testing.T) {
	db := init_test.SetupTestDB(t)
	ctx := context.Background()
	accountID := "4014D4EFDD5DE27B"
	token := "test-token"
	startDate := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 3, 3, 0, 0, 0, 0, time.UTC)

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

		// Verify holdings were created
		holdings, err := holdingRepo.GetByClientID(ctx, accountID)
		require.NoError(t, err)
		assert.Len(t, holdings, 4)

		// Verify transactions were created
		transactions, err := transactionRepo.GetByClientID(ctx, accountID)
		require.NoError(t, err)
		assert.Len(t, transactions, 2)

		// Verify sync logs were created
		syncedDates, err := syncLogRepo.GetSyncedDates(ctx, accountID, startDate, endDate)
		require.NoError(t, err)
		assert.Len(t, syncedDates, 2) // Should have all dates from start to end
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
