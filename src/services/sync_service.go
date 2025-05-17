package services

import (
	"context"
	"fmt"
	"server/src/clients/esco"
	"server/src/models"
	"server/src/repositories"
	"server/src/utils"
	"strconv"
	"sync"
	"time"
)

type SyncServiceI interface {
	SyncDataFromAccount(ctx context.Context, accountID string, startDate, endDate time.Time) error
}

type SyncService struct {
	holdingRepository       repositories.HoldingRepository
	transactionRepository   repositories.TransactionRepository
	assetRepository         repositories.AssetRepository
	assetCategoryRepository repositories.AssetCategoryRepository
	syncLogRepository       repositories.SyncLogRepository

	escoClient esco.ESCOServiceClientI
}

func NewSyncService(
	holdingRepository repositories.HoldingRepository,
	transactionRepository repositories.TransactionRepository,
	assetRepository repositories.AssetRepository,
	assetCategoryRepository repositories.AssetCategoryRepository,
	syncLogRepository repositories.SyncLogRepository,
	escoClient esco.ESCOServiceClientI,
) *SyncService {
	return &SyncService{
		holdingRepository:       holdingRepository,
		transactionRepository:   transactionRepository,
		assetRepository:         assetRepository,
		assetCategoryRepository: assetCategoryRepository,
		syncLogRepository:       syncLogRepository,
		escoClient:              escoClient,
	}
}

func (s *SyncService) SyncDataFromAccount(ctx context.Context, accountID string, startDate, endDate time.Time) error {
	logger := utils.LoggerFromContext(ctx)
	logger.Infof("Starting sync for account %s from %s to %s", accountID, startDate, endDate)

	// Get account details
	account, err := s.escoClient.BuscarCuentas("", accountID)
	if err != nil {
		logger.Errorf("Error fetching account details: %v", err)
		return err
	}
	if len(account) == 0 {
		logger.Errorf("No account found with ID %s", accountID)
		return fmt.Errorf("no account found with ID %s", accountID)
	}

	// Create sync log entry
	if err := s.syncLogRepository.Insert(ctx, accountID, startDate); err != nil {
		logger.Errorf("Error creating sync log: %v", err)
		return err
	}

	var wg sync.WaitGroup
	var syncErr error

	// Fetch and process holdings
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := s.syncHoldings(ctx, account[0], startDate, endDate); err != nil {
			logger.Errorf("Error syncing holdings: %v", err)
			syncErr = err
		}
	}()

	// Fetch and process transactions
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := s.syncTransactions(ctx, account[0], startDate, endDate); err != nil {
			logger.Errorf("Error syncing transactions: %v", err)
			syncErr = err
		}
	}()

	wg.Wait()

	return syncErr
}

func (s *SyncService) syncHoldings(ctx context.Context, account esco.CuentaSchema, startDate, endDate time.Time) error {
	logger := utils.LoggerFromContext(ctx)

	// Get holdings from ESCO
	holdings, err := s.escoClient.GetEstadoCuenta("", account.ID, account.FI, strconv.Itoa(account.N), "0", startDate, false)
	if err != nil {
		return err
	}

	for _, holding := range holdings {
		// Create or get asset category
		category := &models.AssetCategory{
			Name: holding.D,
		}
		if err := s.assetCategoryRepository.Create(ctx, category); err != nil {
			logger.Errorf("Error creating asset category: %v", err)
			continue
		}

		// Create or get asset
		asset := &models.Asset{
			ExternalID: holding.A,
			Name:       holding.D,
			AssetType:  holding.TI,
			CategoryID: category.ID,
			Currency:   holding.M,
		}
		if err := s.assetRepository.Create(ctx, asset); err != nil {
			logger.Errorf("Error creating asset: %v", err)
			continue
		}

		// Create holding
		h := &models.Holding{
			ClientID: account.ID,
			AssetID:  asset.ID,
			Quantity: holding.C,
			Value:    holding.N,
			Date:     startDate,
		}
		if err := s.holdingRepository.Create(ctx, h); err != nil {
			logger.Errorf("Error creating holding: %v", err)
			continue
		}
	}

	return nil
}

func (s *SyncService) syncTransactions(ctx context.Context, account esco.CuentaSchema, startDate, endDate time.Time) error {
	logger := utils.LoggerFromContext(ctx)

	// Get transactions from ESCO
	transactions, err := s.escoClient.GetBoletos("", account.ID, account.FI, strconv.Itoa(account.N), "0", startDate, endDate, false)
	if err != nil {
		return err
	}

	for _, transaction := range transactions {
		// Create or get asset
		asset := &models.Asset{
			ExternalID: transaction.I,
			Name:       transaction.FL,
			AssetType:  transaction.T,
			Currency:   "Pesos",
		}
		if err := s.assetRepository.Create(ctx, asset); err != nil {
			logger.Errorf("Error creating asset: %v", err)
			continue
		}

		// Create transaction
		t := &models.Transaction{
			ClientID:        account.ID,
			AssetID:         asset.ID,
			TransactionType: transaction.T,
			Quantity:        transaction.C,
			PricePerUnit:    transaction.PR,
			TotalValue:      transaction.N,
			Date:            startDate,
		}
		if err := s.transactionRepository.Create(ctx, t); err != nil {
			logger.Errorf("Error creating transaction: %v", err)
			continue
		}
	}

	return nil
}
