package services

import (
	"context"
	"fmt"
	"server/src/repositories"
	"server/src/schemas"
	"sort"
	"time"
)

type ReportServiceI interface {
	GenerateReport(ctx context.Context, startDate, endDate time.Time, interval time.Duration) (*schemas.AccountsReports, error)
}

type ReportService struct {
	holdingRepo       repositories.HoldingRepository
	assetRepo         repositories.AssetRepository
	assetCategoryRepo repositories.AssetCategoryRepository
	transactionRepo   repositories.TransactionRepository
}

func NewReportService(
	holdingRepo repositories.HoldingRepository,
	assetRepo repositories.AssetRepository,
	assetCategoryRepo repositories.AssetCategoryRepository,
	transactionRepo repositories.TransactionRepository,
) *ReportService {
	return &ReportService{
		holdingRepo:       holdingRepo,
		assetRepo:         assetRepo,
		assetCategoryRepo: assetCategoryRepo,
		transactionRepo:   transactionRepo,
	}
}

func (rs *ReportService) GenerateReport(ctx context.Context, startDate, endDate time.Time, interval time.Duration) (*schemas.AccountsReports, error) {
	// Get all assets with their categories
	assets, err := rs.assetRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	// Get all asset categories
	categories, err := rs.assetCategoryRepo.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	// Create a map of category IDs to category names
	categoryMap := make(map[int]string)
	for _, category := range categories {
		categoryMap[category.ID] = category.Name
	}

	// Get all holdings within the date range
	holdings, err := rs.holdingRepo.GetByDateRange(ctx, startDate, endDate)
	if err != nil {
		return nil, err
	}

	// Get all transactions within the date range
	transactions, err := rs.transactionRepo.GetByDateRange(ctx, startDate, endDate)
	if err != nil {
		return nil, err
	}

	// Group assets by category
	assetsByCategory := make(map[string][]schemas.Asset)
	categoryAssets := make(map[string]schemas.Asset)

	for _, asset := range assets {
		categoryName := categoryMap[asset.CategoryID]
		if categoryName == "" {
			continue
		}

		// Convert model.Asset to schemas.Asset
		schemaAsset := schemas.Asset{
			ID:           fmt.Sprintf("%d", asset.ID),
			Category:     categoryName,
			Type:         asset.AssetType,
			Denomination: asset.Currency,
			Holdings:     make([]schemas.Holding, 0),
		}

		// Add to assets by category
		assetsByCategory[categoryName] = append(assetsByCategory[categoryName], schemaAsset)

		// Create category asset if it doesn't exist
		if _, exists := categoryAssets[categoryName]; !exists {
			categoryAssets[categoryName] = schemas.Asset{
				ID:           categoryName,
				Category:     categoryName,
				Type:         "CATEGORY",
				Denomination: asset.Currency,
				Holdings:     make([]schemas.Holding, 0),
			}
		}
	}

	// Group holdings by asset
	holdingsByAsset := make(map[string][]schemas.Holding)
	for _, holding := range holdings {
		assetID := fmt.Sprintf("%d", holding.AssetID)
		holdingsByAsset[assetID] = append(holdingsByAsset[assetID], schemas.Holding{
			DateRequested: &holding.Date,
			Value:         holding.Value,
			Units:         holding.Units,
		})
	}

	// Group transactions by asset
	transactionsByAsset := make(map[string][]schemas.Transaction)
	for _, transaction := range transactions {
		assetID := fmt.Sprintf("%d", transaction.AssetID)
		transactionsByAsset[assetID] = append(transactionsByAsset[assetID], schemas.Transaction{
			Date:  &transaction.Date,
			Value: transaction.TotalValue,
			Units: transaction.Units,
		})
	}

	// Attach holdings and transactions to assets
	for category, assets := range assetsByCategory {
		for i := range assets {
			assets[i].Holdings = holdingsByAsset[assets[i].ID]
			assets[i].Transactions = transactionsByAsset[assets[i].ID]
		}
		assetsByCategory[category] = assets
	}

	// Calculate total holdings by date
	totalHoldingsByDate := make(map[string]schemas.Holding)
	for _, holding := range holdings {
		dateStr := holding.Date.Format("2006-01-02")
		totalHoldingsByDate[dateStr] = schemas.Holding{
			DateRequested: &holding.Date,
			Value:         holding.Value,
			Units:         holding.Units,
		}
	}

	// Convert transactions to schema format
	schemaTransactions := make(map[string]schemas.Transaction)
	for _, transaction := range transactions {
		dateStr := transaction.Date.Format("2006-01-02")
		schemaTransactions[dateStr] = schemas.Transaction{
			Date:  &transaction.Date,
			Value: transaction.TotalValue,
			Units: transaction.Units,
		}
	}

	// Create account state by category
	accountStateByCategory := &schemas.AccountStateByCategory{
		AssetsByCategory:        &assetsByCategory,
		CategoryAssets:          &categoryAssets,
		TotalHoldingsByDate:     &totalHoldingsByDate,
		TotalTransactionsByDate: &schemaTransactions,
	}

	// Generate the report using the report generator
	report, err := rs.generateAccountReports(accountStateByCategory, startDate, endDate, interval)
	if err != nil {
		return nil, err
	}

	return report, nil
}

