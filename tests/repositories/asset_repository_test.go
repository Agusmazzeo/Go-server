package repositories_test

import (
	"context"
	"server/src/models"
	"server/src/repositories"
	"testing"

	"server/tests/init_test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAssetRepository(t *testing.T) {
	// Setup test database connection
	db := init_test.SetupTestDB(t)
	defer init_test.TruncateTables(t, db)

	// Create repository instance
	repo := repositories.NewAssetRepository(db)
	categoryRepo := repositories.NewAssetCategoryRepository(db)
	ctx := context.Background()
	category := &models.AssetCategory{
		Name:        "Test Category",
		Description: "Test Description",
	}
	err := categoryRepo.Create(ctx, category, nil)
	require.NoError(t, err)
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
