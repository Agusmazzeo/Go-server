package repositories_test

import (
	"context"
	"server/src/repositories"
	"testing"
	"time"

	"server/tests/init_test"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTest(t *testing.T) (*pgxpool.Pool, repositories.SyncLogRepository) {
	db := init_test.SetupTestDB(t)
	repo := repositories.NewSyncLogRepository(db)

	// Cleanup test data after test
	t.Cleanup(func() {
		init_test.CleanupTestData(t, db, "test-client-1")
		init_test.CleanupTestData(t, db, "test-client-2")
		init_test.CleanupTestData(t, db, "test-client-3")
		init_test.CleanupTestData(t, db, "test-client-4")
	})

	return db, repo
}

func TestMarkClientForDate(t *testing.T) {
	_, repo := setupTest(t)

	ctx := context.Background()
	clientID := "test-client-1"
	syncDate := time.Now()

	// Test Insert
	err := repo.MarkClientForDate(ctx, clientID, syncDate)
	require.NoError(t, err)

	// Test GetLastSyncDate
	lastSyncDate, err := repo.GetLastSyncDate(ctx, clientID)
	require.NoError(t, err)
	assert.NotNil(t, lastSyncDate)
	assert.True(t, lastSyncDate.Before(syncDate))
}

func TestGetLastSyncDate(t *testing.T) {
	_, repo := setupTest(t)

	ctx := context.Background()

	t.Run("returns nil for non-existent client", func(t *testing.T) {
		nonExistentClientID := "non-existent-client"
		lastSyncDate, err := repo.GetLastSyncDate(ctx, nonExistentClientID)
		require.NoError(t, err)
		assert.Nil(t, lastSyncDate)
	})

	t.Run("returns last sync date for existing client", func(t *testing.T) {
		clientID := "test-client-2"
		dates := []time.Time{
			time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 3, 2, 0, 0, 0, 0, time.UTC),
		}

		err := repo.MarkClientForDates(ctx, clientID, dates)
		require.NoError(t, err)

		lastSyncDate, err := repo.GetLastSyncDate(ctx, clientID)
		require.NoError(t, err)
		assert.NotNil(t, lastSyncDate)
		assert.Equal(t, dates[1], *lastSyncDate)
	})
}

func TestMarkClientForDates(t *testing.T) {
	db, repo := setupTest(t)

	ctx := context.Background()

	t.Run("inserts multiple dates", func(t *testing.T) {
		clientID := "test-client-3"
		dates := []time.Time{
			time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 3, 2, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 3, 3, 0, 0, 0, 0, time.UTC),
		}

		err := repo.MarkClientForDates(ctx, clientID, dates)
		require.NoError(t, err)

		// Verify all dates were inserted
		for _, date := range dates {
			var count int
			err := db.QueryRow(ctx, `
				SELECT COUNT(*)
				FROM sync_logs
				WHERE client_id = $1 AND sync_date = $2
			`, clientID, date).Scan(&count)
			require.NoError(t, err)
			assert.Equal(t, 1, count, "Expected one record for date %v", date)
		}
	})

	t.Run("handles empty dates slice", func(t *testing.T) {
		clientID := "test-client-4"
		dates := []time.Time{}

		err := repo.MarkClientForDates(ctx, clientID, dates)
		require.NoError(t, err)

		// Verify no records were inserted
		var count int
		err = db.QueryRow(ctx, `
			SELECT COUNT(*)
			FROM sync_logs
			WHERE client_id = $1
		`, clientID).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 0, count)
	})

	t.Run("handles duplicate dates", func(t *testing.T) {
		clientID := "test-client-5"
		date := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)
		dates := []time.Time{date, date, date}

		err := repo.MarkClientForDates(ctx, clientID, dates)
		require.NoError(t, err)

		// Verify only one record was inserted
		var count int
		err = db.QueryRow(ctx, `
			SELECT COUNT(*)
			FROM sync_logs
			WHERE client_id = $1 AND sync_date = $2
		`, clientID, date).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 1, count)
	})

	t.Run("handles multiple clients", func(t *testing.T) {
		clientID1 := "test-client-6"
		clientID2 := "test-client-7"
		dates := []time.Time{
			time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 3, 2, 0, 0, 0, 0, time.UTC),
		}

		// Insert for first client
		err := repo.MarkClientForDates(ctx, clientID1, dates)
		require.NoError(t, err)

		// Insert for second client
		err = repo.MarkClientForDates(ctx, clientID2, dates)
		require.NoError(t, err)

		// Verify both clients have their records
		for _, clientID := range []string{clientID1, clientID2} {
			var count int
			err := db.QueryRow(ctx, `
				SELECT COUNT(*)
				FROM sync_logs
				WHERE client_id = $1
			`, clientID).Scan(&count)
			require.NoError(t, err)
			assert.Equal(t, 2, count, "Expected two records for client %s", clientID)
		}
	})
}

