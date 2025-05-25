package controllers

import (
	"context"
	"fmt"
	"server/src/clients/esco"
	"server/src/schemas"
	"server/src/services"
	"server/src/utils"
	"sort"
	"strconv"
	"strings"
	"time"
)

type AccountsControllerI interface {
	GetAllAccounts(ctx context.Context, token, filter string) ([]*schemas.AccountReponse, error)
	GetAccountByID(ctx context.Context, token, id string) (*esco.CuentaSchema, error)
	GetAccountState(ctx context.Context, token, id string, date time.Time) (*schemas.AccountState, error)
	GetAccountStateWithTransactionsDateRange(ctx context.Context, token, id string, startDate, endDate time.Time, interval time.Duration) (*schemas.AccountState, error)
	GetAccountStateDateRange(ctx context.Context, token, id string, startDate, endDate time.Time, interval time.Duration) (*schemas.AccountState, error)
	GetLiquidacionesDateRange(ctx context.Context, token, id string, startDate, endDate time.Time) (*schemas.AccountState, error)
	GetBoletosDateRange(ctx context.Context, token, id string, startDate, endDate time.Time) (*schemas.AccountState, error)
	GetMultiAccountStateWithTransactionsDateRange(ctx context.Context, token string, ids []string, startDate, endDate time.Time, interval time.Duration) ([]*schemas.AccountState, error)
	GetMultiAccountStateByCategoryDateRange(ctx context.Context, token string, ids []string, startDate, endDate time.Time, interval time.Duration) (*schemas.AccountStateByCategory, error)
	SyncAccount(ctx context.Context, token, accountID string, startDate, endDate time.Time) (*schemas.AccountState, error)
}

type AccountsController struct {
	ESCOClient  esco.ESCOServiceClientI
	ESCOService services.ESCOServiceI
	SyncService services.SyncServiceI
}

func NewAccountsController(escoClient esco.ESCOServiceClientI, escoService services.ESCOServiceI, syncService services.SyncServiceI) *AccountsController {
	return &AccountsController{
		ESCOClient:  escoClient,
		ESCOService: escoService,
		SyncService: syncService,
	}
}

func (c *AccountsController) GetAllAccounts(ctx context.Context, token, filter string) ([]*schemas.AccountReponse, error) {
	if filter == "" {
		filter = "*"
	}
	accs, err := c.ESCOClient.BuscarCuentas(token, filter)
	if err != nil {
		return nil, err
	}
	accounts := make([]*schemas.AccountReponse, len(accs))
	for i, account := range accs {
		accounts[i] = &schemas.AccountReponse{ID: strconv.Itoa(account.N), CID: account.ID, FID: account.FI, Name: account.D}
	}
	return accounts, nil
}

func (c *AccountsController) GetAccountByID(ctx context.Context, token, id string) (*esco.CuentaSchema, error) {
	return c.ESCOService.GetAccountByID(ctx, token, id)
}

func (c *AccountsController) GetAccountState(ctx context.Context, token, id string, date time.Time) (*schemas.AccountState, error) {
	return c.ESCOService.GetAccountState(ctx, token, id, date)
}

func (c *AccountsController) GetAccountStateWithTransactionsDateRange(ctx context.Context, token, id string, startDate, endDate time.Time, interval time.Duration) (*schemas.AccountState, error) {
	return c.ESCOService.GetAccountStateWithTransactions(ctx, token, id, startDate, endDate, interval)
}

func (c *AccountsController) GetAccountStateDateRange(ctx context.Context, token, id string, startDate, endDate time.Time, interval time.Duration) (*schemas.AccountState, error) {
	return c.ESCOService.GetAccountStateDateRange(ctx, token, id, startDate, endDate, interval)
}

func (c *AccountsController) GetLiquidacionesDateRange(ctx context.Context, token, id string, startDate, endDate time.Time) (*schemas.AccountState, error) {
	return c.ESCOService.GetLiquidacionesDateRange(ctx, token, id, startDate, endDate)
}

func (c *AccountsController) GetBoletosDateRange(ctx context.Context, token, id string, startDate, endDate time.Time) (*schemas.AccountState, error) {
	return c.ESCOService.GetBoletosDateRange(ctx, token, id, startDate, endDate)
}

func (c *AccountsController) GetMultiAccountStateWithTransactionsDateRange(ctx context.Context, token string, ids []string, startDate, endDate time.Time, interval time.Duration) ([]*schemas.AccountState, error) {
	return c.ESCOService.GetMultiAccountStateWithTransactions(ctx, token, ids, startDate, endDate, interval)
}

