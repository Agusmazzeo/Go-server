package services

import (
	"context"
	"server/src/models"
	"server/src/repositories"
	"server/src/schemas"
	"sort"
	"time"
)

type AccountServiceI interface {
	GetAccountState(ctx context.Context, clientID string, date time.Time) (*schemas.AccountState, error)
	GetMultiAccountStateWithTransactions(ctx context.Context, clientIDs []string, startDate, endDate time.Time, interval time.Duration) ([]*schemas.AccountState, error)
	GetMultiAccountStateByCategory(ctx context.Context, clientIDs []string, startDate, endDate time.Time, interval time.Duration) (*schemas.AccountStateByCategory, error)
}

type AccountService struct {
	holdingRepo     repositories.HoldingRepository
	transactionRepo repositories.TransactionRepository
	assetRepo       repositories.AssetRepository
}

func NewAccountService(
	holdingRepo repositories.HoldingRepository,
	transactionRepo repositories.TransactionRepository,
	assetRepo repositories.AssetRepository,
) *AccountService {
	return &AccountService{
		holdingRepo:     holdingRepo,
		transactionRepo: transactionRepo,
		assetRepo:       assetRepo,
	}
}

// GetAccountState returns the account state for a specific date
func (s *AccountService) GetAccountState(ctx context.Context, clientID string, date time.Time) (*schemas.AccountState, error) {
	// Get holdings for the specific date
	holdings, err := s.holdingRepo.GetByClientID(ctx, clientID, date, date)
	if err != nil {
		return nil, err
	}

	// Get transactions for the specific date
	transactions, err := s.transactionRepo.GetByClientID(ctx, clientID, date, date)
	if err != nil {
		return nil, err
	}

	return s.buildAccountState(ctx, holdings, transactions)
}

// GetMultiAccountStateWithTransactions returns account states for multiple clients
func (s *AccountService) GetMultiAccountStateWithTransactions(ctx context.Context, clientIDs []string, startDate, endDate time.Time, interval time.Duration) ([]*schemas.AccountState, error) {
	// Get all holdings for the client IDs
	holdings, err := s.holdingRepo.GetByClientIDs(ctx, clientIDs, startDate, endDate)
	if err != nil {
		return nil, err
	}

	// Get all transactions for the client IDs
	transactions, err := s.transactionRepo.GetByClientIDs(ctx, clientIDs, startDate, endDate)
	if err != nil {
		return nil, err
	}

	// Group by client ID
	holdingsByClient := s.groupHoldingsByClient(holdings)
	transactionsByClient := s.groupTransactionsByClient(transactions)

	// Build account states for each client
	accountStates := make([]*schemas.AccountState, 0, len(clientIDs))
	for _, clientID := range clientIDs {
		clientHoldings := holdingsByClient[clientID]
		clientTransactions := transactionsByClient[clientID]

		accountState, err := s.buildAccountState(ctx, clientHoldings, clientTransactions)
		if err != nil {
			return nil, err
		}
		accountStates = append(accountStates, accountState)
	}

	return accountStates, nil
}

// GetMultiAccountStateByCategory returns account states grouped by category for multiple clients
func (s *AccountService) GetMultiAccountStateByCategory(ctx context.Context, clientIDs []string, startDate, endDate time.Time, interval time.Duration) (*schemas.AccountStateByCategory, error) {
	// Get grouped data from database
	categoryHoldings, err := s.holdingRepo.GetGroupedByCategoryAndDate(ctx, clientIDs, startDate, endDate)
	if err != nil {
		return nil, err
	}

	categoryTransactions, err := s.transactionRepo.GetGroupedByCategoryAndDate(ctx, clientIDs, startDate, endDate)
	if err != nil {
		return nil, err
	}

	totalHoldings, err := s.holdingRepo.GetTotalByDate(ctx, clientIDs, startDate, endDate)
	if err != nil {
		return nil, err
	}

	totalTransactions, err := s.transactionRepo.GetTotalByDate(ctx, clientIDs, startDate, endDate)
	if err != nil {
		return nil, err
	}

	// Get all assets with categories for building asset lists
	assetsWithCategories, err := s.assetRepo.GetWithCategories(ctx)
	if err != nil {
		return nil, err
	}

	// Get individual account states for asset details
	accountStates, err := s.GetMultiAccountStateWithTransactions(ctx, clientIDs, startDate, endDate, interval)
	if err != nil {
		return nil, err
	}

	return s.buildAccountStateByCategory(
		accountStates,
		categoryHoldings,
		categoryTransactions,
		totalHoldings,
		totalTransactions,
		assetsWithCategories,
	), nil
}