// generateAccountReports calculates the return for each asset per category and returns an AccountsReports struct.
func (rs *ReportService) generateAccountReports(
	accountStateByCategory *schemas.AccountStateByCategory,
	startDate, endDate time.Time,
	interval time.Duration) (*schemas.AccountsReports, error) {
	assetReturnsByCategory := make(map[string][]schemas.AssetReturn)
	categoryAssetReturns := make(map[string]schemas.AssetReturn)

	// Iterate through each category and its associated assets
	for category, assets := range *accountStateByCategory.AssetsByCategory {
		if category == "ARS" {
			continue
		}
		for _, asset := range assets {
			assetReturn, _ := rs.CalculateAssetReturn(asset, interval)
			assetReturnsByCategory[category] = append(assetReturnsByCategory[category], assetReturn)
		}
	}
	// Iterate through each category assets
	for category, asset := range *accountStateByCategory.CategoryAssets {
		if category == "ARS" {
			continue
		}
		assetReturn, _ := rs.CalculateAssetReturn(asset, interval)
		categoryAssetReturns[category] = assetReturn
	}

	totalHoldingsByDate := make([]schemas.Holding, 0, len(*accountStateByCategory.TotalHoldingsByDate))
	for _, holding := range *accountStateByCategory.TotalHoldingsByDate {
		totalHoldingsByDate = append(totalHoldingsByDate, holding)
	}
	totalTransactionsByDate := make([]schemas.Transaction, 0, len(*accountStateByCategory.TotalTransactionsByDate))
	for _, transaction := range *accountStateByCategory.TotalTransactionsByDate {
		totalTransactionsByDate = append(totalTransactionsByDate, transaction)
	}

	totalReturns := rs.CalculateHoldingsReturn(totalHoldingsByDate, totalTransactionsByDate, interval, true)
	finalIntervalReturn := rs.CalculateFinalIntervalReturn(totalReturns)
	filteredAssets := rs.FilterAssetsByCategoryHoldingsByInterval(accountStateByCategory.AssetsByCategory, startDate, endDate, interval)
	filteredCategoryAssets := rs.FilterAssetsHoldingsByInterval(accountStateByCategory.CategoryAssets, startDate, endDate, interval)
	filteredTotalHoldings := rs.FilterHoldingsByInterval(totalHoldingsByDate, startDate, endDate, interval)
	return &schemas.AccountsReports{
		AssetsByCategory:        &filteredAssets,
		AssetsReturnByCategory:  &assetReturnsByCategory,
		CategoryAssets:          &filteredCategoryAssets,
		CategoryAssetsReturn:    &categoryAssetReturns,
		TotalHoldingsByDate:     filteredTotalHoldings,
		TotalTransactionsByDate: totalTransactionsByDate,
		TotalReturns:            totalReturns,
		FinalIntervalReturn:     finalIntervalReturn,
	}, nil
}

// CalculateAssetReturn calculates the return for a single asset by taking holdings in pairs and applying transactions within the date ranges.
func (rs *ReportService) CalculateAssetReturn(asset schemas.Asset, interval time.Duration) (schemas.AssetReturn, error) {
	if len(asset.Holdings) < 2 {
		return schemas.AssetReturn{}, fmt.Errorf("insufficient holdings data to calculate return for asset %s", asset.ID)
	}

	returnsByInterval := rs.CalculateHoldingsReturn(asset.Holdings, asset.Transactions, interval, false)

	// Return the result
	return schemas.AssetReturn{
		ID:                 asset.ID,
		Type:               asset.Type,
		Denomination:       asset.Denomination,
		Category:           asset.Category,
		ReturnsByDateRange: returnsByInterval,
	}, nil
}

func (rs *ReportService) CalculateFinalIntervalReturn(totalReturns []schemas.ReturnByDate) float64 {
	intervalReturn := 1.0
	for _, totalReturn := range totalReturns {
		intervalReturn *= 1 + (totalReturn.ReturnPercentage / 100)
	}
	return intervalReturn
}