func (c *AccountsController) GetMultiAccountStateByCategoryDateRange(ctx context.Context, token string, ids []string, startDate, endDate time.Time, interval time.Duration) (*schemas.AccountStateByCategory, error) {
	return c.ESCOService.GetMultiAccountStateByCategory(ctx, token, ids, startDate, endDate, interval)
}

func (c *AccountsController) GetCtaCteConsolidadoDateRange(ctx context.Context, token, id string, startDate, endDate time.Time) (*schemas.AccountState, error) {

	account, err := c.GetAccountByID(ctx, token, id)
	if err != nil {
		return nil, err
	}
	instrumentos, err := c.ESCOClient.GetCtaCteConsolidado(token, account.ID, account.FI, strconv.Itoa(account.N), "0", startDate, endDate, false)
	if err != nil {
		return nil, err
	}
	return c.parseInstrumentosRecoveriesToAccountState(&instrumentos)
}

func (c *AccountsController) parseInstrumentosRecoveriesToAccountState(instrumentos *[]esco.Instrumentos) (*schemas.AccountState, error) {
	if instrumentos == nil {
		return nil, nil
	}
	var id, currencySign, categoryKey string
	var units, value float64
	categoryMap := c.ESCOClient.GetCategoryMap()
	accStateRes := schemas.NewAccountState()
	for _, ins := range *instrumentos {

		if ins.C < float64(0) && strings.Contains(ins.D, "Retiro de Títulos") {
			id = strings.Split(strings.Split(ins.I, " - ")[1], " /")[0]
			currencySign = ins.PR_S
			units = -ins.C
			value = ins.N
			categoryKey = fmt.Sprintf("%s / %s", ins.F, id)
		} else if strings.Contains(ins.D, "Renta") || strings.Contains(ins.D, "Boleto") {
			id = strings.Split(ins.I, " - ")[1]
			if id == "$" {
				continue
			}
			currencySign = "$"
			units = -ins.C
			value = ins.N
			categoryKey = "CCL"
		} else {
			continue
		}
		var asset schemas.Asset
		var exists bool
		var parsedDate *time.Time

		if asset, exists = (*accStateRes.Assets)[id]; !exists {
			var category string
			var exists bool
			if category, exists = categoryMap[categoryKey]; !exists {
				category = "S / C"
			}
			(*accStateRes.Assets)[id] = schemas.Asset{
				ID:           id,
				Type:         "",
				Denomination: categoryKey,
				Category:     category,
				Transactions: make([]schemas.Transaction, 0, len(*instrumentos)),
			}
			asset = (*accStateRes.Assets)[id]
		}
		if ins.FL != "" {
			// Parse the date as before
			p, err := time.Parse(utils.ShortSlashDateLayout, ins.FL)
			if err != nil {
				return nil, err
			}

			// Set the time to 23:00 in the UTC-3 timezone
			loc, _ := time.LoadLocation("America/Argentina/Buenos_Aires") // UTC-3 timezone (Argentina time)
			p = time.Date(p.Year(), p.Month(), p.Day(), 23, 0, 0, 0, loc)

			parsedDate = &p
		} else {
			parsedDate = nil
		}
		asset.Transactions = append(asset.Transactions, schemas.Transaction{
			Currency:     "Pesos",
			CurrencySign: currencySign,
			Value:        value,
			Units:        -units,
			Date:         parsedDate,
		})
		(*accStateRes.Assets)[id] = asset
	}

	return accStateRes, nil
}

func sortHoldingsByDateRequested(asset *schemas.Asset) {
	sort.Slice(asset.Holdings, func(i, j int) bool {
		return asset.Holdings[i].DateRequested.Before(*asset.Holdings[j].DateRequested)
	})
}

func (c *AccountsController) CollapseAndGroupAccountsStates(accountsStates []*schemas.AccountState) *schemas.AccountStateByCategory {
	collapsedAccountState := collapseAccountStates(accountsStates)
	return groupTotalHoldingsAndTransactionsByDate(&collapsedAccountState)
}

