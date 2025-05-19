package services_test

import (
	"context"
	"server/src/repositories"
	"server/src/services"
	"server/tests/repositories/test_init"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testSyncService struct {
	*services.SyncService
	syncLogRepo repositories.SyncLogRepository
}

func setupTest(t *testing.T) *testSyncService {
	db := test_init.SetupTestDB(t)
	syncLogRepo := repositories.NewSyncLogRepository(db)
	service := services.NewSyncService(nil, nil, nil, nil, syncLogRepo, nil)
	return &testSyncService{
		SyncService: service,
		syncLogRepo: syncLogRepo,
	}
}

func TestIsDataSynced(t *testing.T) {
	service := setupTest(t)
	ctx := context.Background()
	accountID := "test-account-1"
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