func (rs *ReportService) CalculateHoldingsReturn(holdings []schemas.Holding, transactions []schemas.Transaction, interval time.Duration, multiAsset bool) []schemas.ReturnByDate {
	// Sort holdings by date
	sortedHoldings := rs.sortHoldingsByDate(holdings)
	var dailyReturns []schemas.ReturnByDate

	// Iterate through each pair of consecutive holdings
	for i := 0; i < len(sortedHoldings)-1; i++ {
		startingHolding := sortedHoldings[i]
		endingHolding := sortedHoldings[i+1]

		startDate := *startingHolding.DateRequested
		endDate := *endingHolding.DateRequested
		if endDate.Sub(startDate) > 24*time.Hour {
			continue
		}
		startingValue := startingHolding.Value
		endingValue := endingHolding.Value

		// Calculate the net transactions within the date range
		var netStartDateTransactions, netEndDateTransactions float64

		for _, transaction := range transactions {
			if transaction.Date == nil {
				continue
			}
			if transaction.Date.Equal(startDate) {
				if !multiAsset && startingHolding.Units != 0 {
					startingValuePerUnit := startingHolding.Value / startingHolding.Units
					transaction.Value = transaction.Units * startingValuePerUnit
				}
				netStartDateTransactions -= transaction.Value
			} else if transaction.Date.Equal(endDate) {
				if !multiAsset && endingHolding.Units != 0 {
					endingValuePerUnit := endingHolding.Value / endingHolding.Units
					transaction.Value = transaction.Units * endingValuePerUnit
				}
				netEndDateTransactions -= transaction.Value
			}
		}
		// netStartValue := startingValue + netStartDateTransactions
		netStartValue := startingValue
		netEndValue := endingValue + netEndDateTransactions
		// Calculate return for this date range
		if startingValue < 1 && startingValue > -1 {
			continue
		}

		returnPercentage := ((netEndValue - netStartValue) / netStartValue) * 100
		// Append the return for this date range
		dailyReturns = append(dailyReturns, schemas.ReturnByDate{
			StartDate:        startDate,
			EndDate:          endDate,
			ReturnPercentage: returnPercentage,
		})
	}
	// Collapse daily returns into intervals
	return rs.collapseReturnsByInterval(dailyReturns, interval)
}

func (rs *ReportService) collapseReturnsByInterval(dailyReturns []schemas.ReturnByDate, interval time.Duration) []schemas.ReturnByDate {
	var returnsByInterval []schemas.ReturnByDate
	var (
		currentIntervalStart time.Time
		currentIntervalEnd   time.Time
		compoundReturn       float64 = 1.0
	)

	for _, dailyReturn := range dailyReturns {
		if currentIntervalStart.IsZero() {
			currentIntervalStart = dailyReturn.StartDate
			currentIntervalEnd = currentIntervalStart.Add(interval)
		}

		if dailyReturn.EndDate.Before(currentIntervalEnd) {
			// Apply compound calculation
			compoundReturn *= 1 + (dailyReturn.ReturnPercentage / 100)
		} else {
			if interval == 24*time.Hour {
				compoundReturn *= 1 + (dailyReturn.ReturnPercentage / 100)
			}
			// Close the current interval
			returnsByInterval = append(returnsByInterval, schemas.ReturnByDate{
				StartDate:        currentIntervalStart,
				EndDate:          currentIntervalEnd,
				ReturnPercentage: (compoundReturn - 1) * 100,
			})

			// Reset for the new interval
			currentIntervalStart = currentIntervalEnd
			currentIntervalEnd = currentIntervalStart.Add(interval)
			// compoundReturn = 1 + (dailyReturn.ReturnPercentage / 100)
		}
	}

	// Append the last interval
	if !currentIntervalStart.IsZero() {
		returnsByInterval = append(returnsByInterval, schemas.ReturnByDate{
			StartDate:        currentIntervalStart,
			EndDate:          currentIntervalEnd,
			ReturnPercentage: (compoundReturn - 1) * 100,
		})
	}

	return returnsByInterval
}

func (rs *ReportService) FilterAssetsByCategoryHoldingsByInterval(assetsByCategory *map[string][]schemas.Asset, startDate, endDate time.Time, interval time.Duration) map[string][]schemas.Asset {
	filteredAssetsByCategory := make(map[string][]schemas.Asset)

	for category, assets := range *assetsByCategory {
		for _, asset := range assets {
			filteredHoldings := rs.FilterHoldingsByInterval(asset.Holdings, startDate, endDate, interval)

			if len(filteredHoldings) > 0 {
				asset.Holdings = filteredHoldings
				filteredAssetsByCategory[category] = append(filteredAssetsByCategory[category], asset)
			}
		}
	}

	return filteredAssetsByCategory
}

func (rs *ReportService) FilterAssetsHoldingsByInterval(categoryAssets *map[string]schemas.Asset, startDate, endDate time.Time, interval time.Duration) map[string]schemas.Asset {
	filteredAssetsByCategory := *categoryAssets

	for _, asset := range filteredAssetsByCategory {
		filteredHoldings := rs.FilterHoldingsByInterval(asset.Holdings, startDate, endDate, interval)

		if len(filteredHoldings) > 0 {
			asset.Holdings = filteredHoldings
		}
	}

	return filteredAssetsByCategory
}

func (rs *ReportService) FilterHoldingsByInterval(holdings []schemas.Holding, startDate, endDate time.Time, interval time.Duration) []schemas.Holding {
	filteredHoldings := []schemas.Holding{}

	// Generate interval boundaries
	for date := startDate; !date.After(endDate); date = date.Add(interval) {
		for _, holding := range holdings {
			// Include holdings that fall within the exact interval
			if date == *holding.DateRequested {
				filteredHoldings = append(filteredHoldings, holding)
			}
		}
	}

	return filteredHoldings
}

// sortHoldingsByDate sorts the holdings by DateRequested.
func (rs *ReportService) sortHoldingsByDate(holdings []schemas.Holding) []schemas.Holding {
	sort.Slice(holdings, func(i, j int) bool {
		return holdings[i].DateRequested.Before(*holdings[j].DateRequested)
	})
	return holdings
}
