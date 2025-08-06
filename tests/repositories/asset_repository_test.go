package repositories_test

import (
	"context"
	"fmt"
	"server/src/models"
	"server/src/repositories"
	"strings"
	"testing"

	"server/tests/init_test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssetRepository(t *testing.T) {
	// Setup test database connection
	db := init_test.SetupTestDB(t)

	// Create repository instance
	repo := repositories.NewAssetRepository(db)
	categoryRepo := repositories.NewAssetCategoryRepository(db)
	ctx := context.Background()

	// Create test category
	category := &models.AssetCategory{
		Name:        "Test Category",
		Description: "Test Description",
	}
	err := categoryRepo.Create(ctx, category, nil)
	require.NoError(t, err)

	// Cleanup test data after test
	defer func() {
		init_test.CleanupTestDataByClientID(t, db, "test-client")
	}()
	// Test cases
	t.Run("Create and GetByID", func(t *testing.T) {

		asset := &models.Asset{
			ExternalID: "EXT-001",
			Name:       "Test Asset",
			AssetType:  "STOCK",
			CategoryID: category.ID,
			Currency:   "USD",
		}

		// Test Create without transaction
		err = repo.Create(ctx, asset, nil)
		require.NoError(t, err)

		// Test Create with transaction
		asset2 := &models.Asset{
			ExternalID: "EXT-002",
			Name:       "Test Asset 2",
			AssetType:  "BOND",
			CategoryID: category.ID,
			Currency:   "EUR",
		}

		tx, err := db.Begin(ctx)
		require.NoError(t, err)
		defer func() {
			if err != nil {
				_ = tx.Rollback(ctx)
			}
		}()

		err = repo.Create(ctx, asset2, tx)
		require.NoError(t, err)

		err = tx.Commit(ctx)
		require.NoError(t, err)

		// Test GetByID
		retrievedAsset, err := repo.GetByID(ctx, asset.ID)
		require.NoError(t, err)
		assert.Equal(t, asset.ExternalID, retrievedAsset.ExternalID)
		assert.Equal(t, asset.Name, retrievedAsset.Name)
		assert.Equal(t, asset.AssetType, retrievedAsset.AssetType)
		assert.Equal(t, asset.CategoryID, retrievedAsset.CategoryID)
		assert.Equal(t, asset.Currency, retrievedAsset.Currency)
	})

	t.Run("GetAll", func(t *testing.T) {

		// Create multiple assets
		assets := []*models.Asset{
			{
				ExternalID: "EXT-003",
				Name:       "Asset 1",
				AssetType:  "STOCK",
				CategoryID: category.ID,
				Currency:   "USD",
			},
			{
				ExternalID: "EXT-004",
				Name:       "Asset 2",
				AssetType:  "BOND",
				CategoryID: category.ID,
				Currency:   "EUR",
			},
		}

		for _, asset := range assets {
			err := repo.Create(ctx, asset, nil)
			require.NoError(t, err)
		}

		// Test GetAll
		retrievedAssets, err := repo.GetAll(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(retrievedAssets), len(assets))
	})

	t.Run("GetByID for non-existent asset", func(t *testing.T) {
		nonExistentID := 999999

		asset, err := repo.GetByID(ctx, nonExistentID)
		require.Error(t, err)
		assert.Nil(t, asset)
	})
}

