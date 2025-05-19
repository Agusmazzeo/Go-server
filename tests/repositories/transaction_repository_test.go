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

func TestTransactionRepository(t *testing.T) {
	// Setup test database connection
	db := test_init.SetupTestDB(t)
	defer test_init.TruncateTables(t, db)

	// Create repository instance
	repo := repositories.NewTransactionRepository(db)
	categoryRepo := repositories.NewAssetCategoryRepository(db)
	assetRepo := repositories.NewAssetRepository(db)
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

		transaction := &models.Transaction{
			ClientID:        clientID,
			AssetID:         asset.ID,
			TransactionType: "BUY",
			Units:           10,
			PricePerUnit:    100.0,
			TotalValue:      1000.0,
			Date:            time.Now(),
		}

		// Test Create without transaction
		err = repo.Create(ctx, transaction, nil)
		require.NoError(t, err)

		// Test Create with transaction
		transaction2 := &models.Transaction{
			ClientID:        clientID,
			AssetID:         asset.ID,
			TransactionType: "SELL",
			Units:           5,
			PricePerUnit:    110.0,
			TotalValue:      550.0,
			Date:            time.Now(),
		}

		tx, err := db.Begin(ctx)
		require.NoError(t, err)
		defer func() {
			if err != nil {
				_ = tx.Rollback(ctx)
			}
		}()

		err = repo.Create(ctx, transaction2, tx)
		require.NoError(t, err)

		err = tx.Commit(ctx)
		require.NoError(t, err)

		// Test GetByClientID
		transactions, err := repo.GetByClientID(ctx, clientID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(transactions), 2)
		assert.Equal(t, clientID, transactions[0].ClientID)
		assert.Equal(t, transaction.AssetID, transactions[0].AssetID)
		assert.Equal(t, transaction.TransactionType, transactions[0].TransactionType)
		assert.Equal(t, transaction.Units, transactions[0].Units)
		assert.Equal(t, transaction.PricePerUnit, transactions[0].PricePerUnit)
		assert.Equal(t, transaction.TotalValue, transactions[0].TotalValue)
	})

	t.Run("GetByClientID for non-existent client", func(t *testing.T) {
		ctx := context.Background()
		nonExistentClientID := "non-existent-client"

		transactions, err := repo.GetByClientID(ctx, nonExistentClientID)
		require.NoError(t, err)
		assert.Empty(t, transactions)
	})
}