// Group assets by category after collapsing, with sorting for consistent ordering
// In addition of calculating the total holding value
func groupTotalHoldingsAndTransactionsByDate(state *schemas.AccountState) *schemas.AccountStateByCategory {
	totalHoldingsByDate := make(map[string]schemas.Holding)
	totalTransactionsByDate := make(map[string]schemas.Transaction)

	// Assets grouped by category
	assetsByCategory := make(map[string][]schemas.Asset)

	// Joined assets by category as new holdings
	categoryHoldingsByDate := make(map[string]map[string]schemas.Holding)

	// Joined asset transactions by category as new transactions
	categoryTransactionsByDate := make(map[string]map[string]schemas.Transaction)

	for _, asset := range *state.Assets {
		category := asset.Category
		assetsByCategory[category] = append(assetsByCategory[category], asset)

		if _, exists := categoryHoldingsByDate[category]; !exists {
			categoryHoldingsByDate[category] = make(map[string]schemas.Holding)
		}

		if _, exists := categoryTransactionsByDate[category]; !exists {
			categoryTransactionsByDate[category] = make(map[string]schemas.Transaction)
		}

		for _, holding := range asset.Holdings {
			date := *holding.DateRequested
			dateStr := date.Format("2006-01-02")
			if _, exists := totalHoldingsByDate[dateStr]; !exists {
				totalHoldingsByDate[dateStr] = schemas.Holding{
					Currency:      "Pesos",
					CurrencySign:  "$",
					Value:         0,
					DateRequested: &date,
					Date:          &date,
				}
			}

			total := totalHoldingsByDate[dateStr]
			total.Value += holding.Value
			totalHoldingsByDate[dateStr] = total

			if _, exists := categoryHoldingsByDate[category][dateStr]; !exists {
				categoryHoldingsByDate[category][dateStr] = schemas.Holding{
					Currency:      "Pesos",
					CurrencySign:  "$",
					Value:         0,
					DateRequested: &date,
					Date:          &date,
				}
			}

			categoryHolding := categoryHoldingsByDate[category][dateStr]
			categoryHolding.Value += holding.Value
			categoryHoldingsByDate[category][dateStr] = categoryHolding
		}

		for _, transaction := range asset.Transactions {
			date := *transaction.Date
			dateStr := date.Format("2006-01-02")
			if _, exists := totalTransactionsByDate[dateStr]; !exists {
				totalTransactionsByDate[dateStr] = schemas.Transaction{
					Currency:     "Pesos",
					CurrencySign: "$",
					Value:        0,
					Date:         &date,
				}
			}
			total := totalTransactionsByDate[dateStr]
			total.Value += transaction.Value
			totalTransactionsByDate[dateStr] = total

			if _, exists := categoryTransactionsByDate[category][dateStr]; !exists {
				categoryTransactionsByDate[category][dateStr] = schemas.Transaction{
					Currency:     "Pesos",
					CurrencySign: "$",
					Value:        0,
					Units:        0,
					Date:         &date,
				}
			}

			categoryTransaction := categoryTransactionsByDate[category][dateStr]
			categoryTransaction.Value += transaction.Value
			categoryTransactionsByDate[category][dateStr] = categoryTransaction
		}
	}
	// Sort each category's assets by ID for consistent ordering
	for category := range assetsByCategory {
		sort.Slice(assetsByCategory[category], func(i, j int) bool {
			return assetsByCategory[category][i].ID < assetsByCategory[category][j].ID
		})
	}

	categoryAssets := generateCategoryAssets(categoryHoldingsByDate, categoryTransactionsByDate)

	return &schemas.AccountStateByCategory{
		AssetsByCategory:        &assetsByCategory,
		CategoryAssets:          &categoryAssets,
		TotalHoldingsByDate:     &totalHoldingsByDate,
		TotalTransactionsByDate: &totalTransactionsByDate,
	}
}

