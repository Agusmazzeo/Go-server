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
	IsDataSynced(ctx context.Context, token, accountID string, startDate, endDate time.Time) (bool, error)
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

	isSynced, err := s.IsDataSynced(ctx, token, accountID, startDate, endDate)
	if err != nil {
		return err
	}
	if isSynced {
		logger.Infof("Data is already synced for account %s from %s to %s", accountID, startDate, endDate)
		return nil
	}

	accountState, err := s.escoService.GetAccountStateWithTransactions(ctx, token, accountID, startDate, endDate, time.Hour*24)
	if err != nil {
		logger.Error(err)
		return err
	}

	err = s.storeAccountState(ctx, accountID, accountState)
	if err != nil {
		logger.Error(err)
		return err
	}

	return nil
}

func (s *SyncService) IsDataSynced(ctx context.Context, token, accountID string, startDate, endDate time.Time) (bool, error) {
	logger := utils.LoggerFromContext(ctx)
	logger.Infof("Checking if data is synced for account %s from %s to %s", accountID, startDate, endDate)
	syncedDates, err := s.syncLogRepository.GetSyncedDates(ctx, accountID, startDate, endDate)
	if err != nil {
		return false, err
	}
	expectedDates := make([]time.Time, 0)
	for date := startDate; date.Before(endDate); date = date.AddDate(0, 0, 1) {
		expectedDates = append(expectedDates, date)
	}
	if len(syncedDates) != len(expectedDates) {
		return false, nil
	}

	return true, nil
}

func (s *SyncService) storeAccountState(ctx context.Context, accountID string, accountState *schemas.AccountState) error {
	logger := utils.LoggerFromContext(ctx)
	logger.Infof("Storing account state for account %s", accountID)
	var err error
	dates := make(map[time.Time]bool)
	for _, asset := range *accountState.Assets {
		err = s.storeAsset(ctx, &asset)
		if err != nil {
			return fmt.Errorf("error storing asset %s: %w", asset.ID, err)
		}
		err = s.storeHoldings(ctx, accountID, asset.ID, asset.Holdings)
		if err != nil {
			return fmt.Errorf("error storing holdings for asset %s: %w", asset.ID, err)
		}
		err = s.storeTransactions(ctx, accountID, asset.ID, asset.Transactions)
		if err != nil {
			return fmt.Errorf("error storing transactions for asset %s: %w", asset.ID, err)
		}
		for _, holding := range asset.Holdings {
			dates[*holding.DateRequested] = true
		}
		for _, transaction := range asset.Transactions {
			dates[*transaction.Date] = true
		}
	}
	datesList := make([]time.Time, 0)
	for date := range dates {
		datesList = append(datesList, date)
	}
	err = s.markDatesAsSynced(ctx, accountID, datesList)
	if err != nil {
		return fmt.Errorf("error marking dates as synced: %w", err)
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
