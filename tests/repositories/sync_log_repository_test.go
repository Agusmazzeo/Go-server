package repositories_test

import (
	"context"
	"server/src/repositories"
	"testing"
	"time"

	"server/tests/repositories/test_init"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSyncLogRepository(t *testing.T) {
	// Setup test database connection
	db := test_init.SetupTestDB(t)
	defer test_init.TruncateTables(t)

	// Create repository instance
	repo := repositories.NewSyncLogRepository(db)

	// Test cases
	t.Run("Insert and GetLastSyncDate", func(t *testing.T) {
		ctx := context.Background()
		clientID := "test-client-1"
		syncDate := time.Now()

		// Test Insert
		err := repo.Insert(ctx, clientID, syncDate)
		require.NoError(t, err)

		// Test GetLastSyncDate
		lastSyncDate, err := repo.GetLastSyncDate(ctx, clientID)
		require.NoError(t, err)
		assert.NotNil(t, lastSyncDate)
		assert.True(t, lastSyncDate.Before(syncDate))
	})

	t.Run("GetLastSyncDate for non-existent client", func(t *testing.T) {
		ctx := context.Background()
		nonExistentClientID := "non-existent-client"

		lastSyncDate, err := repo.GetLastSyncDate(ctx, nonExistentClientID)
		require.NoError(t, err)
		assert.Nil(t, lastSyncDate)
	})
}