// Collapse multiple account states into one, ensuring consistent aggregation and ordering of holdings and transactions
func collapseAccountStates(states []*schemas.AccountState) schemas.AccountState {
	holdingMapByAssetID := make(map[string]map[string]schemas.Holding)
	transactionMapByAssetID := make(map[string]map[string]schemas.Transaction)
	assetMapByID := make(map[string]schemas.Asset)

	for _, state := range states {
		if state.Assets == nil {
			continue
		}
		var holdingMap map[string]schemas.Holding
		var transactionMap map[string]schemas.Transaction
		var found bool
		for assetID, asset := range *state.Assets {
			if _, found := assetMapByID[assetID]; !found {
				assetMapByID[assetID] = asset
			}
			if holdingMap, found = holdingMapByAssetID[assetID]; !found {
				holdingMapByAssetID[assetID] = make(map[string]schemas.Holding)
				holdingMap = holdingMapByAssetID[assetID]
			}
			if transactionMap, found = transactionMapByAssetID[assetID]; !found {
				transactionMapByAssetID[assetID] = make(map[string]schemas.Transaction)
				transactionMap = transactionMapByAssetID[assetID]
			}
			for _, holding := range asset.Holdings {
				date := holding.DateRequested.Format("2006-01-02")
				if existing, found := holdingMap[date]; !found {
					holdingMap[date] = holding
				} else {
					existing.Value += holding.Value
					holdingMap[date] = existing
				}
			}
			holdingMapByAssetID[assetID] = holdingMap

			for _, transaction := range asset.Transactions {
				key := transaction.Date.Format("2006-01-02")
				if existing, found := transactionMap[key]; !found {
					transactionMap[key] = transaction
				} else {
					existing.Value += transaction.Value
					existing.Units += transaction.Units
					transactionMap[key] = existing
				}
			}
			transactionMapByAssetID[assetID] = transactionMap
		}
	}
	collapsed := make(map[string]schemas.Asset)
	for assetID, asset := range assetMapByID {
		var existing schemas.Asset
		var ok bool
		if existing, ok = collapsed[assetID]; !ok {
			collapsed[assetID] = schemas.Asset{
				ID:           assetID,
				Type:         asset.Type,
				Category:     asset.Category,
				Denomination: asset.Denomination,
				Holdings:     []schemas.Holding{},
				Transactions: []schemas.Transaction{},
			}
			existing = collapsed[assetID]
		}
		holdings := existing.Holdings
		for _, holding := range holdingMapByAssetID[assetID] {
			holdings = append(holdings, holding)
		}
		existing.Holdings = holdings
		// Sort holdings for consistent order
		sort.SliceStable(existing.Holdings, func(i, j int) bool {
			return existing.Holdings[i].DateRequested.Before(*existing.Holdings[j].DateRequested)
		})

		transactions := existing.Transactions
		for _, transaction := range transactionMapByAssetID[assetID] {
			transactions = append(transactions, transaction)
		}
		existing.Transactions = transactions
		// Sort holdings for consistent order
		sort.SliceStable(existing.Transactions, func(i, j int) bool {
			return existing.Transactions[i].Date.Before(*existing.Transactions[j].Date)
		})

		collapsed[assetID] = existing
	}
	return schemas.AccountState{Assets: &collapsed}
}

func generateCategoryAssets(
	categoryHoldings map[string]map[string]schemas.Holding,
	categoryTransactions map[string]map[string]schemas.Transaction,
) map[string]schemas.Asset {
	categoryAssets := map[string]schemas.Asset{}
	for category, holdingsByDate := range categoryHoldings {
		if _, exist := categoryAssets[category]; !exist {
			categoryAssets[category] = schemas.Asset{
				ID:           category,
				Type:         "Category",
				Denomination: category,
				Category:     category,
				Holdings:     []schemas.Holding{},
				Transactions: []schemas.Transaction{},
			}
		}
		categoryAsset := categoryAssets[category]
		for _, holding := range holdingsByDate {
			categoryAsset.Holdings = append(categoryAsset.Holdings, holding)
		}
		categoryAssets[category] = categoryAsset
	}

	for category, transactionsByDate := range categoryTransactions {
		if _, exist := categoryAssets[category]; !exist {
			categoryAssets[category] = schemas.Asset{
				ID:           category,
				Type:         "Category",
				Denomination: category,
				Category:     category,
				Holdings:     []schemas.Holding{},
				Transactions: []schemas.Transaction{},
			}
		}
		categoryAsset := categoryAssets[category]
		for _, transaction := range transactionsByDate {
			categoryAsset.Transactions = append(categoryAsset.Transactions, transaction)
		}
		categoryAssets[category] = categoryAsset
	}
	return categoryAssets
}

// SyncAccount syncs account data for a given account ID and date range
func (c *AccountsController) SyncAccount(ctx context.Context, token, accountID string, startDate, endDate time.Time) (*schemas.AccountState, error) {

	// Use syncService to sync the data
	err := c.SyncService.SyncDataFromAccount(ctx, token, accountID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	// Get the synced account state
	accountState, err := c.ESCOService.GetAccountStateWithTransactions(ctx, token, accountID, startDate, endDate, time.Hour*24)
	if err != nil {
		return nil, err
	}

	return accountState, nil
}
