package services

import (
	"context"
	"server/src/repositories"
	"server/src/utils"
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

	return nil
}