// buildAccountState builds an AccountState from holdings and transactions data
func (s *AccountService) buildAccountState(ctx context.Context, holdings []models.Holding, transactions []models.Transaction) (*schemas.AccountState, error) {
	assets := make(map[string]schemas.Asset)
	assetMap := make(map[int]*models.Asset)

	// Get unique asset IDs
	assetIDs := make(map[int]bool)
	for _, holding := range holdings {
		assetIDs[holding.AssetID] = true
	}
	for _, transaction := range transactions {
		assetIDs[transaction.AssetID] = true
	}

	// Get all assets in one query
	assetIDList := make([]int, 0, len(assetIDs))
	for id := range assetIDs {
		assetIDList = append(assetIDList, id)
	}

	assetsList, err := s.assetRepo.GetByIDs(ctx, assetIDList)
	if err != nil {
		return nil, err
	}

	// Build asset map
	for i := range assetsList {
		assetMap[assetsList[i].ID] = &assetsList[i]
	}

	// Process holdings
	for _, holding := range holdings {
		asset, exists := assetMap[holding.AssetID]
		if !exists {
			continue // Skip if asset not found
		}

		assetKey := asset.ExternalID
		if _, exists := assets[assetKey]; !exists {
			assets[assetKey] = schemas.Asset{
				ID:           asset.ExternalID,
				Type:         asset.AssetType,
				Denomination: asset.Name,
				Category:     "", // Will be populated if we have category info
				Holdings:     []schemas.Holding{},
				Transactions: []schemas.Transaction{},
			}
		}

		assetState := assets[assetKey]
		assetState.Holdings = append(assetState.Holdings, schemas.Holding{
			Currency:      asset.Currency,
			CurrencySign:  getCurrencySign(asset.Currency),
			Value:         holding.Value,
			Units:         holding.Units,
			DateRequested: &holding.Date,
			Date:          &holding.Date,
		})
		assets[assetKey] = assetState
	}

	// Process transactions
	for _, transaction := range transactions {
		asset, exists := assetMap[transaction.AssetID]
		if !exists {
			continue // Skip if asset not found
		}

		assetKey := asset.ExternalID
		if _, exists := assets[assetKey]; !exists {
			assets[assetKey] = schemas.Asset{
				ID:           asset.ExternalID,
				Type:         asset.AssetType,
				Denomination: asset.Name,
				Category:     "", // Will be populated if we have category info
				Holdings:     []schemas.Holding{},
				Transactions: []schemas.Transaction{},
			}
		}

		assetState := assets[assetKey]
		assetState.Transactions = append(assetState.Transactions, schemas.Transaction{
			Currency:     asset.Currency,
			CurrencySign: getCurrencySign(asset.Currency),
			Value:        transaction.TotalValue,
			Units:        transaction.Units,
			Date:         &transaction.Date,
		})
		assets[assetKey] = assetState
	}

	// Sort holdings and transactions for each asset
	for assetKey, asset := range assets {
		sort.Slice(asset.Holdings, func(i, j int) bool {
			return asset.Holdings[i].DateRequested.Before(*asset.Holdings[j].DateRequested)
		})
		sort.Slice(asset.Transactions, func(i, j int) bool {
			return asset.Transactions[i].Date.Before(*asset.Transactions[j].Date)
		})
		assets[assetKey] = asset
	}

	return &schemas.AccountState{Assets: &assets}, nil
}

// groupHoldingsByClient groups holdings by client ID
func (s *AccountService) groupHoldingsByClient(holdings []models.Holding) map[string][]models.Holding {
	result := make(map[string][]models.Holding)
	for _, holding := range holdings {
		result[holding.ClientID] = append(result[holding.ClientID], holding)
	}
	return result
}

// groupTransactionsByClient groups transactions by client ID
func (s *AccountService) groupTransactionsByClient(transactions []models.Transaction) map[string][]models.Transaction {
	result := make(map[string][]models.Transaction)
	for _, transaction := range transactions {
		result[transaction.ClientID] = append(result[transaction.ClientID], transaction)
	}
	return result
}

