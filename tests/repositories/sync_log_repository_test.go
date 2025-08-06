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
		init_test.CleanupTestDataByClientID(t, db, "test-client-1")
		init_test.CleanupTestDataByClientID(t, db, "test-client-2")
		init_test.CleanupTestDataByClientID(t, db, "test-client-3")
		init_test.CleanupTestDataByClientID(t, db, "test-client-4")
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
	// Setup test database
	pool := init_test.SetupTestDB(t)
	defer init_test.CleanupTestDB()

	repo := repositories.NewSyncLogRepository(pool)
	ctx := context.Background()

	// Test data
	clientID := "test-client-123"
	testDate1 := time.Date(2023, 11, 1, 0, 0, 0, 0, time.UTC)
	testDate2 := time.Date(2023, 11, 2, 0, 0, 0, 0, time.UTC)
	testDate3 := time.Date(2023, 11, 3, 0, 0, 0, 0, time.UTC)

	// Insert test data
	err := repo.MarkClientForDates(ctx, clientID, []time.Time{testDate1, testDate2, testDate3})
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Test 1: Get all dates in range
	startDate := time.Date(2023, 11, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2023, 11, 4, 0, 0, 0, 0, time.UTC)

	dates, err := repo.GetSyncedDates(ctx, clientID, startDate, endDate)
	if err != nil {
		t.Fatalf("GetSyncedDates failed: %v", err)
	}

	if len(dates) != 3 {
		t.Errorf("Expected 3 dates, got %d", len(dates))
	}

	// Verify the dates are correct
	expectedDates := []time.Time{testDate1, testDate2, testDate3}
	for i, expected := range expectedDates {
		if i >= len(dates) {
			t.Errorf("Missing date at index %d", i)
			continue
		}
		if !dates[i].Equal(expected) {
			t.Errorf("Expected date %v, got %v", expected, dates[i])
		}
	}

	// Test 2: Get dates in partial range
	startDate2 := time.Date(2023, 11, 2, 0, 0, 0, 0, time.UTC)
	endDate2 := time.Date(2023, 11, 3, 0, 0, 0, 0, time.UTC)

	dates2, err := repo.GetSyncedDates(ctx, clientID, startDate2, endDate2)
	if err != nil {
		t.Fatalf("GetSyncedDates failed: %v", err)
	}

	if len(dates2) != 1 {
		t.Errorf("Expected 1 date, got %d", len(dates2))
	}

	if !dates2[0].Equal(testDate2) {
		t.Errorf("Expected date %v, got %v", testDate2, dates2[0])
	}

	// Test 3: Get dates for non-existent client
	dates3, err := repo.GetSyncedDates(ctx, "non-existent-client", startDate, endDate)
	if err != nil {
		t.Fatalf("GetSyncedDates failed: %v", err)
	}

	if len(dates3) != 0 {
		t.Errorf("Expected 0 dates for non-existent client, got %d", len(dates3))
	}

	// Test 4: Get dates in empty range
	startDate4 := time.Date(2023, 12, 1, 0, 0, 0, 0, time.UTC)
	endDate4 := time.Date(2023, 12, 2, 0, 0, 0, 0, time.UTC)

	dates4, err := repo.GetSyncedDates(ctx, clientID, startDate4, endDate4)
	if err != nil {
		t.Fatalf("GetSyncedDates failed: %v", err)
	}

	if len(dates4) != 0 {
		t.Errorf("Expected 0 dates for empty range, got %d", len(dates4))
	}
}

