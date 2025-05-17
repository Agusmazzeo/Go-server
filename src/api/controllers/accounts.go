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
}

type AccountsController struct {
	ESCOClient  esco.ESCOServiceClientI
	ESCOService services.ESCOServiceI
}

func NewAccountsController(escoClient esco.ESCOServiceClientI, escoService services.ESCOServiceI) *AccountsController {
	return &AccountsController{
		ESCOClient:  escoClient,
		ESCOService: escoService,
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
		var voucher schemas.Voucher
		var exists bool
		var parsedDate *time.Time

		if voucher, exists = (*accStateRes.Vouchers)[id]; !exists {
			var category string
			var exists bool
			if category, exists = categoryMap[categoryKey]; !exists {
				category = "S / C"
			}
			(*accStateRes.Vouchers)[id] = schemas.Voucher{
				ID:           id,
				Type:         "",
				Denomination: categoryKey,
				Category:     category,
				Transactions: make([]schemas.Transaction, 0, len(*instrumentos)),
			}
			voucher = (*accStateRes.Vouchers)[id]
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
		voucher.Transactions = append(voucher.Transactions, schemas.Transaction{
			Currency:     "Pesos",
			CurrencySign: currencySign,
			Value:        value,
			Units:        -units,
			Date:         parsedDate,
		})
		(*accStateRes.Vouchers)[id] = voucher
	}

	return accStateRes, nil
}

func sortHoldingsByDateRequested(voucher *schemas.Voucher) {
	sort.Slice(voucher.Holdings, func(i, j int) bool {
		return voucher.Holdings[i].DateRequested.Before(*voucher.Holdings[j].DateRequested)
	})
}

func (c *AccountsController) CollapseAndGroupAccountsStates(accountsStates []*schemas.AccountState) *schemas.AccountStateByCategory {
	collapsedAccountState := collapseAccountStates(accountsStates)
	return groupTotalHoldingsAndTransactionsByDate(&collapsedAccountState)
}

// Group vouchers by category after collapsing, with sorting for consistent ordering
// In addition of calculating the total holding value
func groupTotalHoldingsAndTransactionsByDate(state *schemas.AccountState) *schemas.AccountStateByCategory {
	totalHoldingsByDate := make(map[string]schemas.Holding)
	totalTransactionsByDate := make(map[string]schemas.Transaction)

	// Vouchers grouped by category
	vouchersByCategory := make(map[string][]schemas.Voucher)

	// Joined vouchers by category as new holdings
	categoryHoldingsByDate := make(map[string]map[string]schemas.Holding)

	// Joined voucher transactions by category as new transactions
	categoryTransactionsByDate := make(map[string]map[string]schemas.Transaction)

	for _, voucher := range *state.Vouchers {
		category := voucher.Category
		vouchersByCategory[category] = append(vouchersByCategory[category], voucher)

		if _, exists := categoryHoldingsByDate[category]; !exists {
			categoryHoldingsByDate[category] = make(map[string]schemas.Holding)
		}

		if _, exists := categoryTransactionsByDate[category]; !exists {
			categoryTransactionsByDate[category] = make(map[string]schemas.Transaction)
		}

		for _, holding := range voucher.Holdings {
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

		for _, transaction := range voucher.Transactions {
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
	// Sort each category's vouchers by ID for consistent ordering
	for category := range vouchersByCategory {
		sort.Slice(vouchersByCategory[category], func(i, j int) bool {
			return vouchersByCategory[category][i].ID < vouchersByCategory[category][j].ID
		})
	}

	categoryVouchers := generateCategoryVouchers(categoryHoldingsByDate, categoryTransactionsByDate)

	return &schemas.AccountStateByCategory{
		VouchersByCategory:      &vouchersByCategory,
		CategoryVouchers:        &categoryVouchers,
		TotalHoldingsByDate:     &totalHoldingsByDate,
		TotalTransactionsByDate: &totalTransactionsByDate,
	}
}

// Collapse multiple account states into one, ensuring consistent aggregation and ordering of holdings and transactions
func collapseAccountStates(states []*schemas.AccountState) schemas.AccountState {
	holdingMapByVoucherID := make(map[string]map[string]schemas.Holding)
	transactionMapByVoucherID := make(map[string]map[string]schemas.Transaction)
	voucherMapByID := make(map[string]schemas.Voucher)

	for _, state := range states {
		if state.Vouchers == nil {
			continue
		}
		var holdingMap map[string]schemas.Holding
		var transactionMap map[string]schemas.Transaction
		var found bool
		for voucherID, voucher := range *state.Vouchers {
			if _, found := voucherMapByID[voucherID]; !found {
				voucherMapByID[voucherID] = voucher
			}
			if holdingMap, found = holdingMapByVoucherID[voucherID]; !found {
				holdingMapByVoucherID[voucherID] = make(map[string]schemas.Holding)
				holdingMap = holdingMapByVoucherID[voucherID]
			}
			if transactionMap, found = transactionMapByVoucherID[voucherID]; !found {
				transactionMapByVoucherID[voucherID] = make(map[string]schemas.Transaction)
				transactionMap = transactionMapByVoucherID[voucherID]
			}
			for _, holding := range voucher.Holdings {
				date := holding.DateRequested.Format("2006-01-02")
				if existing, found := holdingMap[date]; !found {
					holdingMap[date] = holding
				} else {
					existing.Value += holding.Value
					holdingMap[date] = existing
				}
			}
			holdingMapByVoucherID[voucherID] = holdingMap

			for _, transaction := range voucher.Transactions {
				key := transaction.Date.Format("2006-01-02")
				if existing, found := transactionMap[key]; !found {
					transactionMap[key] = transaction
				} else {
					existing.Value += transaction.Value
					existing.Units += transaction.Units
					transactionMap[key] = existing
				}
			}
			transactionMapByVoucherID[voucherID] = transactionMap
		}
	}
	collapsed := make(map[string]schemas.Voucher)
	for voucherID, voucher := range voucherMapByID {
		var existing schemas.Voucher
		var ok bool
		if existing, ok = collapsed[voucherID]; !ok {
			collapsed[voucherID] = schemas.Voucher{
				ID:           voucherID,
				Type:         voucher.Type,
				Category:     voucher.Category,
				Denomination: voucher.Denomination,
				Holdings:     []schemas.Holding{},
				Transactions: []schemas.Transaction{},
			}
			existing = collapsed[voucherID]
		}
		holdings := existing.Holdings
		for _, holding := range holdingMapByVoucherID[voucherID] {
			holdings = append(holdings, holding)
		}
		existing.Holdings = holdings
		// Sort holdings for consistent order
		sort.SliceStable(existing.Holdings, func(i, j int) bool {
			return existing.Holdings[i].DateRequested.Before(*existing.Holdings[j].DateRequested)
		})

		transactions := existing.Transactions
		for _, transaction := range transactionMapByVoucherID[voucherID] {
			transactions = append(transactions, transaction)
		}
		existing.Transactions = transactions
		// Sort holdings for consistent order
		sort.SliceStable(existing.Transactions, func(i, j int) bool {
			return existing.Transactions[i].Date.Before(*existing.Transactions[j].Date)
		})

		collapsed[voucherID] = existing
	}
	return schemas.AccountState{Vouchers: &collapsed}
}

func generateCategoryVouchers(
	categoryHoldings map[string]map[string]schemas.Holding,
	categoryTransactions map[string]map[string]schemas.Transaction,
) map[string]schemas.Voucher {
	categoryVouchers := map[string]schemas.Voucher{}
	for category, holdingsByDate := range categoryHoldings {
		if _, exist := categoryVouchers[category]; !exist {
			categoryVouchers[category] = schemas.Voucher{
				ID:           category,
				Type:         "Category",
				Denomination: category,
				Category:     category,
				Holdings:     []schemas.Holding{},
				Transactions: []schemas.Transaction{},
			}
		}
		categoryVoucher := categoryVouchers[category]
		for _, holding := range holdingsByDate {
			categoryVoucher.Holdings = append(categoryVoucher.Holdings, holding)
		}
		categoryVouchers[category] = categoryVoucher
	}

	for category, transactionsByDate := range categoryTransactions {
		if _, exist := categoryVouchers[category]; !exist {
			categoryVouchers[category] = schemas.Voucher{
				ID:           category,
				Type:         "Category",
				Denomination: category,
				Category:     category,
				Holdings:     []schemas.Holding{},
				Transactions: []schemas.Transaction{},
			}
		}
		categoryVoucher := categoryVouchers[category]
		for _, transaction := range transactionsByDate {
			categoryVoucher.Transactions = append(categoryVoucher.Transactions, transaction)
		}
		categoryVouchers[category] = categoryVoucher
	}
	return categoryVouchers
}
