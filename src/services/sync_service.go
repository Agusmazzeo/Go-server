package services

import (
	"context"
	"server/src/clients/esco"
	"server/src/repositories"
	"time"
)

type SyncServiceI interface {
	SyncDataFromAccount(ctx context.Context, accountID string, startDate, endDate time.Time)
}

type SyncService struct {
	holdingRepository     repositories.HoldingRepository
	transactionRepository repositories.TransactionRepository

	escoClient esco.ESCOServiceClientI
}

func NewSyncService(holdingRepository repositories.HoldingRepository, transactionRepository repositories.TransactionRepository, escoClient esco.ESCOServiceClientI) *SyncService {
	return &SyncService{
		holdingRepository:     holdingRepository,
		transactionRepository: transactionRepository,
		escoClient:            escoClient,
	}
}

func (s *SyncService) SyncDataFromAccount(ctx context.Context, accountID string, startDate, endDate time.Time) {

}
