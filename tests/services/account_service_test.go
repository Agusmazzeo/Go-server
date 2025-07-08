package services_test

import (
	"context"
	"server/src/models"
	"server/src/repositories"
	"server/src/services"
	"testing"
	"time"

	"server/tests/init_test"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccountService(t *testing.T) {
	// Setup test database connection
	db := init_test.SetupTestDB(t)
	defer init_test.TruncateTables(t, db)

	// Create repositories
	holdingRepo := repositories.NewHoldingRepository(db)
	transactionRepo := repositories.NewTransactionRepository(db)
	assetRepo := repositories.NewAssetRepository(db)
	categoryRepo := repositories.NewAssetCategoryRepository(db)

	// Create service instance
	accountService := services.NewAccountService(holdingRepo, transactionRepo, assetRepo)

	// Test cases
	t.Run("GetAccountState", func(t *testing.T) {
		ctx := context.Background()
		clientID := "test-client-1"

		// Create test data
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
		err = holdingRepo.Create(ctx, holding, nil)
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
		err = transactionRepo.Create(ctx, transaction, nil)
		require.NoError(t, err)

		// Test GetAccountState
		accountState, err := accountService.GetAccountState(ctx, clientID, time.Now())
		require.NoError(t, err)
		assert.NotNil(t, accountState)
		assert.NotNil(t, accountState.Assets)
		assert.Len(t, *accountState.Assets, 1)

		// Check asset data
		assetState := (*accountState.Assets)[asset.ExternalID]
		assert.Equal(t, asset.ExternalID, assetState.ID)
		assert.Equal(t, asset.AssetType, assetState.Type)
		assert.Equal(t, asset.Name, assetState.Denomination)
		assert.Len(t, assetState.Holdings, 1)
		assert.Len(t, assetState.Transactions, 1)

		// Check holding data
		holdingState := assetState.Holdings[0]
		assert.Equal(t, asset.Currency, holdingState.Currency)
		assert.Equal(t, "$", holdingState.CurrencySign)
		assert.Equal(t, holding.Value, holdingState.Value)
		assert.Equal(t, holding.Units, holdingState.Units)

		// Check transaction data
		transactionState := assetState.Transactions[0]
		assert.Equal(t, asset.Currency, transactionState.Currency)
		assert.Equal(t, "$", transactionState.CurrencySign)
		assert.Equal(t, transaction.TotalValue, transactionState.Value)
		assert.Equal(t, transaction.Units, transactionState.Units)
	})

	t.Run("GetMultiAccountStateWithTransactions", func(t *testing.T) {
		ctx := context.Background()
		clientIDs := []string{"test-client-4", "test-client-5"}

		// Create test data for multiple clients
		category := &models.AssetCategory{
			Name:        "Test Category 4",
			Description: "Test Description 4",
		}
		err := categoryRepo.Create(ctx, category, nil)
		require.NoError(t, err)

		asset := &models.Asset{
			ExternalID: "EXT-004",
			Name:       "Test Asset 4",
			AssetType:  "STOCK",
			CategoryID: category.ID,
			Currency:   "USD",
		}
		err = assetRepo.Create(ctx, asset, nil)
		require.NoError(t, err)

		// Create data for each client
		for _, clientID := range clientIDs {
			holding := &models.Holding{
				ClientID: clientID,
				AssetID:  asset.ID,
				Units:    100,
				Value:    1000.0,
				Date:     time.Now(),
			}
			err = holdingRepo.Create(ctx, holding, nil)
			require.NoError(t, err)
		}

		// Test GetMultiAccountStateWithTransactions
		startDate := time.Now().AddDate(0, 0, -1)
		endDate := time.Now().AddDate(0, 0, 1)
		accountStates, err := accountService.GetMultiAccountStateWithTransactions(ctx, clientIDs, startDate, endDate, time.Hour*24)
		require.NoError(t, err)
		assert.Len(t, accountStates, 2)

		// Check that each account state has the expected data
		for _, accountState := range accountStates {
			assert.NotNil(t, accountState.Assets)
			assert.Len(t, *accountState.Assets, 1)
		}
	})

	t.Run("GetMultiAccountStateByCategory", func(t *testing.T) {
		ctx := context.Background()
		clientIDs := []string{"test-client-6", "test-client-7"}

		// Create test data for multiple clients
		category := &models.AssetCategory{
			Name:        "Test Category 5",
			Description: "Test Description 5",
		}
		err := categoryRepo.Create(ctx, category, nil)
		require.NoError(t, err)

		asset := &models.Asset{
			ExternalID: "EXT-005",
			Name:       "Test Asset 5",
			AssetType:  "BOND",
			CategoryID: category.ID,
			Currency:   "ARS",
		}
		err = assetRepo.Create(ctx, asset, nil)
		require.NoError(t, err)

		// Create data for each client
		for _, clientID := range clientIDs {
			holding := &models.Holding{
				ClientID: clientID,
				AssetID:  asset.ID,
				Units:    100,
				Value:    1000.0,
				Date:     time.Now(),
			}
			err = holdingRepo.Create(ctx, holding, nil)
			require.NoError(t, err)
		}

		// Test GetMultiAccountStateByCategory
		startDate := time.Now().AddDate(0, 0, -1)
		endDate := time.Now().AddDate(0, 0, 1)
		accountStateByCategory, err := accountService.GetMultiAccountStateByCategory(ctx, clientIDs, startDate, endDate, time.Hour*24)
		require.NoError(t, err)
		assert.NotNil(t, accountStateByCategory)
		assert.NotNil(t, accountStateByCategory.AssetsByCategory)
		assert.NotNil(t, accountStateByCategory.TotalHoldingsByDate)
		assert.NotNil(t, accountStateByCategory.TotalTransactionsByDate)
		assert.NotNil(t, accountStateByCategory.CategoryAssets)

		// Check that we have the expected category
		assetsByCategory := *accountStateByCategory.AssetsByCategory
		assert.Contains(t, assetsByCategory, category.Name)
		assert.Len(t, assetsByCategory[category.Name], 2) // One asset per client
	})

	t.Run("Repository Grouping Methods", func(t *testing.T) {
		ctx := context.Background()
		clientIDs := []string{"test-client-8", "test-client-9"}

		// Create test data
		category := &models.AssetCategory{
			Name:        "Test Category 6",
			Description: "Test Description 6",
		}
		err := categoryRepo.Create(ctx, category, nil)
		require.NoError(t, err)

		asset := &models.Asset{
			ExternalID: "EXT-006",
			Name:       "Test Asset 6",
			AssetType:  "STOCK",
			CategoryID: category.ID,
			Currency:   "USD",
		}
		err = assetRepo.Create(ctx, asset, nil)
		require.NoError(t, err)

		// Create holdings and transactions for each client
		for _, clientID := range clientIDs {
			holding := &models.Holding{
				ClientID: clientID,
				AssetID:  asset.ID,
				Units:    100,
				Value:    1000.0,
				Date:     time.Now(),
			}
			err = holdingRepo.Create(ctx, holding, nil)
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
			err = transactionRepo.Create(ctx, transaction, nil)
			require.NoError(t, err)
		}

		// Test repository grouping methods
		startDate := time.Now().AddDate(0, 0, -1)
		endDate := time.Now().AddDate(0, 0, 1)

		// Test GetByClientIDs
		holdings, err := holdingRepo.GetByClientIDs(ctx, clientIDs, startDate, endDate)
		require.NoError(t, err)
		assert.Len(t, holdings, 2)

		transactions, err := transactionRepo.GetByClientIDs(ctx, clientIDs, startDate, endDate)
		require.NoError(t, err)
		assert.Len(t, transactions, 2)

		// Test GetGroupedByCategoryAndDate
		categoryHoldings, err := holdingRepo.GetGroupedByCategoryAndDate(ctx, clientIDs, startDate, endDate)
		require.NoError(t, err)
		assert.Contains(t, categoryHoldings, category.Name)

		categoryTransactions, err := transactionRepo.GetGroupedByCategoryAndDate(ctx, clientIDs, startDate, endDate)
		require.NoError(t, err)
		assert.Contains(t, categoryTransactions, category.Name)

		// Test GetTotalByDate
		totalHoldings, err := holdingRepo.GetTotalByDate(ctx, clientIDs, startDate, endDate)
		require.NoError(t, err)
		assert.NotEmpty(t, totalHoldings)

		totalTransactions, err := transactionRepo.GetTotalByDate(ctx, clientIDs, startDate, endDate)
		require.NoError(t, err)
		assert.NotEmpty(t, totalTransactions)
	})
}
