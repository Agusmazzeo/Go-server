package repositories_test

import (
	"context"
	"fmt"
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

func TestHoldingRepository_CreateBatch(t *testing.T) {
	// Setup test database connection
	db := init_test.SetupTestDB(t)

	// Create repository instances
	repo := repositories.NewHoldingRepository(db)
	assetRepo := repositories.NewAssetRepository(db)
	categoryRepo := repositories.NewAssetCategoryRepository(db)

	ctx := context.Background()
	clientID := "test-client-batch"

	// Cleanup test data after test
	defer func() {
		init_test.CleanupTestDataByClientID(t, db, clientID)
		init_test.CleanupTestDataByAssetName(t, db, "Batch Asset 1")
		init_test.CleanupTestDataByAssetName(t, db, "Batch Asset 2")
		init_test.CleanupTestDataByAssetCategoryName(t, db, "Batch Category")
	}()

	// Create test category
	category := &models.AssetCategory{
		Name:        "Batch Category",
		Description: "Category for batch testing",
	}
	err := categoryRepo.Create(ctx, category, nil)
	require.NoError(t, err)

	// Create test assets
	asset1 := &models.Asset{
		ExternalID: "BATCH-001",
		Name:       "Batch Asset 1",
		AssetType:  "STOCK",
		CategoryID: category.ID,
		Currency:   "USD",
	}
	err = assetRepo.Create(ctx, asset1, nil)
	require.NoError(t, err)

	asset2 := &models.Asset{
		ExternalID: "BATCH-002",
		Name:       "Batch Asset 2",
		AssetType:  "BOND",
		CategoryID: category.ID,
		Currency:   "USD",
	}
	err = assetRepo.Create(ctx, asset2, nil)
	require.NoError(t, err)

	t.Run("CreateBatch with multiple holdings", func(t *testing.T) {
		// Create batch of holdings
		holdings := []models.Holding{
			{
				ClientID: clientID,
				AssetID:  asset1.ID,
				Units:    100.0,
				Value:    1000.0,
				Date:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			{
				ClientID: clientID,
				AssetID:  asset2.ID,
				Units:    50.0,
				Value:    500.0,
				Date:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			{
				ClientID: clientID,
				AssetID:  asset1.ID,
				Units:    200.0,
				Value:    2000.0,
				Date:     time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
			},
		}

		// Execute batch create
		err := repo.CreateBatch(ctx, holdings, nil)
		require.NoError(t, err)

		// Verify all holdings were created
		startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC)

		createdHoldings, err := repo.GetByClientID(ctx, clientID, startDate, endDate)
		require.NoError(t, err)
		assert.Len(t, createdHoldings, 3)

		// Verify the holdings have correct data
		holdingMap := make(map[string]models.Holding)
		for _, h := range createdHoldings {
			key := fmt.Sprintf("%d-%s", h.AssetID, h.Date.Format("2006-01-02"))
			holdingMap[key] = h
		}

		// Check first holding
		key1 := fmt.Sprintf("%d-%s", asset1.ID, "2024-01-01")
		holding1, exists := holdingMap[key1]
		assert.True(t, exists)
		assert.Equal(t, 100.0, holding1.Units)
		assert.Equal(t, 1000.0, holding1.Value)

		// Check second holding
		key2 := fmt.Sprintf("%d-%s", asset2.ID, "2024-01-01")
		holding2, exists := holdingMap[key2]
		assert.True(t, exists)
		assert.Equal(t, 50.0, holding2.Units)
		assert.Equal(t, 500.0, holding2.Value)

		// Check third holding
		key3 := fmt.Sprintf("%d-%s", asset1.ID, "2024-01-02")
		holding3, exists := holdingMap[key3]
		assert.True(t, exists)
		assert.Equal(t, 200.0, holding3.Units)
		assert.Equal(t, 2000.0, holding3.Value)
	})

	t.Run("CreateBatch with conflict resolution", func(t *testing.T) {
		// First, create a holding
		initialHolding := models.Holding{
			ClientID: clientID,
			AssetID:  asset1.ID,
			Units:    75.0,
			Value:    750.0,
			Date:     time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC),
		}
		err := repo.Create(ctx, &initialHolding, nil)
		require.NoError(t, err)

		// Now create a batch with a conflicting holding (same client_id, asset_id, date)
		holdings := []models.Holding{
			{
				ClientID: clientID,
				AssetID:  asset1.ID,
				Units:    150.0, // Different values
				Value:    1500.0,
				Date:     time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC), // Same date as initial
			},
			{
				ClientID: clientID,
				AssetID:  asset2.ID,
				Units:    25.0,
				Value:    250.0,
				Date:     time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC),
			},
		}

		// Execute batch create - should update the existing holding
		err = repo.CreateBatch(ctx, holdings, nil)
		require.NoError(t, err)

		// Verify the conflict was resolved (updated, not duplicated)
		startDate := time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2024, 1, 6, 0, 0, 0, 0, time.UTC)

		createdHoldings, err := repo.GetByClientID(ctx, clientID, startDate, endDate)
		require.NoError(t, err)
		assert.Len(t, createdHoldings, 2) // Should have exactly 2 holdings, not 3

		// Find the updated holding
		var updatedHolding *models.Holding
		for _, h := range createdHoldings {
			if h.AssetID == asset1.ID {
				updatedHolding = &h
				break
			}
		}

		require.NotNil(t, updatedHolding)
		assert.Equal(t, 150.0, updatedHolding.Units) // Should have the new values
		assert.Equal(t, 1500.0, updatedHolding.Value)
	})

	t.Run("CreateBatch with empty array", func(t *testing.T) {
		// Should handle empty array gracefully
		err := repo.CreateBatch(ctx, []models.Holding{}, nil)
		require.NoError(t, err)
	})

	t.Run("CreateBatch with transaction", func(t *testing.T) {
		// Test with explicit transaction
		tx, err := db.Begin(ctx)
		require.NoError(t, err)
		defer tx.Rollback(ctx)

		holdings := []models.Holding{
			{
				ClientID: clientID,
				AssetID:  asset1.ID,
				Units:    300.0,
				Value:    3000.0,
				Date:     time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC),
			},
		}

		err = repo.CreateBatch(ctx, holdings, tx)
		require.NoError(t, err)

		// Commit the transaction
		err = tx.Commit(ctx)
		require.NoError(t, err)

		// Verify the holding was created
		startDate := time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2024, 1, 11, 0, 0, 0, 0, time.UTC)

		createdHoldings, err := repo.GetByClientID(ctx, clientID, startDate, endDate)
		require.NoError(t, err)
		assert.Len(t, createdHoldings, 1)
		assert.Equal(t, 300.0, createdHoldings[0].Units)
	})
}
