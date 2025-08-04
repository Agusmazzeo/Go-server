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

func TestTransactionRepository(t *testing.T) {
	// Setup test database connection
	db := init_test.SetupTestDB(t)

	// Create repository instance
	repo := repositories.NewTransactionRepository(db)
	categoryRepo := repositories.NewAssetCategoryRepository(db)
	assetRepo := repositories.NewAssetRepository(db)

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
		transactions, err := repo.GetByClientID(ctx, clientID, time.Now().Add(-time.Hour*24), time.Now())
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

		transactions, err := repo.GetByClientID(ctx, nonExistentClientID, time.Now().Add(-time.Hour*24), time.Now())
		require.NoError(t, err)
		assert.Empty(t, transactions)
	})
}

func TestTransactionRepository_DeleteByClientIDAndDateRange(t *testing.T) {
	pool := init_test.SetupTestDB(t)
	defer init_test.CleanupTestDB()

	repo := repositories.NewTransactionRepository(pool)
	ctx := context.Background()

	// Create test data
	clientID := "test-client-456"
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

	// Insert test transactions
	transactions := []struct {
		date  time.Time
		value float64
	}{
		{time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC), 100.0},
		{time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), 200.0},
		{time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC), 300.0},
		{time.Date(2024, 1, 4, 0, 0, 0, 0, time.UTC), 400.0}, // Outside range
	}

	for _, txn := range transactions {
		_, err := pool.Exec(ctx, "INSERT INTO transactions (client_id, asset_id, transaction_type, units, price_per_unit, total_value, date, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)",
			clientID, assetID, "BUY", 10.0, txn.value, txn.value, txn.date, time.Now())
		assert.NoError(t, err)
	}

	// Insert transactions for different client (should not be deleted)
	_, err = pool.Exec(ctx, "INSERT INTO transactions (client_id, asset_id, transaction_type, units, price_per_unit, total_value, date, created_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)",
		"other-client", assetID, "BUY", 10.0, 500.0, 500.0, time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC), time.Now())
	assert.NoError(t, err)

	// Verify initial data
	var count int
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM transactions WHERE client_id = $1", clientID).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 4, count)

	// Test deletion
	err = repo.DeleteByClientIDAndDateRange(ctx, clientID, startDate, endDate, nil)
	assert.NoError(t, err)

	// Verify deletions
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM transactions WHERE client_id = $1", clientID).Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count) // Only the one outside the range should remain

	// Verify other client's data is untouched
	err = pool.QueryRow(ctx, "SELECT COUNT(*) FROM transactions WHERE client_id = $1", "other-client").Scan(&count)
	assert.NoError(t, err)
	assert.Equal(t, 1, count)

	// Verify the remaining transaction is the one outside the range
	var remainingDate time.Time
	err = pool.QueryRow(ctx, "SELECT date FROM transactions WHERE client_id = $1", clientID).Scan(&remainingDate)
	assert.NoError(t, err)
	assert.Equal(t, time.Date(2024, 1, 4, 0, 0, 0, 0, time.UTC), remainingDate)
}
