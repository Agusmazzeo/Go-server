package services

import (
	"context"
	"fmt"
	"server/src/models"
	"server/src/repositories"
	"server/src/schemas"
	"server/src/utils"
	"strconv"
	"time"
)

type SyncServiceI interface {
	GetDatesToSync(ctx context.Context, token, accountID string, startDate, endDate time.Time) ([]time.Time, error)
	SyncDataFromAccount(ctx context.Context, token, accountID string, startDate, endDate time.Time, refreshCache bool) error
	ForceRefreshData(ctx context.Context, accountID string, startDate, endDate time.Time) error
}

type SyncService struct {
	holdingRepository       repositories.HoldingRepository
	transactionRepository   repositories.TransactionRepository
	assetRepository         repositories.AssetRepository
	assetCategoryRepository repositories.AssetCategoryRepository
	syncLogRepository       repositories.SyncLogRepository

	escoService ESCOServiceI
}

func NewSyncService(
	holdingRepository repositories.HoldingRepository,
	transactionRepository repositories.TransactionRepository,
	assetRepository repositories.AssetRepository,
	assetCategoryRepository repositories.AssetCategoryRepository,
	syncLogRepository repositories.SyncLogRepository,
	escoService ESCOServiceI,
) *SyncService {
	return &SyncService{
		holdingRepository:       holdingRepository,
		transactionRepository:   transactionRepository,
		assetRepository:         assetRepository,
		assetCategoryRepository: assetCategoryRepository,
		syncLogRepository:       syncLogRepository,
		escoService:             escoService,
	}
}

func (s *SyncService) SyncDataFromAccount(ctx context.Context, token, accountID string, startDate, endDate time.Time, refreshCache bool) error {
	logger := utils.LoggerFromContext(ctx)
	logger.Infof("Starting sync for account %s from %s to %s", accountID, startDate, endDate)

	datesToSync, err := s.GetDatesToSync(ctx, token, accountID, startDate, endDate)
	if err != nil {
		return err
	}
	if len(datesToSync) == 0 {
		logger.Infof("Data is already synced for account %s from %s to %s", accountID, startDate, endDate)
		return nil
	}

	accountState, err := s.escoService.GetAccountStateWithTransactions(ctx, token, accountID, startDate, endDate, time.Hour*24, refreshCache)
	if err != nil {
		logger.Error(err)
		return err
	}

	if accountState == nil {
		logger.Infof("No account state returned for account %s", accountID)
		return nil
	}
	logger.Infof("Data to sync: %v", datesToSync)
	err = s.StoreAccountState(ctx, accountID, accountState, datesToSync)
	if err != nil {
		logger.Error(err)
		return err
	}

	return nil
}