// buildAccountStateByCategory builds AccountStateByCategory from grouped data
func (s *AccountService) buildAccountStateByCategory(
	accountStates []*schemas.AccountState,
	categoryHoldings map[string]map[string]float64,
	categoryTransactions map[string]map[string]float64,
	totalHoldings map[string]float64,
	totalTransactions map[string]float64,
	assetsWithCategories []models.AssetWithCategory,
) *schemas.AccountStateByCategory {

	// Build assets by category from individual account states
	assetsByCategory := make(map[string][]schemas.Asset)

	// Create asset map with category information
	assetCategoryMap := make(map[string]string)
	for _, asset := range assetsWithCategories {
		assetCategoryMap[asset.ExternalID] = asset.CategoryName
	}

	// Group assets by category from account states
	for _, accountState := range accountStates {
		if accountState.Assets == nil {
			continue
		}
		for _, asset := range *accountState.Assets {
			category := assetCategoryMap[asset.ID]
			if category == "" {
				category = "S / C" // Default category
			}
			// Update asset category
			asset.Category = category
			assetsByCategory[category] = append(assetsByCategory[category], asset)
		}
	}

	// Sort assets within each category
	for category := range assetsByCategory {
		sort.Slice(assetsByCategory[category], func(i, j int) bool {
			return assetsByCategory[category][i].ID < assetsByCategory[category][j].ID
		})
	}

	// Convert grouped data to schema format
	totalHoldingsByDate := make(map[string]schemas.Holding)
	for dateStr, value := range totalHoldings {
		date, _ := time.Parse("2006-01-02", dateStr)
		totalHoldingsByDate[dateStr] = schemas.Holding{
			Currency:      "Pesos",
			CurrencySign:  "$",
			Value:         value,
			DateRequested: &date,
			Date:          &date,
		}
	}

	totalTransactionsByDate := make(map[string]schemas.Transaction)
	for dateStr, value := range totalTransactions {
		date, _ := time.Parse("2006-01-02", dateStr)
		totalTransactionsByDate[dateStr] = schemas.Transaction{
			Currency:     "Pesos",
			CurrencySign: "$",
			Value:        value,
			Date:         &date,
		}
	}

	// Build category assets
	categoryAssets := make(map[string]schemas.Asset)
	for category, holdingsByDate := range categoryHoldings {
		categoryAssets[category] = schemas.Asset{
			ID:           category,
			Type:         "Category",
			Denomination: category,
			Category:     category,
			Holdings:     []schemas.Holding{},
			Transactions: []schemas.Transaction{},
		}

		// Add holdings
		for dateStr, value := range holdingsByDate {
			date, _ := time.Parse("2006-01-02", dateStr)
			asset := categoryAssets[category]
			asset.Holdings = append(asset.Holdings, schemas.Holding{
				Currency:      "Pesos",
				CurrencySign:  "$",
				Value:         value,
				DateRequested: &date,
				Date:          &date,
			})
			categoryAssets[category] = asset
		}
	}

	// Add transactions to category assets
	for category, transactionsByDate := range categoryTransactions {
		if _, exists := categoryAssets[category]; !exists {
			categoryAssets[category] = schemas.Asset{
				ID:           category,
				Type:         "Category",
				Denomination: category,
				Category:     category,
				Holdings:     []schemas.Holding{},
				Transactions: []schemas.Transaction{},
			}
		}

		// Add transactions
		for dateStr, value := range transactionsByDate {
			date, _ := time.Parse("2006-01-02", dateStr)
			asset := categoryAssets[category]
			asset.Transactions = append(asset.Transactions, schemas.Transaction{
				Currency:     "Pesos",
				CurrencySign: "$",
				Value:        value,
				Date:         &date,
			})
			categoryAssets[category] = asset
		}
	}

	return &schemas.AccountStateByCategory{
		AssetsByCategory:        &assetsByCategory,
		CategoryAssets:          &categoryAssets,
		TotalHoldingsByDate:     &totalHoldingsByDate,
		TotalTransactionsByDate: &totalTransactionsByDate,
	}
}

// getCurrencySign returns the currency sign for a given currency
func getCurrencySign(currency string) string {
	switch currency {
	case "USD":
		return "$"
	case "EUR":
		return "€"
	case "ARS":
		return "$"
	default:
		return "$"
	}
}
