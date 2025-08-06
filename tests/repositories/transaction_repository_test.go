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

func TestTransactionRepository_CreateBatch(t *testing.T) {
	// Setup test database connection
	db := init_test.SetupTestDB(t)

	// Create repository instances
	repo := repositories.NewTransactionRepository(db)
	assetRepo := repositories.NewAssetRepository(db)
	categoryRepo := repositories.NewAssetCategoryRepository(db)

	ctx := context.Background()
	clientID := "test-client-batch-tx"

	// Cleanup test data after test
	defer func() {
		init_test.CleanupTestDataByClientID(t, db, clientID)
		init_test.CleanupTestDataByAssetName(t, db, "Batch TX Asset 1")
		init_test.CleanupTestDataByAssetName(t, db, "Batch TX Asset 2")
		init_test.CleanupTestDataByAssetCategoryName(t, db, "Batch TX Category")
	}()

	// Create test category
	category := &models.AssetCategory{
		Name:        "Batch TX Category",
		Description: "Category for batch transaction testing",
	}
	err := categoryRepo.Create(ctx, category, nil)
	require.NoError(t, err)

	// Create test assets
	asset1 := &models.Asset{
		ExternalID: "BATCH-TX-001",
		Name:       "Batch TX Asset 1",
		AssetType:  "STOCK",
		CategoryID: category.ID,
		Currency:   "USD",
	}
	err = assetRepo.Create(ctx, asset1, nil)
	require.NoError(t, err)

	asset2 := &models.Asset{
		ExternalID: "BATCH-TX-002",
		Name:       "Batch TX Asset 2",
		AssetType:  "BOND",
		CategoryID: category.ID,
		Currency:   "USD",
	}
	err = assetRepo.Create(ctx, asset2, nil)
	require.NoError(t, err)

	t.Run("CreateBatch with multiple transactions", func(t *testing.T) {
		// Create batch of transactions
		transactions := []models.Transaction{
			{
				ClientID:        clientID,
				AssetID:         asset1.ID,
				TransactionType: "BUY",
				Units:           100.0,
				PricePerUnit:    10.0,
				TotalValue:      1000.0,
				Date:            time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			{
				ClientID:        clientID,
				AssetID:         asset2.ID,
				TransactionType: "BUY",
				Units:           50.0,
				PricePerUnit:    10.0,
				TotalValue:      500.0,
				Date:            time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			},
			{
				ClientID:        clientID,
				AssetID:         asset1.ID,
				TransactionType: "SELL",
				Units:           25.0,
				PricePerUnit:    12.0,
				TotalValue:      300.0,
				Date:            time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
			},
		}

		// Execute batch create
		err := repo.CreateBatch(ctx, transactions, nil)
		require.NoError(t, err)

		// Verify all transactions were created
		startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC)

		createdTransactions, err := repo.GetByClientID(ctx, clientID, startDate, endDate)
		require.NoError(t, err)
		assert.Len(t, createdTransactions, 3)

		// Verify the transactions have correct data
		transactionMap := make(map[string]models.Transaction)
		for _, tx := range createdTransactions {
			key := fmt.Sprintf("%d-%s-%s", tx.AssetID, tx.TransactionType, tx.Date.Format("2006-01-02"))
			transactionMap[key] = tx
		}

		// Check first transaction
		key1 := fmt.Sprintf("%d-%s-%s", asset1.ID, "BUY", "2024-01-01")
		tx1, exists := transactionMap[key1]
		assert.True(t, exists)
		assert.Equal(t, 100.0, tx1.Units)
		assert.Equal(t, 10.0, tx1.PricePerUnit)
		assert.Equal(t, 1000.0, tx1.TotalValue)

		// Check second transaction
		key2 := fmt.Sprintf("%d-%s-%s", asset2.ID, "BUY", "2024-01-01")
		tx2, exists := transactionMap[key2]
		assert.True(t, exists)
		assert.Equal(t, 50.0, tx2.Units)
		assert.Equal(t, 10.0, tx2.PricePerUnit)
		assert.Equal(t, 500.0, tx2.TotalValue)

		// Check third transaction
		key3 := fmt.Sprintf("%d-%s-%s", asset1.ID, "SELL", "2024-01-02")
		tx3, exists := transactionMap[key3]
		assert.True(t, exists)
		assert.Equal(t, 25.0, tx3.Units)
		assert.Equal(t, 12.0, tx3.PricePerUnit)
		assert.Equal(t, 300.0, tx3.TotalValue)
	})

	t.Run("CreateBatch with empty array", func(t *testing.T) {
		// Should handle empty array gracefully
		err := repo.CreateBatch(ctx, []models.Transaction{}, nil)
		require.NoError(t, err)
	})

	t.Run("CreateBatch with transaction context", func(t *testing.T) {
		// Test with explicit transaction
		tx, err := db.Begin(ctx)
		require.NoError(t, err)
		defer tx.Rollback(ctx)

		transactions := []models.Transaction{
			{
				ClientID:        clientID,
				AssetID:         asset1.ID,
				TransactionType: "BUY",
				Units:           75.0,
				PricePerUnit:    15.0,
				TotalValue:      1125.0,
				Date:            time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC),
			},
		}

		err = repo.CreateBatch(ctx, transactions, tx)
		require.NoError(t, err)

		// Commit the transaction
		err = tx.Commit(ctx)
		require.NoError(t, err)

		// Verify the transaction was created
		startDate := time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2024, 1, 11, 0, 0, 0, 0, time.UTC)

		createdTransactions, err := repo.GetByClientID(ctx, clientID, startDate, endDate)
		require.NoError(t, err)
		assert.Len(t, createdTransactions, 1)
		assert.Equal(t, 75.0, createdTransactions[0].Units)
		assert.Equal(t, 15.0, createdTransactions[0].PricePerUnit)
		assert.Equal(t, 1125.0, createdTransactions[0].TotalValue)
	})

	t.Run("CreateBatch with large batch", func(t *testing.T) {
		// Test with a larger batch to verify performance
		batchSize := 100
		transactions := make([]models.Transaction, batchSize)

		for i := 0; i < batchSize; i++ {
			transactions[i] = models.Transaction{
				ClientID:        clientID,
				AssetID:         asset1.ID,
				TransactionType: "BUY",
				Units:           float64(i + 1),
				PricePerUnit:    10.0,
				TotalValue:      float64((i + 1) * 10),
				Date:            time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC),
			}
		}

		// Execute batch create
		err := repo.CreateBatch(ctx, transactions, nil)
		require.NoError(t, err)

		// Verify all transactions were created
		startDate := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
		endDate := time.Date(2024, 2, 2, 0, 0, 0, 0, time.UTC)

		createdTransactions, err := repo.GetByClientID(ctx, clientID, startDate, endDate)
		require.NoError(t, err)
		assert.Len(t, createdTransactions, batchSize)

		// Verify some sample transactions
		assert.Equal(t, 1.0, createdTransactions[0].Units)
		assert.Equal(t, 10.0, createdTransactions[0].TotalValue)

		// Find the last transaction (should have units = 100)
		var lastTransaction *models.Transaction
		for _, tx := range createdTransactions {
			if tx.Units == 100.0 {
				lastTransaction = &tx
				break
			}
		}
		require.NotNil(t, lastTransaction)
		assert.Equal(t, 1000.0, lastTransaction.TotalValue)
	})
}
