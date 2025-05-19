package repositories_test

import (
	"context"
	"server/src/models"
	"server/src/repositories"
	"testing"
	"time"

	"server/tests/repositories/test_init"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHoldingRepository(t *testing.T) {
	// Setup test database connection
	db := test_init.SetupTestDB(t)
	defer test_init.TruncateTables(t, db)

	// Create repository instance
	repo := repositories.NewHoldingRepository(db)
	assetRepo := repositories.NewAssetRepository(db)
	categoryRepo := repositories.NewAssetCategoryRepository(db)

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
			Date:     time.Now(),
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
		holdings, err := repo.GetByClientID(ctx, clientID)
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

		holdings, err := repo.GetByClientID(ctx, nonExistentClientID)
		require.NoError(t, err)
		assert.Empty(t, holdings)
	})
}