func (s *SyncService) GetDatesToSync(ctx context.Context, token, accountID string, startDate, endDate time.Time) ([]time.Time, error) {
	logger := utils.LoggerFromContext(ctx)
	logger.Infof("Checking if data is synced for account %s from %s to %s", accountID, startDate, endDate)
	syncedDates, err := s.syncLogRepository.GetSyncedDates(ctx, accountID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	datesToSync := make([]time.Time, 0)
	for date := startDate; date.Before(endDate); date = date.AddDate(0, 0, 1) {
		alreadySynced := false
		for _, syncedDate := range syncedDates {
			if date.Format("2006-01-02") == syncedDate.Format("2006-01-02") {
				alreadySynced = true
				break
			}
		}
		if !alreadySynced {
			datesToSync = append(datesToSync, date)
		}
	}
	logger.Infof("Dates to sync: %v", len(datesToSync))

	return datesToSync, nil
}

func (s *SyncService) StoreAccountState(ctx context.Context, accountID string, accountState *schemas.AccountState, datesToSync []time.Time) error {
	logger := utils.LoggerFromContext(ctx)
	logger.Infof("Storing account state for account %s", accountID)
	var err error
	dates := make(map[time.Time]bool)

	// Create a map for quick lookup of dates to sync
	datesToSyncMap := make(map[string]bool)
	for _, date := range datesToSync {
		datesToSyncMap[date.Format("2006-01-02")] = true
	}

	// First pass: collect all unique categories and assets for batch creation
	err = s.storeAssetsInBatch(ctx, accountState.Assets)
	if err != nil {
		return fmt.Errorf("error storing assets in batch: %w", err)
	}

	// Second pass: collect all holdings and transactions for batch creation
	allHoldings := make([]models.Holding, 0)
	allTransactions := make([]models.Transaction, 0)
	totalFilteredHoldings := 0
	totalFilteredTransactions := 0

	for _, asset := range *accountState.Assets {
		assetIDInt, err := strconv.Atoi(asset.ID)
		if err != nil {
			return fmt.Errorf("error converting asset ID %s to int: %w", asset.ID, err)
		}

		// Filter and collect holdings
		filteredHoldings := s.filterHoldingsByDates(asset.Holdings, datesToSyncMap)
		totalFilteredHoldings += len(filteredHoldings)

		for _, holding := range filteredHoldings {
			allHoldings = append(allHoldings, models.Holding{
				ClientID:  accountID,
				AssetID:   assetIDInt,
				Value:     holding.Value,
				Units:     holding.Units,
				Date:      *holding.DateRequested,
				CreatedAt: time.Now(),
			})
			dates[*holding.DateRequested] = true
		}

		// Filter and collect transactions
		filteredTransactions := s.filterTransactionsByDates(asset.Transactions, datesToSyncMap)
		totalFilteredTransactions += len(filteredTransactions)

		for _, transaction := range filteredTransactions {
			allTransactions = append(allTransactions, models.Transaction{
				ClientID:  accountID,
				AssetID:   assetIDInt,
				Units:     transaction.Units,
				Date:      *transaction.Date,
				CreatedAt: time.Now(),
			})
			dates[*transaction.Date] = true
		}
	}

	logger.Infof("Collected %d holdings and %d transactions for batch creation", totalFilteredHoldings, totalFilteredTransactions)

	// Third pass: create all holdings in batch
	if len(allHoldings) > 0 {
		logger.Infof("Creating %d holdings in batch", len(allHoldings))
		err = s.holdingRepository.CreateBatch(ctx, allHoldings, nil)
		if err != nil {
			return fmt.Errorf("error creating holdings in batch: %w", err)
		}
	}

	// Fourth pass: create all transactions in batch
	if len(allTransactions) > 0 {
		logger.Infof("Creating %d transactions in batch", len(allTransactions))
		err = s.transactionRepository.CreateBatch(ctx, allTransactions, nil)
		if err != nil {
			return fmt.Errorf("error creating transactions in batch: %w", err)
		}
	}

	// Mark dates as synced
	datesList := make([]time.Time, 0)
	for date := range dates {
		datesList = append(datesList, date)
	}

	if len(datesList) > 0 {
		err = s.markDatesAsSynced(ctx, accountID, datesList)
		if err != nil {
			return fmt.Errorf("error marking dates as synced: %w", err)
		}
	}

	logger.Infof("Successfully stored account state for account %s with %d holdings and %d transactions",
		accountID, len(allHoldings), len(allTransactions))
	return nil
}

// storeAssetsInBatch efficiently stores all categories and assets in batches
func (s *SyncService) storeAssetsInBatch(ctx context.Context, assetsMap *map[string]schemas.Asset) error {
	logger := utils.LoggerFromContext(ctx)
	logger.Infof("Starting batch storage of %d assets", len(*assetsMap))

	// Collect all unique categories
	categoryMap := make(map[string]*models.AssetCategory)
	for _, asset := range *assetsMap {
		if _, exists := categoryMap[asset.Category]; !exists {
			categoryMap[asset.Category] = &models.AssetCategory{
				Name:        asset.Category,
				Description: fmt.Sprintf("Category for %s assets", asset.Category),
			}
		}
	}

	// Create categories in batch
	if len(categoryMap) > 0 {
		categories := make([]models.AssetCategory, 0, len(categoryMap))
		for _, category := range categoryMap {
			categories = append(categories, *category)
		}

		logger.Infof("Creating %d categories in batch", len(categories))
		err := s.assetCategoryRepository.CreateBatch(ctx, categories, nil)
		if err != nil {
			return fmt.Errorf("error creating asset categories in batch: %w", err)
		}

		// Update the category map with the created IDs by fetching them back
		for categoryName := range categoryMap {
			dbCategory, err := s.assetCategoryRepository.GetByName(ctx, categoryName)
			if err != nil {
				return fmt.Errorf("error getting created category %s: %w", categoryName, err)
			}
			if dbCategory != nil {
				categoryMap[categoryName] = dbCategory
			}
		}
	}

	// Collect all assets for batch creation
	assetsToCreate := make([]models.Asset, 0, len(*assetsMap))

	for _, asset := range *assetsMap {
		category := categoryMap[asset.Category]
		if category == nil {
			return fmt.Errorf("category %s not found after batch creation", asset.Category)
		}

		dbAsset := models.Asset{
			ExternalID: asset.ID,
			Name:       asset.Denomination,
			AssetType:  asset.Type,
			CategoryID: category.ID,
			Currency:   utils.AssetCurrencyPesos,
		}
		assetsToCreate = append(assetsToCreate, dbAsset)
	}

	// Create assets in batch
	if len(assetsToCreate) > 0 {
		logger.Infof("Creating %d assets in batch", len(assetsToCreate))
		err := s.assetRepository.CreateBatch(ctx, assetsToCreate, nil)
		if err != nil {
			return fmt.Errorf("error creating assets in batch: %w", err)
		}

		// Update the original assets with their new internal IDs
		// We need to fetch them back since batch insert doesn't return IDs
		allAssets, err := s.assetRepository.GetAll(ctx)
		if err != nil {
			return fmt.Errorf("error getting assets after batch creation: %w", err)
		}

		// Create a map for quick lookup of internal IDs by external ID
		externalToInternalID := make(map[string]int)
		for _, dbAsset := range allAssets {
			externalToInternalID[dbAsset.ExternalID] = dbAsset.ID
		}

		// Update the original assets map with internal IDs
		for externalID, asset := range *assetsMap {
			if internalID, exists := externalToInternalID[asset.ID]; exists {
				asset.ID = strconv.Itoa(internalID)
				(*assetsMap)[externalID] = asset
			}
		}
	}

	logger.Infof("Successfully completed batch storage of assets")
	return nil
}

func (s *SyncService) markDatesAsSynced(ctx context.Context, accountID string, dates []time.Time) error {
	logger := utils.LoggerFromContext(ctx)
	logger.Infof("Marking dates as synced for account %s", accountID)
	return s.syncLogRepository.MarkClientForDates(ctx, accountID, dates)
}

// filterHoldingsByDates filters holdings to only include those with dates in the datesToSync map
func (s *SyncService) filterHoldingsByDates(holdings []schemas.Holding, datesToSync map[string]bool) []schemas.Holding {
	var filteredHoldings []schemas.Holding
	for _, holding := range holdings {
		if holding.DateRequested != nil {
			dateStr := holding.DateRequested.Format("2006-01-02")
			if datesToSync[dateStr] {
				filteredHoldings = append(filteredHoldings, holding)
			}
		}
	}
	return filteredHoldings
}

// filterTransactionsByDates filters transactions to only include those with dates in the datesToSync map
func (s *SyncService) filterTransactionsByDates(transactions []schemas.Transaction, datesToSync map[string]bool) []schemas.Transaction {
	var filteredTransactions []schemas.Transaction
	for _, transaction := range transactions {
		if transaction.Date != nil {
			dateStr := transaction.Date.Format("2006-01-02")
			if datesToSync[dateStr] {
				filteredTransactions = append(filteredTransactions, transaction)
			}
		}
	}
	return filteredTransactions
}

// ForceRefreshData deletes all holdings, transactions, and sync logs for the given client and date range
// This is used when the x-force-refresh header is present to ensure fresh data sync
func (s *SyncService) ForceRefreshData(ctx context.Context, accountID string, startDate, endDate time.Time) error {
	logger := utils.LoggerFromContext(ctx)
	logger.Infof("Force refreshing data for account %s from %s to %s", accountID, startDate, endDate)

	// Delete holdings for the client and date range (will create its own transaction)
	err := s.holdingRepository.DeleteByClientIDAndDateRange(ctx, accountID, startDate, endDate, nil)
	if err != nil {
		return fmt.Errorf("failed to delete holdings: %w", err)
	}

	// Delete transactions for the client and date range (will create its own transaction)
	err = s.transactionRepository.DeleteByClientIDAndDateRange(ctx, accountID, startDate, endDate, nil)
	if err != nil {
		return fmt.Errorf("failed to delete transactions: %w", err)
	}

	// Delete sync logs for the client and date range
	err = s.syncLogRepository.DeleteByClientIDAndDateRange(ctx, accountID, startDate, endDate)
	if err != nil {
		return fmt.Errorf("failed to delete sync logs: %w", err)
	}

	logger.Infof("Successfully force refreshed data for account %s", accountID)
	return nil
}