func TestAssetRepository_CreateBatch(t *testing.T) {
	// Setup test database connection
	db := init_test.SetupTestDB(t)

	// Create repository instances
	repo := repositories.NewAssetRepository(db)
	categoryRepo := repositories.NewAssetCategoryRepository(db)

	ctx := context.Background()

	// Create test categories first
	categories := []models.AssetCategory{
		{
			Name:        "Batch Asset Category 1",
			Description: "First category for batch assets",
		},
		{
			Name:        "Batch Asset Category 2",
			Description: "Second category for batch assets",
		},
	}

	err := categoryRepo.CreateBatch(ctx, categories, nil)
	require.NoError(t, err)

	// Get the created categories to use their IDs
	category1, err := categoryRepo.GetByName(ctx, "Batch Asset Category 1")
	require.NoError(t, err)
	require.NotNil(t, category1)

	category2, err := categoryRepo.GetByName(ctx, "Batch Asset Category 2")
	require.NoError(t, err)
	require.NotNil(t, category2)

	// Cleanup test data after test
	defer func() {
		init_test.CleanupTestDataByAssetName(t, db, "Batch Asset 1")
		init_test.CleanupTestDataByAssetName(t, db, "Batch Asset 2")
		init_test.CleanupTestDataByAssetName(t, db, "Batch Asset 3")
		init_test.CleanupTestDataByAssetName(t, db, "TX Asset")
		init_test.CleanupTestDataByAssetCategoryName(t, db, "Batch Asset Category 1")
		init_test.CleanupTestDataByAssetCategoryName(t, db, "Batch Asset Category 2")
		// Clean up large batch assets
		for i := 1; i <= 50; i++ {
			init_test.CleanupTestDataByAssetName(t, db, fmt.Sprintf("Large Batch Asset %d", i))
		}
	}()

	t.Run("CreateBatch with multiple assets", func(t *testing.T) {
		// Create batch of assets
		assets := []models.Asset{
			{
				ExternalID: "BATCH-001",
				Name:       "Batch Asset 1",
				AssetType:  "STOCK",
				CategoryID: category1.ID,
				Currency:   "USD",
			},
			{
				ExternalID: "BATCH-002",
				Name:       "Batch Asset 2",
				AssetType:  "BOND",
				CategoryID: category2.ID,
				Currency:   "EUR",
			},
			{
				ExternalID: "BATCH-003",
				Name:       "Batch Asset 3",
				AssetType:  "ETF",
				CategoryID: category1.ID,
				Currency:   "USD",
			},
		}

		// Execute batch create
		err := repo.CreateBatch(ctx, assets, nil)
		require.NoError(t, err)

		// Verify all assets were created
		for _, expectedAsset := range assets {
			// Find by external ID since we don't have a GetByExternalID method
			allAssets, err := repo.GetAll(ctx)
			require.NoError(t, err)

			var foundAsset *models.Asset
			for _, asset := range allAssets {
				if asset.ExternalID == expectedAsset.ExternalID {
					foundAsset = &asset
					break
				}
			}

			require.NotNil(t, foundAsset, "Asset with external_id %s not found", expectedAsset.ExternalID)
			assert.Equal(t, expectedAsset.ExternalID, foundAsset.ExternalID)
			assert.Equal(t, expectedAsset.Name, foundAsset.Name)
			assert.Equal(t, expectedAsset.AssetType, foundAsset.AssetType)
			assert.Equal(t, expectedAsset.CategoryID, foundAsset.CategoryID)
			assert.Equal(t, expectedAsset.Currency, foundAsset.Currency)
			assert.NotZero(t, foundAsset.ID)
		}
	})

	t.Run("CreateBatch with conflict resolution", func(t *testing.T) {
		// First, create an asset
		initialAssets := []models.Asset{
			{
				ExternalID: "CONFLICT-001",
				Name:       "Original Asset Name",
				AssetType:  "STOCK",
				CategoryID: category1.ID,
				Currency:   "USD",
			},
		}

		err := repo.CreateBatch(ctx, initialAssets, nil)
		require.NoError(t, err)

		// Get the original asset to check ID
		allAssets, err := repo.GetAll(ctx)
		require.NoError(t, err)

		var originalAsset *models.Asset
		for _, asset := range allAssets {
			if asset.ExternalID == "CONFLICT-001" {
				originalAsset = &asset
				break
			}
		}
		require.NotNil(t, originalAsset)

		// Now create batch with same external_id but different data (should update)
		conflictAssets := []models.Asset{
			{
				ExternalID: "CONFLICT-001",       // Same external_id
				Name:       "Updated Asset Name", // Different name
				AssetType:  "BOND",               // Different type
				CategoryID: category2.ID,         // Different category
				Currency:   "EUR",                // Different currency
			},
		}

		err = repo.CreateBatch(ctx, conflictAssets, nil)
		require.NoError(t, err)

		// Verify the asset was updated, not duplicated
		allAssets, err = repo.GetAll(ctx)
		require.NoError(t, err)

		var updatedAsset *models.Asset
		for _, asset := range allAssets {
			if asset.ExternalID == "CONFLICT-001" {
				updatedAsset = &asset
				break
			}
		}

		require.NotNil(t, updatedAsset)
		assert.Equal(t, originalAsset.ID, updatedAsset.ID)       // Same ID
		assert.Equal(t, "Updated Asset Name", updatedAsset.Name) // Updated name
		assert.Equal(t, "BOND", updatedAsset.AssetType)          // Updated type
		assert.Equal(t, category2.ID, updatedAsset.CategoryID)   // Updated category
		assert.Equal(t, "EUR", updatedAsset.Currency)            // Updated currency

		// Cleanup this specific asset
		init_test.CleanupTestDataByAssetName(t, db, "Updated Asset Name")
	})

	t.Run("CreateBatch with empty array", func(t *testing.T) {
		// Should handle empty array gracefully
		err := repo.CreateBatch(ctx, []models.Asset{}, nil)
		require.NoError(t, err)
	})

	t.Run("CreateBatch with transaction context", func(t *testing.T) {
		// Test with explicit transaction
		tx, err := db.Begin(ctx)
		require.NoError(t, err)
		defer tx.Rollback(ctx)

		assets := []models.Asset{
			{
				ExternalID: "TX-001",
				Name:       "TX Asset",
				AssetType:  "CRYPTO",
				CategoryID: category1.ID,
				Currency:   "BTC",
			},
		}

		err = repo.CreateBatch(ctx, assets, tx)
		require.NoError(t, err)

		// Commit the transaction
		err = tx.Commit(ctx)
		require.NoError(t, err)

		// Verify the asset was created
		allAssets, err := repo.GetAll(ctx)
		require.NoError(t, err)

		var foundAsset *models.Asset
		for _, asset := range allAssets {
			if asset.ExternalID == "TX-001" {
				foundAsset = &asset
				break
			}
		}

		require.NotNil(t, foundAsset)
		assert.Equal(t, "TX Asset", foundAsset.Name)
		assert.Equal(t, "CRYPTO", foundAsset.AssetType)
		assert.Equal(t, "BTC", foundAsset.Currency)
	})

	t.Run("CreateBatch with large batch", func(t *testing.T) {
		// Test with a larger batch to verify performance
		batchSize := 50
		assets := make([]models.Asset, batchSize)

		for i := 0; i < batchSize; i++ {
			assets[i] = models.Asset{
				ExternalID: fmt.Sprintf("LARGE-BATCH-%03d", i+1),
				Name:       fmt.Sprintf("Large Batch Asset %d", i+1),
				AssetType:  "STOCK",
				CategoryID: category1.ID,
				Currency:   "USD",
			}
		}

		// Execute batch create
		err := repo.CreateBatch(ctx, assets, nil)
		require.NoError(t, err)

		// Verify some sample assets were created
		allAssets, err := repo.GetAll(ctx)
		require.NoError(t, err)

		// Find first and last assets to verify they exist
		var firstAsset, lastAsset *models.Asset
		for _, asset := range allAssets {
			if asset.ExternalID == "LARGE-BATCH-001" {
				firstAsset = &asset
			}
			if asset.ExternalID == "LARGE-BATCH-050" {
				lastAsset = &asset
			}
		}

		require.NotNil(t, firstAsset)
		assert.Equal(t, "Large Batch Asset 1", firstAsset.Name)
		assert.Equal(t, "STOCK", firstAsset.AssetType)

		require.NotNil(t, lastAsset)
		assert.Equal(t, "Large Batch Asset 50", lastAsset.Name)
		assert.Equal(t, "STOCK", lastAsset.AssetType)

		// Count how many of our test assets exist
		testAssetCount := 0
		for _, asset := range allAssets {
			if strings.Contains(asset.ExternalID, "LARGE-BATCH-") {
				testAssetCount++
			}
		}
		assert.Equal(t, batchSize, testAssetCount)
	})
}
