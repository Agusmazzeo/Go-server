package repositories_test

import (
	"context"
	"server/src/models"
	"server/src/repositories"
	"testing"
	"time"

	"server/tests/init_test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHoldingRepository(t *testing.T) {
	// Setup test database connection
	db := init_test.SetupTestDB(t)

	// Create repository instance
	repo := repositories.NewHoldingRepository(db)
	assetRepo := repositories.NewAssetRepository(db)
	categoryRepo := repositories.NewAssetCategoryRepository(db)

	// Cleanup test data after test
	defer func() {
		init_test.CleanupTestDataByClientID(t, db, "test-client-1")
	}()

	// Test cases
	t.Run("Create and GetByClientID", func(t *testing.T) {
		ctx := context.Background()
		clientID := "test-client-1"
		category := &models.AssetCategory{
			Name:        "Test Category",
			Description: "Test Description",
		}
		err := categoryRepo.Create(ctx, category, nil)
		require.NoError(t, err)

		asset := &models.Asset{
			ExternalID: "EXT-001",
			Name:       "Test Asset",
			AssetType:  "STOCK",
			CategoryID: category.ID,
			Currency:   "USD",
		}
		err = assetRepo.Create(ctx, asset, nil)
		require.NoError(t, err)

		holding := &models.Holding{
			ClientID: clientID,
			AssetID:  asset.ID,
			Units:    100,
			Value:    1000.0,
			Date:     time.Now(),
		}

		// Test Create without transaction
		err = repo.Create(ctx, holding, nil)
		require.NoError(t, err)

		// Test Create with transaction
		holding2 := &models.Holding{
			ClientID: clientID,
			AssetID:  asset.ID,
			Units:    200,
			Value:    2000.0,
			Date:     time.Now().Add(-time.Hour * 24),
		}

		tx, err := db.Begin(ctx)
		require.NoError(t, err)
		defer func() {
			if err != nil {
				_ = tx.Rollback(ctx)
			}
		}()

		err = repo.Create(ctx, holding2, tx)
		require.NoError(t, err)

		err = tx.Commit(ctx)
		require.NoError(t, err)

		// Test GetByClientID
		holdings, err := repo.GetByClientID(ctx, clientID, time.Now().Add(-time.Hour*24), time.Now())
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(holdings), 2)
		assert.Equal(t, clientID, holdings[0].ClientID)
		assert.Equal(t, holding.AssetID, holdings[0].AssetID)
		assert.Equal(t, holding.Units, holdings[0].Units)
		assert.Equal(t, holding.Value, holdings[0].Value)
	})

	t.Run("GetByClientID for non-existent client", func(t *testing.T) {
		ctx := context.Background()
		nonExistentClientID := "non-existent-client"

		holdings, err := repo.GetByClientID(ctx, nonExistentClientID, time.Now(), time.Now())
		require.NoError(t, err)
		assert.Empty(t, holdings)
	})
}

func TestHoldingRepository_DeleteByClientIDAndDateRange(t *testing.T) {
	pool := init_test.SetupTestDB(t)
	defer init_test.CleanupTestDB()

	repo := repositories.NewHoldingRepository(pool)
	ctx := context.Background()

	// Create test data
	clientID := "test-client-123"
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC)
	init_test.CleanupTestDataByClientID(t, pool, clientID)
	init_test.CleanupTestDataByAssetName(t, pool, "Test Asset")
	init_test.CleanupTestDataByAssetCategoryName(t, pool, "Test Category")
	// Cleanup after test
	defer func() {
		init_test.CleanupTestDataByClientID(t, pool, clientID)
		init_test.CleanupTestDataByAssetName(t, pool, "Test Asset")
		init_test.CleanupTestDataByAssetCategoryName(t, pool, "Test Category")
	}()

	// Create asset category first
	_, err := pool.Exec(ctx, "INSERT INTO asset_categories (name, description) VALUES ($1, $2)",
		"Test Category", "Test Category Description")
	assert.NoError(t, err)

	// Get the generated category ID
	var categoryID int
	err = pool.QueryRow(ctx, "SELECT id FROM asset_categories WHERE name = $1", "Test Category").Scan(&categoryID)
	assert.NoError(t, err)

	// Create asset
	_, err = pool.Exec(ctx, "INSERT INTO assets (external_id, name, asset_type, category_id, currency) VALUES ($1, $2, $3, $4, $5)",
		"test-asset", "Test Asset", "STOCK", categoryID, "PESOS")
	assert.NoError(t, err)

	// Get the generated asset ID
	var assetID int
	err = pool.QueryRow(ctx, "SELECT id FROM assets WHERE external_id = $1", "test-asset").Scan(&assetID)
	assert.NoError(t, err)

	// Insert test holdings
	holdings := []struct {
		date  time.Time
		value float64
	}{
		{time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), 100.0},
		{time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), 200.0},
		{time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC), 300.0},
		{time.Date(2024, 1, 4, 0, 0, 0, 0, time.UTC), 400.0}, // Outside range
	}

	for _, h := range holdings {
		_, err := pool.Exec(ctx, "INSERT INTO holdings (client_id, asset_id, value, units, date, created_at) VALUES ($1, $2, $3, $4, $5, $6)",
			clientID, assetID, h.value, 10.0, h.date, time.Now())
		assert.NoError(t, err)
	}

	// Insert holdings for different client (should not be deleted)
	_, err = pool.Exec(ctx, "INSERT INTO holdings (client_id, asset_id, value, units, date, created_at) VALUES ($1, $2, $3, $4, $5, $6)",
		"other-client", assetID, 500.0, 10.0, time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), time.Now())
	assert.NoError(t, err)

	// Verify initial data
	var count int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM holdings WHERE client_id = $1", clientID).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 4, count)

	// Test deletion
	err = repo.DeleteByClientIDAndDateRange(ctx, clientID, startDate, endDate, nil)
	assert.NoError(t, err)

	// Verify deletions
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM holdings WHERE client_id = $1", clientID).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count) // Only the one outside the range should remain

	// Verify other client's data is untouched
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM holdings WHERE client_id = $1", "other-client").Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count)

	// Verify the remaining holding is the one outside the range
	var remainingDate time.Time
	err = pool.QueryRow(ctx, "SELECT date FROM holdings WHERE client_id = $1", clientID).Scan(&remainingDate)
	assert.NoError(t, err)
	assert.Equal(t, time.Date(2024, 1, 4, 0, 0, 0, 0, time.UTC), remainingDate)
}