func TestGetSyncedDates(t *testing.T) {
	_, repo := setupTest(t)

	ctx := context.Background()
	clientID := "test-client-range-1"

	// Create test data
	dates := []time.Time{
		time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 3, 2, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 3, 3, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 3, 4, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 3, 5, 0, 0, 0, 0, time.UTC),
	}

	err := repo.MarkClientForDates(ctx, clientID, dates)
	require.NoError(t, err)

	t.Run("returns all dates in range", func(t *testing.T) {
		syncedDates, err := repo.GetSyncedDates(ctx, clientID, time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 3, 6, 0, 0, 0, 0, time.UTC))
		require.NoError(t, err)
		assert.Equal(t, dates, syncedDates)
	})

	t.Run("returns partial range", func(t *testing.T) {
		syncedDates, err := repo.GetSyncedDates(ctx, clientID, time.Date(2024, 3, 2, 0, 0, 0, 0, time.UTC), time.Date(2024, 3, 4, 0, 0, 0, 0, time.UTC))
		require.NoError(t, err)
		assert.Equal(t, dates[1:3], syncedDates)
	})

	t.Run("returns empty slice for no matches", func(t *testing.T) {
		syncedDates, err := repo.GetSyncedDates(ctx, clientID, time.Date(2024, 3, 6, 0, 0, 0, 0, time.UTC), time.Date(2024, 3, 7, 0, 0, 0, 0, time.UTC))
		require.NoError(t, err)
		assert.Empty(t, syncedDates)
	})

	t.Run("handles duplicate dates", func(t *testing.T) {
		clientID := "test-client-range-2"
		duplicateDates := []time.Time{
			time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 3, 2, 0, 0, 0, 0, time.UTC),
		}

		err := repo.MarkClientForDates(ctx, clientID, duplicateDates)
		require.NoError(t, err)

		syncedDates, err := repo.GetSyncedDates(ctx, clientID, time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 3, 3, 0, 0, 0, 0, time.UTC))
		require.NoError(t, err)
		assert.Equal(t, []time.Time{
			time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 3, 2, 0, 0, 0, 0, time.UTC),
		}, syncedDates)
	})

	t.Run("handles non-existent client", func(t *testing.T) {
		syncedDates, err := repo.GetSyncedDates(ctx, "non-existent-client", time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 3, 6, 0, 0, 0, 0, time.UTC))
		require.NoError(t, err)
		assert.Empty(t, syncedDates)
	})
}

func TestCleanupSyncLogs(t *testing.T) {
	db, repo := setupTest(t)

	_, err := db.Exec(context.Background(), `DELETE FROM sync_logs`)
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("cleans up logs before specified date", func(t *testing.T) {
		clientID := "test-cleanup-1"
		dates := []time.Time{
			time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 3, 2, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 3, 3, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 3, 4, 0, 0, 0, 0, time.UTC),
		}

		// Insert test data
		err := repo.MarkClientForDates(ctx, clientID, dates)
		require.NoError(t, err)

		// Clean up logs before March 3rd
		cleanupDate := time.Date(2024, 3, 3, 0, 0, 0, 0, time.UTC)
		err = repo.CleanupSyncLogs(ctx, clientID, cleanupDate, cleanupDate)
		require.NoError(t, err)

		// Verify only logs from March 3rd and later remain
		var count int
		err = db.QueryRow(ctx, `
			SELECT COUNT(*)
			FROM sync_logs
			WHERE client_id = $1
		`, clientID).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 3, count, "Expected 3 records to remain")

		// Verify specific dates remain
		remainingDates, err := repo.GetSyncedDates(ctx, clientID, time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC), time.Date(2024, 3, 5, 0, 0, 0, 0, time.UTC))
		require.NoError(t, err)
		assert.Equal(t, []time.Time{
			time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 3, 2, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 3, 4, 0, 0, 0, 0, time.UTC),
		}, remainingDates)
	})

	t.Run("handles non-existent client", func(t *testing.T) {
		nonExistentClientID := "non-existent-cleanup"
		err := repo.CleanupSyncLogs(ctx, nonExistentClientID, time.Now(), time.Time{})
		require.NoError(t, err)
	})

	t.Run("handles empty date range", func(t *testing.T) {
		clientID := "test-cleanup-2"
		dates := []time.Time{
			time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 3, 2, 0, 0, 0, 0, time.UTC),
		}

		// Insert test data
		err := repo.MarkClientForDates(ctx, clientID, dates)
		require.NoError(t, err)

		// Clean up with empty date range
		err = repo.CleanupSyncLogs(ctx, clientID, time.Time{}, time.Time{})
		require.NoError(t, err)

		// Verify all records remain
		var count int
		err = db.QueryRow(ctx, `
			SELECT COUNT(*)
			FROM sync_logs
			WHERE client_id = $1
		`, clientID).Scan(&count)
		require.NoError(t, err)
		assert.Equal(t, 2, count, "Expected all records to remain")
	})

	t.Run("preserves logs for other clients", func(t *testing.T) {
		clientID1 := "test-cleanup-3"
		clientID2 := "test-cleanup-4"
		dates := []time.Time{
			time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC),
			time.Date(2024, 3, 2, 0, 0, 0, 0, time.UTC),
		}

		// Insert test data for both clients
		err := repo.MarkClientForDates(ctx, clientID1, dates)
		require.NoError(t, err)
		err = repo.MarkClientForDates(ctx, clientID2, dates)
		require.NoError(t, err)

		// Clean up logs for first client
		cleanupDate := time.Date(2024, 3, 2, 0, 0, 0, 0, time.UTC)
		err = repo.CleanupSyncLogs(ctx, clientID1, cleanupDate, cleanupDate)
		require.NoError(t, err)

		// Verify first client's logs are cleaned up
		var count1 int
		err = db.QueryRow(ctx, `
			SELECT COUNT(*)
			FROM sync_logs
			WHERE client_id = $1
		`, clientID1).Scan(&count1)
		require.NoError(t, err)
		assert.Equal(t, 1, count1, "Expected one record to remain for first client")

		// Verify second client's logs are preserved
		var count2 int
		err = db.QueryRow(ctx, `
			SELECT COUNT(*)
			FROM sync_logs
			WHERE client_id = $1
		`, clientID2).Scan(&count2)
		require.NoError(t, err)
		assert.Equal(t, 2, count2, "Expected all records to remain for second client")
	})
}
