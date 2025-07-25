package services

import (
	"context"
	"fmt"
	"server/src/schemas"
	"server/src/utils"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-gota/gota/dataframe"
	"github.com/go-gota/gota/series"
	"github.com/xuri/excelize/v2"
)

type ReportServiceI interface {
	GenerateReport(ctx context.Context, accountStateByCategory *schemas.AccountStateByCategory, startDate, endDate time.Time, interval time.Duration) (*schemas.AccountsReports, error)
	GenerateReportDataframes(ctx context.Context, accountsReport *schemas.AccountsReports, startDate, endDate time.Time, interval time.Duration) (*schemas.ReportDataframes, error)
	GenerateXLSXReport(ctx context.Context, dataframesAndCharts *schemas.ReportDataframes) (*excelize.File, error)
}

type ReportService struct{}

func NewReportService() *ReportService {
	return &ReportService{}
}

// generateAccountReports calculates the return for each asset per category and returns an AccountsReports struct.
func (rs *ReportService) GenerateReport(
	ctx context.Context,
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

	// Calculate total returns using weighted asset returns instead of total holdings/transactions
	totalReturns := rs.CalculateWeightedTotalReturns(assetReturnsByCategory, *accountStateByCategory.AssetsByCategory, totalHoldingsByDate, interval)
	finalIntervalReturn := rs.CalculateFinalIntervalReturn(totalReturns)
	filteredAssets := rs.FilterAssetsByCategoryHoldingsByInterval(accountStateByCategory.AssetsByCategory, startDate, endDate, interval)
	filteredCategoryAssets := rs.FilterAssetsHoldingsByInterval(accountStateByCategory.CategoryAssets, startDate, endDate, interval)
	filteredTotalHoldings := rs.FilterHoldingsByInterval(totalHoldingsByDate, startDate, endDate, interval)
	return &schemas.AccountsReports{
		AssetsByCategory:       &filteredAssets,
		AssetsReturnByCategory: &assetReturnsByCategory,
		CategoryAssets:         &filteredCategoryAssets,
		CategoryAssetsReturn:   &categoryAssetReturns,
		TotalHoldingsByDate:    filteredTotalHoldings,
		TotalReturns:           totalReturns,
		FinalIntervalReturn:    finalIntervalReturn,
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

	// If no holdings, return empty slice
	if len(sortedHoldings) == 0 {
		return dailyReturns
	}

	// Find the overall date range
	startDate := *sortedHoldings[0].DateRequested
	endDate := *sortedHoldings[len(sortedHoldings)-1].DateRequested

	// Create a map of holdings by date for easy lookup
	holdingsByDate := make(map[string]schemas.Holding)
	for _, holding := range sortedHoldings {
		dateStr := holding.DateRequested.Format("2006-01-02")
		holdingsByDate[dateStr] = holding
	}

	// Generate returns for each day in the range, filling missing dates with zero holdings
	dailyInterval := 24 * time.Hour
	for currentDate := startDate; !currentDate.After(endDate); currentDate = currentDate.Add(dailyInterval) {
		nextDate := currentDate.Add(dailyInterval)
		dateStr := currentDate.Format("2006-01-02")
		nextDateStr := nextDate.Format("2006-01-02")

		// Get holdings for current and next date, or create zero holdings if missing
		var startingHolding, endingHolding schemas.Holding

		if holding, exists := holdingsByDate[dateStr]; exists {
			startingHolding = holding
		} else {
			// Create zero holding for missing date
			startingHolding = schemas.Holding{}
		}

		if holding, exists := holdingsByDate[nextDateStr]; exists {
			endingHolding = holding
		} else {
			// Create zero holding for missing date
			endingHolding = schemas.Holding{}
		}

		startingValue := startingHolding.Value
		endingValue := endingHolding.Value

		// Calculate the net transactions within the date range
		var netEndDateTransactions float64

		for _, transaction := range transactions {
			if transaction.Date == nil {
				continue
			}

			if !rs.isSameDate(*transaction.Date, nextDate) {
				continue
			}

			var transactionValue float64
			if transaction.Value != 0 {
				// Use the transaction value directly if it's not zero
				transactionValue = transaction.Value
			} else if !multiAsset && endingHolding.Units != 0 && transaction.Units != 0 {
				// Calculate value from units using the end holding's value per unit
				endingValuePerUnit := endingHolding.Value / endingHolding.Units
				transactionValue = transaction.Units * endingValuePerUnit
			} else {
				// For zero-value transactions without units, use a small default value
				transactionValue = 0.0
			}
			netEndDateTransactions -= transactionValue
		}

		// Calculate return for this date range
		// The correct formula is: (End value - Start value - Net transactions) / Start value
		// This gives us the market return excluding the impact of transactions
		var returnPercentage, netEndValue float64
		netEndValue = endingValue + netEndDateTransactions
		if startingValue < 1 && startingValue > -1 || (netEndValue < 1 && netEndValue > -1) {
			// If start value is too small, treat as 0% return
			returnPercentage = 0.0
		} else {
			// Calculate market return: (End value - Start value - Net transactions) / Start value
			returnPercentage = ((netEndValue - startingValue) / startingValue) * 100

		}

		// Append the return for this date range
		dailyReturns = append(dailyReturns, schemas.ReturnByDate{
			StartDate:        currentDate,
			EndDate:          nextDate,
			ReturnPercentage: returnPercentage,
		})
	}

	// Collapse daily returns into intervals
	return rs.CollapseReturnsByInterval(dailyReturns, interval)
}

// CalculateWeightedTotalReturns calculates total portfolio returns by weighting individual asset returns
// by their percentage of the total portfolio value, properly handling intervals
func (rs *ReportService) CalculateWeightedTotalReturns(assetReturnsByCategory map[string][]schemas.AssetReturn, assetsByCategory map[string][]schemas.Asset, totalHoldingsByDate []schemas.Holding, interval time.Duration) []schemas.ReturnByDate {
	// Create a map of total holdings by date for easy lookup
	totalHoldingsByDateMap := make(map[string]float64)
	for _, holding := range totalHoldingsByDate {
		if holding.DateRequested != nil {
			dateStr := holding.DateRequested.Format("2006-01-02")
			totalHoldingsByDateMap[dateStr] = holding.Value
		}
	}

	// Create a map of assets by ID for easy lookup
	assetsByID := make(map[string]schemas.Asset)
	for _, assets := range assetsByCategory {
		for _, asset := range assets {
			assetsByID[asset.ID] = asset
		}
	}

	// Create a map to store weighted returns by interval end date
	weightedReturnsByInterval := make(map[string]float64)
	intervalWeights := make(map[string]float64)

	// Iterate through each category and its asset returns
	for _, assetReturns := range assetReturnsByCategory {
		for _, assetReturn := range assetReturns {
			// Find the corresponding asset data
			asset, exists := assetsByID[assetReturn.ID]
			if !exists {
				continue // Skip if asset not found
			}

			// For each asset return period, calculate its contribution to total returns
			for _, returnPeriod := range assetReturn.ReturnsByDateRange {
				// Get the asset's value at the start of this return period
				assetStartValue := rs.getAssetValueAtDate(asset, returnPeriod.StartDate)

				// Get total portfolio value at the start date
				startDateStr := returnPeriod.StartDate.Format("2006-01-02")
				totalPortfolioValue := totalHoldingsByDateMap[startDateStr]

				// Calculate the asset's weight in the portfolio
				var assetWeight float64
				if totalPortfolioValue > 0 {
					assetWeight = assetStartValue / totalPortfolioValue
				} else {
					assetWeight = 0
				}

				// Calculate weighted return contribution
				weightedReturn := returnPeriod.ReturnPercentage * assetWeight

				// Use the end date as the key for this interval
				endDateStr := returnPeriod.EndDate.Format("2006-01-02")
				weightedReturnsByInterval[endDateStr] += weightedReturn
				intervalWeights[endDateStr] += assetWeight
			}
		}
	}

	// Convert the map to a slice of ReturnByDate and normalize by weights
	var totalReturns []schemas.ReturnByDate
	for endDateStr, weightedReturn := range weightedReturnsByInterval {
		endDate, err := time.Parse("2006-01-02", endDateStr)
		if err != nil {
			continue // Skip invalid dates
		}

		// Calculate the start date based on the interval
		startDate := endDate.Add(-interval)

		// Normalize by total weight for this interval
		var finalReturn float64
		if weight := intervalWeights[endDateStr]; weight > 0 {
			finalReturn = weightedReturn / weight
		} else {
			finalReturn = weightedReturn
		}

		totalReturns = append(totalReturns, schemas.ReturnByDate{
			StartDate:        startDate,
			EndDate:          endDate,
			ReturnPercentage: finalReturn,
		})
	}

	// Sort by start date
	sort.Slice(totalReturns, func(i, j int) bool {
		return totalReturns[i].StartDate.Before(totalReturns[j].StartDate)
	})

	return totalReturns
}

// getAssetValueAtDate returns the asset's value at a specific date
func (rs *ReportService) getAssetValueAtDate(asset schemas.Asset, date time.Time) float64 {
	// Find the holding for the given date
	for _, holding := range asset.Holdings {
		if holding.DateRequested != nil && rs.isSameDate(*holding.DateRequested, date) {
			return holding.Value
		}
	}

	// If no exact match found, find the closest previous holding
	var closestHolding *schemas.Holding
	for _, holding := range asset.Holdings {
		if holding.DateRequested != nil && (*holding.DateRequested).Before(date) || rs.isSameDate(*holding.DateRequested, date) {
			if closestHolding == nil || (*holding.DateRequested).After(*closestHolding.DateRequested) {
				closestHolding = &holding
			}
		}
	}

	if closestHolding != nil {
		return closestHolding.Value
	}

	return 0.0
}

func (rs *ReportService) CollapseReturnsByInterval(dailyReturns []schemas.ReturnByDate, interval time.Duration) []schemas.ReturnByDate {
	if len(dailyReturns) == 0 {
		return []schemas.ReturnByDate{}
	}

	// Sort daily returns by start date to ensure proper order
	sortedReturns := rs.sortReturnsByDate(dailyReturns)

	var returnsByInterval []schemas.ReturnByDate
	var (
		currentIntervalStart time.Time
		currentIntervalEnd   time.Time
		compoundReturn       float64 = 1.0
	)

	// Initialize the first interval
	currentIntervalStart = sortedReturns[0].StartDate
	currentIntervalEnd = currentIntervalStart.Add(interval)

	for _, dailyReturn := range sortedReturns {
		// Check if this daily return belongs to the current interval
		if dailyReturn.StartDate.Before(currentIntervalEnd) {
			// This daily return belongs to the current interval
			compoundReturn *= 1 + (dailyReturn.ReturnPercentage / 100)
		} else {
			// This daily return belongs to a new interval
			// First, close the current interval
			intervalReturnPercentage := (compoundReturn - 1) * 100

			returnsByInterval = append(returnsByInterval, schemas.ReturnByDate{
				StartDate:        currentIntervalStart,
				EndDate:          currentIntervalEnd,
				ReturnPercentage: intervalReturnPercentage,
			})

			// Start a new interval
			currentIntervalStart = currentIntervalEnd
			currentIntervalEnd = currentIntervalStart.Add(interval)
			compoundReturn = 1.0 + (dailyReturn.ReturnPercentage / 100)
		}
	}

	// Append the last interval
	intervalReturnPercentage := (compoundReturn - 1) * 100

	returnsByInterval = append(returnsByInterval, schemas.ReturnByDate{
		StartDate:        currentIntervalStart,
		EndDate:          currentIntervalEnd,
		ReturnPercentage: intervalReturnPercentage,
	})

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
			// Compare only date part (day, month, year) without time
			if holding.DateRequested != nil {
				holdingDate := *holding.DateRequested
				if rs.isSameDate(date, holdingDate) {
					filteredHoldings = append(filteredHoldings, holding)
				}
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

// sortReturnsByDate sorts the returns by StartDate.
func (rs *ReportService) sortReturnsByDate(returns []schemas.ReturnByDate) []schemas.ReturnByDate {
	sort.Slice(returns, func(i, j int) bool {
		return returns[i].StartDate.Before(returns[j].StartDate)
	})
	return returns
}

func (rs *ReportService) GenerateReportDataframes(ctx context.Context, accountsReport *schemas.AccountsReports, startDate, endDate time.Time, interval time.Duration) (*schemas.ReportDataframes, error) {
	var reportDf *dataframe.DataFrame
	var reportPercentageDf *dataframe.DataFrame
	var returnsDf *dataframe.DataFrame
	var referenceVariablesDf *dataframe.DataFrame
	var categoryDf *dataframe.DataFrame
	var categoryPercentageDf *dataframe.DataFrame

	var wg sync.WaitGroup
	wg.Add(4)
	var errChan = make(chan error, 4)

	go func() {
		defer wg.Done()
		var err error
		reportDf, err = rs.parseAccountsReportToDataFrame(ctx, accountsReport, startDate, endDate, interval)
		if err != nil {
			errChan <- err
			return
		}
		if reportDf != nil {
			reportPercentageDf = rs.divideByTotal(reportDf)
		}
	}()

	go func() {
		defer wg.Done()
		var err error
		categoryDf, err = rs.parseAccountsCategoryToDataFrame(ctx, accountsReport, startDate, endDate, interval)
		if err != nil {
			errChan <- err
			return
		}
		if categoryDf != nil {
			categoryPercentageDf = rs.divideByTotal(categoryDf)
		}
	}()

	go func() {
		defer wg.Done()
		var err error
		returnsDf, err = rs.parseAccountsReturnToDataFrame(ctx, accountsReport, startDate, endDate, interval)
		if err != nil {
			errChan <- err
			return
		}
	}()

	go func() {
		defer wg.Done()
		var err error
		referenceVariablesDf, err = rs.parseReferenceVariablesToDataFrame(ctx, accountsReport, startDate, endDate, interval)
		if err != nil {
			errChan <- err
			return
		}
	}()

	wg.Wait()
	close(errChan)
	if err := <-errChan; err != nil {
		return nil, err
	}

	// Order dataframes columns by defined order
	firstColumns := []string{
		"DateRequested",
	}
	finalColumns := []string{
		"TOTAL",
	}
	reportDf = utils.SortDataFrameColumns(reportDf, firstColumns, finalColumns)
	returnsDf = utils.SortDataFrameColumns(returnsDf, firstColumns, finalColumns)
	reportPercentageDf = utils.SortDataFrameColumns(reportPercentageDf, firstColumns, finalColumns)
	referenceVariablesDf = utils.SortDataFrameColumns(referenceVariablesDf, firstColumns, finalColumns)
	categoryDf = utils.SortDataFrameColumns(categoryDf, firstColumns, finalColumns)
	categoryPercentageDf = utils.SortDataFrameColumns(categoryPercentageDf, firstColumns, finalColumns)

	return &schemas.ReportDataframes{
		ReportDF:             reportDf,
		ReportPercentageDf:   reportPercentageDf,
		ReturnDF:             returnsDf,
		ReferenceVariablesDF: referenceVariablesDf,
		CategoryDF:           categoryDf,
		CategoryPercentageDF: categoryPercentageDf,
	}, nil
}

func (rs *ReportService) GenerateXLSXReport(ctx context.Context, dataframesAndCharts *schemas.ReportDataframes) (*excelize.File, error) {
	file, err := rs.convertReportDataframeToExcel(nil, dataframesAndCharts.ReportDF, "Tenencia", false, true, true)
	if err != nil {
		return nil, err
	}

	if dataframesAndCharts.ReportPercentageDf != nil {
		file, err = rs.convertReportDataframeToExcel(file, dataframesAndCharts.ReportPercentageDf, "Tenencia_Porcentaje", true, true, false)
		if err != nil {
			return nil, err
		}
	}

	if dataframesAndCharts.ReturnDF != nil {
		file, err = rs.convertReportDataframeToExcel(file, dataframesAndCharts.ReturnDF, "Retorno", false, true, false)
		if err != nil {
			return nil, err
		}
	}

	if dataframesAndCharts.ReferenceVariablesDF != nil {
		file, err = rs.convertReportDataframeToExcel(file, dataframesAndCharts.ReferenceVariablesDF, "Referencias", false, true, false)
		if err != nil {
			return nil, err
		}
	}

	err = rs.applyStylesToAllSheets(file)
	if err != nil {
		return nil, err
	}
	return file, nil
}

// isSameDate compares only the date part (day, month, year) of two time.Time values
func (rs *ReportService) isSameDate(date1, date2 time.Time) bool {
	return date1.Year() == date2.Year() &&
		date1.Month() == date2.Month() &&
		date1.Day() == date2.Day()
}

func (rs *ReportService) parseAccountsReportToDataFrame(ctx context.Context, accountsReport *schemas.AccountsReports, startDate, endDate time.Time, interval time.Duration) (*dataframe.DataFrame, error) {
	dates, err := utils.GenerateDates(startDate, endDate, interval)
	if err != nil {
		return nil, err
	}
	dateStrs := make([]string, len(dates))
	for i, date := range dates {
		dateStrs[i] = date.Format("2006-01-02")
	}

	// Initialize an empty DataFrame with the DateRequested as the index (as the first column)
	df := dataframe.New(
		series.New(dateStrs, series.String, "DateRequested"),
	)

	// Iterate through the assets and add each as a new column
	for _, assets := range *accountsReport.AssetsByCategory {
		for _, asset := range assets {
			assetValues := make([]string, len(dates))

			// Initialize all rows with empty values for this asset
			for i := range assetValues {
				assetValues[i] = "0.0" // Default value if no match found
			}

			// Iterate through holdings and match the dates to fill the corresponding values
			for _, holding := range asset.Holdings {
				if holding.DateRequested != nil {
					holdingDate := *holding.DateRequested
					// Compare only date part (day, month, year) without time
					for i, date := range dates {
						if rs.isSameDate(date, holdingDate) {
							if holding.Value >= 1.0 || holding.Value <= -1.0 {
								assetValues[i] = fmt.Sprintf("%.2f", holding.Value)
							} else {
								assetValues[i] = "0.0"
							}
							break
						}
					}
				}
			}

			// Add the new series (column) for this asset to the DataFrame
			updatedDf, err := rs.updateDataFrame(df, fmt.Sprintf("%s-%s", asset.Category, asset.ID), assetValues)
			if err != nil {
				return nil, err
			}
			df = *updatedDf
		}
	}

	totalValues := make([]string, len(dates))
	for _, totalHolding := range accountsReport.TotalHoldingsByDate {
		if totalHolding.DateRequested != nil {
			holdingDate := *totalHolding.DateRequested
			// Compare only date part (day, month, year) without time
			for i, date := range dates {
				if rs.isSameDate(date, holdingDate) {
					if totalHolding.Value >= 1.0 {
						totalValues[i] = fmt.Sprintf("%.2f", totalHolding.Value)
					} else {
						totalValues[i] = "0.0"
					}
					break
				}
			}
		}
	}
	for i, v := range totalValues {
		if v == "" {
			totalValues[i] = "0.0"
		}
	}
	// Add the new series (column) for this asset to the DataFrame
	updatedDf, err := rs.updateDataFrame(df, "TOTAL", totalValues)
	if err != nil {
		return nil, err
	}
	df = *updatedDf

	return &df, nil
}

func (rs *ReportService) parseAccountsReturnToDataFrame(ctx context.Context, accountsReport *schemas.AccountsReports, startDate, endDate time.Time, interval time.Duration) (*dataframe.DataFrame, error) {
	dates, err := utils.GenerateDates(startDate, endDate, interval)
	if err != nil {
		return nil, err
	}
	dateStrs := make([]string, len(dates))
	for i, date := range dates {
		dateStrs[i] = date.Format("2006-01-02")
	}

	// Initialize an empty DataFrame with the DateRequested as the index (as the first column)
	df := dataframe.New(
		series.New(dateStrs, series.String, "DateRequested"),
	)

	// Iterate through the assets and add each as a new column
	for _, assets := range *accountsReport.AssetsReturnByCategory {
		for _, asset := range assets {
			assetValues := make([]string, len(dates))

			// Initialize all rows with empty values for this asset
			for i := range assetValues {
				assetValues[i] = "0.0" // Default value if no match found
			}

			// Iterate through assets return and match the dates to fill the corresponding values
			for _, returnsByDate := range asset.ReturnsByDateRange {
				returnDate := returnsByDate.EndDate
				// Compare only date part (day, month, year) without time
				for i, date := range dates {
					if rs.isSameDate(date, returnDate) {
						assetValues[i] = fmt.Sprintf("%.2f", returnsByDate.ReturnPercentage)
						break
					}
				}
			}

			// Add the new series (column) for this asset to the DataFrame
			updatedDf, err := rs.updateDataFrame(df, fmt.Sprintf("%s-%s", asset.Category, asset.ID), assetValues)
			if err != nil {
				return nil, err
			}
			df = *updatedDf
		}
	}
	totalReturnValues := make([]string, len(dates))
	for _, totalReturn := range accountsReport.TotalReturns {
		for i, date := range dates {
			if rs.isSameDate(date, totalReturn.EndDate) {
				totalReturnValues[i] = fmt.Sprintf("%.2f", totalReturn.ReturnPercentage)
				continue
			}
		}
		updatedDf, err := rs.updateDataFrame(df, "TOTAL", totalReturnValues)
		if err != nil {
			return nil, err
		}
		df = *updatedDf
	}

	return &df, nil
}

func (rs *ReportService) parseReferenceVariablesToDataFrame(ctx context.Context, accountsReport *schemas.AccountsReports, startDate, endDate time.Time, interval time.Duration) (*dataframe.DataFrame, error) {
	if accountsReport.ReferenceVariables == nil {
		// Return empty dataframe if no reference variables
		dates, err := utils.GenerateDates(startDate, endDate, interval)
		if err != nil {
			return nil, err
		}
		dateStrs := make([]string, len(dates))
		for i, date := range dates {
			dateStrs[i] = date.Format("2006-01-02")
		}
		df := dataframe.New(series.New(dateStrs, series.String, "DateRequested"))
		return &df, nil
	}

	dates, err := utils.GenerateDates(startDate, endDate, interval)
	if err != nil {
		return nil, err
	}
	dateStrs := make([]string, len(dates))
	for i, date := range dates {
		dateStrs[i] = date.Format("2006-01-02")
	}

	// Initialize an empty DataFrame with the DateRequested as the index (as the first column)
	df := dataframe.New(
		series.New(dateStrs, series.String, "DateRequested"),
	)

	// Iterate through the reference variables and add each as a new column
	for variableName, variable := range *accountsReport.ReferenceVariables {
		variableValues := make([]string, len(dates))

		// Initialize all rows with empty values for this variable
		for i := range variableValues {
			variableValues[i] = "0.0" // Default value if no match found
		}

		// Iterate through variable valuations and match the dates to fill the corresponding values
		for _, valuation := range variable.Valuations {
			// Parse the string date to time.Time
			valuationDate, err := time.Parse("2006-01-02", valuation.Date)
			if err != nil {
				continue // Skip invalid dates
			}
			// Compare only date part (day, month, year) without time
			for i, date := range dates {
				if rs.isSameDate(date, valuationDate) {
					variableValues[i] = fmt.Sprintf("%.2f", valuation.Value)
					break
				}
			}
		}

		// Add the new series (column) for this variable to the DataFrame
		updatedDf, err := rs.updateDataFrame(df, variableName, variableValues)
		if err != nil {
			return nil, err
		}
		df = *updatedDf
	}

	return &df, nil
}

func (rs *ReportService) parseAccountsCategoryToDataFrame(_ context.Context, accountsReport *schemas.AccountsReports, startDate, endDate time.Time, interval time.Duration) (*dataframe.DataFrame, error) {
	dates, err := utils.GenerateDates(startDate, endDate, interval)
	if err != nil {
		return nil, err
	}
	dateStrs := make([]string, len(dates))
	for i, date := range dates {
		dateStrs[i] = date.Format("2006-01-02")
	}

	// Initialize an empty DataFrame with the DateRequested as the index (as the first column)
	df := dataframe.New(
		series.New(dateStrs, series.String, "DateRequested"),
	)

	// Iterate through the category assets and add each as a new column
	for categoryName, asset := range *accountsReport.CategoryAssets {
		categoryValues := make([]string, len(dates))

		// Initialize all rows with empty values for this category
		for i := range categoryValues {
			categoryValues[i] = "0.0" // Default value if no match found
		}

		// Iterate through holdings and match the dates to fill the corresponding values
		for _, holding := range asset.Holdings {
			if holding.DateRequested != nil {
				holdingDate := *holding.DateRequested
				// Compare only date part (day, month, year) without time
				for i, date := range dates {
					if rs.isSameDate(date, holdingDate) {
						if holding.Value >= 1.0 || holding.Value <= -1.0 {
							categoryValues[i] = fmt.Sprintf("%.2f", holding.Value)
						} else {
							categoryValues[i] = "0.0"
						}
						break
					}
				}
			}
		}

		// Add the new series (column) for this category to the DataFrame
		updatedDf, err := rs.updateDataFrame(df, categoryName, categoryValues)
		if err != nil {
			return nil, err
		}
		df = *updatedDf
	}

	return &df, nil
}

func (rs *ReportService) divideByTotal(df *dataframe.DataFrame) *dataframe.DataFrame {
	if df == nil || df.Nrow() == 0 || df.Ncol() == 0 {
		return df
	}

	// Check if TOTAL column exists
	colNames := df.Names()
	totalColExists := false
	for _, name := range colNames {
		if name == "TOTAL" {
			totalColExists = true
			break
		}
	}

	if !totalColExists {
		return df
	}

	// Get the "TOTAL" column
	totalCol := df.Col("TOTAL")
	if totalCol.Len() == 0 {
		return df
	}

	// Create a new DataFrame with the same structure
	newDf := df.Copy()

	// Iterate through each column (except DateRequested and TOTAL)
	for _, colName := range df.Names() {
		if colName == "DateRequested" || colName == "TOTAL" {
			continue
		}

		col := df.Col(colName)
		if col.Len() == 0 {
			continue
		}

		newValues := make([]string, col.Len())

		for i := 0; i < col.Len(); i++ {
			colValue := col.Elem(i).Float()
			totalValue := totalCol.Elem(i).Float()

			if totalValue == 0 {
				newValues[i] = "0.0"
				continue
			}

			percentage := (colValue / totalValue) * 100
			newValues[i] = fmt.Sprintf("%.2f", percentage)
		}

		// Update the column with new values
		newSeries := series.New(newValues, series.String, colName)
		newDf = newDf.Mutate(newSeries)
	}

	return &newDf
}

func (rs *ReportService) updateDataFrame(df dataframe.DataFrame, columnName string, newValues []string) (*dataframe.DataFrame, error) {
	// Create a new series with the column name and values
	newSeries := series.New(newValues, series.String, columnName)

	// Add the new series to the DataFrame
	newDf := df.Mutate(newSeries)

	return &newDf, nil
}

func (rs *ReportService) convertReportDataframeToExcel(
	f *excelize.File,
	reportDf *dataframe.DataFrame,
	sheetName string,
	percentageData bool,
	includeBarGraph bool,
	includePieGraph bool,
) (*excelize.File, error) {
	// Check if DataFrame is nil or empty
	if reportDf == nil || reportDf.Nrow() == 0 || reportDf.Ncol() == 0 {
		// Return the file as is if DataFrame is nil/empty
		return f, nil
	}

	// Create a new Excel file
	var err error
	var index int

	if f == nil {
		f = excelize.NewFile()
		err := f.SetSheetName("Sheet1", sheetName)
		if err != nil {
			return nil, err
		}
	} else {
		index, err = f.NewSheet(sheetName)
		if err != nil {
			return nil, err
		}
		defer f.SetActiveSheet(index)
	}

	// Define variables for column and row indices
	startRow := 1
	categoryRow := startRow
	idRow := startRow + 1

	// Extract column names from DataFrame
	cols := reportDf.Names()

	// Variables to track categories and column ranges for merging
	categoryStartCol := make(map[string]int)
	categoryEndCol := make(map[string]int)

	// Set the DateRequested in the first column (A) for both the first and second rows
	err = f.SetCellValue(sheetName, "A2", "Fecha")
	if err != nil {
		return nil, err
	}

	// Iterate over the columns in the DataFrame, starting from the second one (ignoring DateRequested)
	columnIndex := 2               // Excel columns start from B (index 2) for data columns
	for _, col := range cols[1:] { // Skip the first column "DateRequested"

		var category, id string

		if col == "TOTAL" {
			category = "TOTAL"
			id = "-"
		} else {
			parts := strings.Split(col, "-")
			if len(parts) >= 2 {
				category = parts[0]
				id = parts[1]
			} else {
				category = col
				id = "-"
			}
		}

		// Set the ID in the second row (e.g., ID1, ID2, etc.)
		cell := fmt.Sprintf("%s%d", rs.toAlphaString(columnIndex), idRow)
		err = f.SetCellValue(sheetName, cell, id)
		if err != nil {
			return nil, err
		}

		// Track the start and end columns for merging categories
		if _, exists := categoryStartCol[category]; !exists {
			categoryStartCol[category] = columnIndex
		}
		categoryEndCol[category] = columnIndex

		columnIndex++
	}

	// Merge cells for each category and set the category name in the first row
	for category, startCol := range categoryStartCol {
		endCol := categoryEndCol[category]
		startCell := fmt.Sprintf("%s%d", rs.toAlphaString(startCol), categoryRow)
		endCell := fmt.Sprintf("%s%d", rs.toAlphaString(endCol), categoryRow)
		err = f.MergeCell(sheetName, startCell, endCell)
		if err != nil {
			return nil, err
		}
		err = f.SetCellValue(sheetName, startCell, category)
		if err != nil {
			return nil, err
		}
	}

	numFmt := 8
	if percentageData {
		numFmt = 10
	}
	// Format cells as currency
	cellStyle, err := f.NewStyle(&excelize.Style{
		NumFmt: numFmt,
	})
	if err != nil {
		return nil, err
	}

	// Now fill in the data for the rest of the rows starting from the third row
	for rowIndex, row := range reportDf.Records()[1:] { // Skip the first row (headers)
		for colIndex, cellValue := range row {
			cell := fmt.Sprintf("%s%d", rs.toAlphaString(colIndex+1), rowIndex+3) // colIndex+1 to skip DateRequested
			numCellValue, err := strconv.ParseFloat(cellValue, 64)
			if err != nil {
				err = f.SetCellValue(sheetName, cell, cellValue)
				if err != nil {
					return nil, err
				}
			} else {
				err = f.SetCellValue(sheetName, cell, numCellValue)
				if err != nil {
					return nil, err
				}
			}

			if err = f.SetCellStyle(sheetName, cell, cell, cellStyle); err != nil {
				return nil, err
			}
		}
	}

	return f, nil
}

func (rs *ReportService) toAlphaString(column int) string {
	result := ""
	for column > 0 {
		column--
		result = string(rune('A'+column%26)) + result
		column /= 26
	}
	return result
}

func (rs *ReportService) applyStylesToAllSheets(f *excelize.File) error {
	sheets := f.GetSheetList()
	for _, sheetName := range sheets {
		// Get the used range
		rows, err := f.GetRows(sheetName)
		if err != nil {
			return err
		}

		if len(rows) == 0 {
			continue
		}

		// Get the last row and column
		lastRow := len(rows)
		lastCol := len(rows[0])

		// Create a style for the header rows
		headerStyle, err := f.NewStyle(&excelize.Style{
			Font: &excelize.Font{
				Bold: true,
			},
			Fill: excelize.Fill{
				Type:    "pattern",
				Color:   []string{"#E6E6E6"},
				Pattern: 1,
			},
			Border: []excelize.Border{
				{Type: "left", Color: "000000", Style: 1},
				{Type: "top", Color: "000000", Style: 1},
				{Type: "bottom", Color: "000000", Style: 1},
				{Type: "right", Color: "000000", Style: 1},
			},
			Alignment: &excelize.Alignment{
				Horizontal: "center",
				Vertical:   "center",
			},
		})
		if err != nil {
			return err
		}

		// Apply header style to the first two rows
		err = f.SetCellStyle(sheetName, "A1", fmt.Sprintf("%s2", rs.toAlphaString(lastCol)), headerStyle)
		if err != nil {
			return err
		}

		// Create a style for data cells
		dataStyle, err := f.NewStyle(&excelize.Style{
			Border: []excelize.Border{
				{Type: "left", Color: "000000", Style: 1},
				{Type: "top", Color: "000000", Style: 1},
				{Type: "bottom", Color: "000000", Style: 1},
				{Type: "right", Color: "000000", Style: 1},
			},
			Alignment: &excelize.Alignment{
				Horizontal: "center",
				Vertical:   "center",
			},
		})
		if err != nil {
			return err
		}

		// Apply data style to the rest of the rows
		if lastRow > 2 {
			err = f.SetCellStyle(sheetName, "A3", fmt.Sprintf("%s%d", rs.toAlphaString(lastCol), lastRow), dataStyle)
			if err != nil {
				return err
			}
		}

		// Auto-fit columns
		for i := 1; i <= lastCol; i++ {
			colName := rs.toAlphaString(i)
			err = f.SetColWidth(sheetName, colName, colName, 15)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
