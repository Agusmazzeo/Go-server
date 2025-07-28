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
	SyncDataFromAccount(ctx context.Context, token, accountID string, startDate, endDate time.Time) error
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

func (s *SyncService) SyncDataFromAccount(ctx context.Context, token, accountID string, startDate, endDate time.Time) error {
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

	accountState, err := s.escoService.GetAccountStateWithTransactions(ctx, token, accountID, startDate, endDate, time.Hour*24)
	if err != nil {
		logger.Error(err)
		return err
	}

	if accountState == nil {
		logger.Infof("No account state returned for account %s", accountID)
		return nil
	}

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
			if date.Equal(syncedDate) {
				alreadySynced = true
				break
			}
		}
		if !alreadySynced {
			datesToSync = append(datesToSync, date)
		}
	}

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

	for _, asset := range *accountState.Assets {
		err = s.storeAsset(ctx, &asset)
		if err != nil {
			return fmt.Errorf("error storing asset %s: %w", asset.ID, err)
		}

		// Filter holdings to only include dates in datesToSync
		filteredHoldings := s.filterHoldingsByDates(asset.Holdings, datesToSyncMap)
		logger.Infof("Filtered holdings for asset %s: %d out of %d", asset.ID, len(filteredHoldings), len(asset.Holdings))
		if len(filteredHoldings) > 0 {
			err = s.storeHoldings(ctx, accountID, asset.ID, filteredHoldings)
			if err != nil {
				return fmt.Errorf("error storing holdings for asset %s: %w", asset.ID, err)
			}
		}

		// Filter transactions to only include dates in datesToSync
		filteredTransactions := s.filterTransactionsByDates(asset.Transactions, datesToSyncMap)
		logger.Infof("Filtered transactions for asset %s: %d out of %d", asset.ID, len(filteredTransactions), len(asset.Transactions))
		if len(filteredTransactions) > 0 {
			err = s.storeTransactions(ctx, accountID, asset.ID, filteredTransactions)
			if err != nil {
				return fmt.Errorf("error storing transactions for asset %s: %w", asset.ID, err)
			}
		}

		// Only collect dates that are in datesToSync
		for _, holding := range filteredHoldings {
			dates[*holding.DateRequested] = true
		}
		for _, transaction := range filteredTransactions {
			dates[*transaction.Date] = true
		}
	}

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

	return nil
}

func (s *SyncService) markDatesAsSynced(ctx context.Context, accountID string, dates []time.Time) error {
	logger := utils.LoggerFromContext(ctx)
	logger.Infof("Marking dates as synced for account %s", accountID)
	return s.syncLogRepository.MarkClientForDates(ctx, accountID, dates)
}

func (s *SyncService) storeAsset(ctx context.Context, asset *schemas.Asset) error {
	logger := utils.LoggerFromContext(ctx)
	logger.Infof("Storing asset %s", asset.ID)
	dbAssetCategory, err := s.assetCategoryRepository.GetByName(ctx, asset.Category)
	if err != nil {
		return fmt.Errorf("error getting asset category: %w", err)
	}
	if dbAssetCategory == nil {
		dbAssetCategory = &models.AssetCategory{
			Name: asset.Category,
		}
		err = s.assetCategoryRepository.Create(ctx, dbAssetCategory, nil)
		if err != nil {
			return fmt.Errorf("error creating asset category: %w", err)
		}
	}
	dbAsset := models.Asset{
		ExternalID: asset.ID,
		Name:       asset.Denomination,
		AssetType:  asset.Type,
		CategoryID: dbAssetCategory.ID,
		Currency:   utils.AssetCurrencyPesos,
	}
	err = s.assetRepository.Create(ctx, &dbAsset, nil)
	if err != nil {
		return fmt.Errorf("error creating asset: %w", err)
	}
	asset.ID = strconv.Itoa(dbAsset.ID)
	return nil
}

func (s *SyncService) storeHoldings(ctx context.Context, accountID, assetID string, holdings []schemas.Holding) error {
	logger := utils.LoggerFromContext(ctx)
	logger.Infof("Storing holdings for account %s", accountID)
	assetIDInt, err := strconv.Atoi(assetID)
	if err != nil {
		return err
	}
	for _, holding := range holdings {
		err := s.holdingRepository.Create(ctx, &models.Holding{
			ClientID:  accountID,
			AssetID:   assetIDInt,
			Value:     holding.Value,
			Units:     holding.Units,
			Date:      *holding.DateRequested,
			CreatedAt: time.Now(),
		}, nil)
		if err != nil {
			return fmt.Errorf("error creating holding: %w", err)
		}
	}
	return nil
}

func (s *SyncService) storeTransactions(ctx context.Context, accountID, assetID string, transactions []schemas.Transaction) error {
	logger := utils.LoggerFromContext(ctx)
	logger.Infof("Storing transactions for account %s", accountID)
	assetIDInt, err := strconv.Atoi(assetID)
	if err != nil {
		return fmt.Errorf("error creating transaction: %w", err)
	}
	for _, transaction := range transactions {
		err = s.transactionRepository.Create(ctx, &models.Transaction{
			ClientID:  accountID,
			AssetID:   assetIDInt,
			Units:     transaction.Units,
			Date:      *transaction.Date,
			CreatedAt: time.Now(),
		}, nil)
		if err != nil {
			return fmt.Errorf("error creating transaction: %w", err)
		}
	}
	return nil
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