// Debug function to help identify issues in your environment
func TestGetSyncedDatesDebug(t *testing.T) {
	// Setup test database
	pool := init_test.SetupTestDB(t)
	defer init_test.CleanupTestDB()

	repo := repositories.NewSyncLogRepository(pool)
	ctx := context.Background()

	// Test data
	clientID := "debug-client"
	testDate := time.Date(2023, 11, 1, 0, 0, 0, 0, time.UTC)

	// Step 1: Check if data exists
	t.Logf("1. Checking if data exists for client: %s", clientID)
	var count int
	err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM sync_logs WHERE client_id = $1", clientID).Scan(&count)
	if err != nil {
		t.Logf("Error checking count: %v", err)
	} else {
		t.Logf("   Found %d records for this client", count)
	}

	// Step 2: Insert test data
	t.Logf("2. Inserting test data for date: %s", testDate.Format("2006-01-02"))
	err = repo.MarkClientForDate(ctx, clientID, testDate)
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}
	t.Log("   Test data inserted successfully")

	// Step 3: Show all dates for this client
	t.Log("3. All dates for this client:")
	rows, err := pool.Query(ctx, "SELECT sync_date FROM sync_logs WHERE client_id = $1 ORDER BY sync_date", clientID)
	if err != nil {
		t.Logf("Error querying dates: %v", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var date time.Time
			if err := rows.Scan(&date); err != nil {
				t.Logf("Error scanning date: %v", err)
			} else {
				t.Logf("   - %s", date.Format("2006-01-02"))
			}
		}
	}

	// Step 4: Test the GetSyncedDates function
	t.Log("4. Testing GetSyncedDates function:")
	startDate := time.Date(2023, 11, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2023, 11, 2, 0, 0, 0, 0, time.UTC)

	t.Logf("   Querying for dates between %s and %s",
		startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))

	dates, err := repo.GetSyncedDates(ctx, clientID, startDate, endDate)
	if err != nil {
		t.Fatalf("GetSyncedDates failed: %v", err)
	}

	t.Logf("   Found %d dates:", len(dates))
	for i, date := range dates {
		t.Logf("     %d: %s", i+1, date.Format("2006-01-02"))
	}

	// Step 5: Manual SQL query for comparison
	t.Log("5. Manual SQL query for comparison:")
	manualRows, err := pool.Query(ctx, `
		SELECT sync_date
		FROM sync_logs
		WHERE client_id = $1
		AND sync_date >= $2
		AND sync_date < $3
		ORDER BY sync_date ASC
	`, clientID, startDate, endDate)
	if err != nil {
		t.Logf("Manual query failed: %v", err)
	} else {
		defer manualRows.Close()
		manualCount := 0
		for manualRows.Next() {
			manualCount++
			var date time.Time
			if err := manualRows.Scan(&date); err != nil {
				t.Logf("Error scanning manual query: %v", err)
			} else {
				t.Logf("   Manual query found: %s", date.Format("2006-01-02"))
			}
		}
		t.Logf("   Manual query found %d rows", manualCount)
	}

	t.Log("=== Debug Complete ===")
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

func TestSyncLogRepository_DeleteByClientIDAndDateRange(t *testing.T) {
	pool := init_test.SetupTestDB(t)
	defer init_test.CleanupTestDB()

	repo := repositories.NewSyncLogRepository(pool)
	ctx := context.Background()

	// Create test data
	clientID := "test-client-789"
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC)

	// Cleanup after test
	defer func() {
		init_test.CleanupTestDataByClientID(t, pool, clientID)
		init_test.CleanupTestDataByAssetName(t, pool, "Test Asset")
		init_test.CleanupTestDataByAssetCategoryName(t, pool, "Test Category")
	}()

	// Insert test sync logs
	syncDates := []time.Time{
		time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC),
		time.Date(2024, 1, 4, 0, 0, 0, 0, time.UTC), // Outside range
	}

	for _, date := range syncDates {
		_, err := pool.Exec(ctx, "INSERT INTO sync_logs (client_id, sync_date) VALUES ($1, $2)",
			clientID, date)
		assert.NoError(t, err)
	}

	// Insert sync logs for different client (should not be deleted)
	_, err := pool.Exec(ctx, "INSERT INTO sync_logs (client_id, sync_date) VALUES ($1, $2)",
		"other-client", time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC))
	assert.NoError(t, err)

	// Verify initial data
	var count int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM sync_logs WHERE client_id = $1", clientID).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 4, count)

	// Test deletion
	err = repo.DeleteByClientIDAndDateRange(ctx, clientID, startDate, endDate)
	assert.NoError(t, err)

	// Verify deletions
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM sync_logs WHERE client_id = $1", clientID).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count) // Only the one outside the range should remain

	// Verify other client's data is untouched
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM sync_logs WHERE client_id = $1", "other-client").Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count)

	// Verify the remaining sync log is the one outside the range
	var remainingDate time.Time
	err = pool.QueryRow(ctx, "SELECT sync_date FROM sync_logs WHERE client_id = $1", clientID).Scan(&remainingDate)
	assert.NoError(t, err)
	assert.Equal(t, time.Date(2024, 1, 4, 0, 0, 0, 0, time.UTC), remainingDate)
}
