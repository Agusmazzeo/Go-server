package controllers

import (
	"context"
	"fmt"
	"server/src/clients/esco"
	"server/src/schemas"
	"server/src/utils"
	"sort"
	"strconv"
	"strings"
	"sync"
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
	ESCOClient esco.ESCOServiceClientI
}

func NewAccountsController(escoClient esco.ESCOServiceClientI) *AccountsController {
	return &AccountsController{ESCOClient: escoClient}
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
	acc, err := c.ESCOClient.BuscarCuentas(token, id)
	if err != nil {
		return nil, err
	}
	if len(acc) == 0 {
		return nil, fmt.Errorf("no accounts received for given id %s", id)
	}
	if len(acc) != 1 {
		return nil, fmt.Errorf("more than 1 account received for given id %s", id)
	}
	return &acc[0], nil
}

func (c *AccountsController) GetAccountState(ctx context.Context, token, id string, date time.Time) (*schemas.AccountState, error) {

	account, err := c.GetAccountByID(ctx, token, id)
	if err != nil {
		return nil, err
	}
	accStateData, err := c.ESCOClient.GetEstadoCuenta(token, account.ID, account.FI, strconv.Itoa(account.N), "-1", date, false)
	if err != nil {
		return nil, err
	}
	return c.parseEstadoToAccountState(&accStateData, &date)
}

func (c *AccountsController) GetAccountStateWithTransactionsDateRange(ctx context.Context, token, id string, startDate, endDate time.Time, interval time.Duration) (*schemas.AccountState, error) {
	var err error
	logger := utils.LoggerFromContext(ctx)
	var wg sync.WaitGroup
	wg.Add(3)

	var accountState *schemas.AccountState
	var liquidaciones *schemas.AccountState
	var boletos *schemas.AccountState
	go func() {
		accountState, err = c.GetAccountStateDateRange(ctx, token, id, startDate, endDate, interval)
		if err != nil {
			logger.Errorf("error while on GetAccountStateDateRange: %v", err)
		}
		wg.Done()
	}()
	go func() {
		retries := 3
		for {
			liquidaciones, err = c.GetLiquidacionesDateRange(ctx, token, id, startDate, endDate)
			if err != nil {
				logger.Errorf("error while on GetLiquidacionesDateRange: %v. Retrying...", err)
				retries--
			} else {
				break
			}
			if retries == 0 {
				logger.Errorf("exhausted retries on GetLiquidacionesDateRange: %v. Retrying...", err)
				break
			}
		}
		wg.Done()
	}()
	go func() {
		retries := 3
		for {
			boletos, err = c.GetBoletosDateRange(ctx, token, id, startDate, endDate)
			if err != nil {
				logger.Errorf("error while on GetBoletosDateRange: %v. Retrying...", err)
				retries--
			} else {
				break
			}
			if retries == 0 {
				logger.Errorf("exhausted retries on GetBoletosDateRange: %v. Retrying...", err)
				break
			}
		}
		wg.Done()
	}()
	wg.Wait()
	if err != nil {
		return nil, err
	}
	if accountState == nil {
		logger.Warn("empty account state received")
		return nil, fmt.Errorf("empty account state received")
	}

	for id := range *accountState.Vouchers {
		voucher := (*accountState.Vouchers)[id]
		if boletos != nil {
			if boleto, ok := (*boletos.Vouchers)[id]; ok {
				voucher.Transactions = append(voucher.Transactions, boleto.Transactions...)
				(*accountState.Vouchers)[id] = voucher
			}
		}
		if liquidaciones != nil {
			if liquidacion, ok := (*liquidaciones.Vouchers)[id]; ok {
				voucher.Transactions = append(voucher.Transactions, liquidacion.Transactions...)
				(*accountState.Vouchers)[id] = voucher
			}
		}
	}
	return accountState, nil
}

func (c *AccountsController) GetAccountStateDateRange(ctx context.Context, token, id string, startDate, endDate time.Time, interval time.Duration) (*schemas.AccountState, error) {
	logger := utils.LoggerFromContext(ctx)
	account, err := c.GetAccountByID(ctx, token, id)
	if err != nil {
		logger.Errorf("error while on GetAccountByID: %v", err)
		return nil, err
	}
	intervalHours := interval.Hours()
	// Calculate the number of days between startDate and endDate
	numDays := int(endDate.Sub(startDate).Hours()/intervalHours) + 1
	vouchers := make(map[string]schemas.Voucher)

	var wg sync.WaitGroup
	var errChan = make(chan error, numDays)
	var voucherChan = make(chan *schemas.AccountState, numDays)
	wg.Add(numDays)
	for i := 0; i < numDays; i++ {
		go func(i int) {
			defer wg.Done()
			var retries = 3
			var accStateData []esco.EstadoCuentaSchema
			date := startDate.AddDate(0, 0, i*int(intervalHours/24))
			for {
				accStateData, err = c.ESCOClient.GetEstadoCuenta(token, account.ID, account.FI, strconv.Itoa(account.N), "-1", date, false)
				if err != nil || accStateData == nil {
					retries -= 1
					logger.Warnf("error while on GetEstadoCuenta: %v. Retrying..", err)
					time.Sleep(100 * time.Millisecond)
				} else {
					break
				}
				if retries == 0 {
					errChan <- err
					logger.Errorf("retries exceeded for GetEstadoCuenta: %v", err)
					return
				}
			}
			accountState, err := c.parseEstadoToAccountState(&accStateData, &date)
			if err != nil {
				errChan <- err
				return
			}
			voucherChan <- accountState
		}(i)
	}

	go func() {
		wg.Wait()
		close(errChan)
		close(voucherChan)
	}()

	for accountState := range voucherChan {
		for key, value := range *accountState.Vouchers {
			if v, ok := vouchers[key]; ok {
				v.Holdings = append(v.Holdings, value.Holdings...)
				vouchers[key] = v
			} else {
				vouchers[key] = value
			}
		}
	}
	for _, voucher := range vouchers {
		sortHoldingsByDateRequested(&voucher)
	}
	return &schemas.AccountState{Vouchers: &vouchers}, <-errChan
}

func (c *AccountsController) parseEstadoToAccountState(accStateData *[]esco.EstadoCuentaSchema, date *time.Time) (*schemas.AccountState, error) {
	var categoryKey string
	categoryMap := c.ESCOClient.GetCategoryMap()
	accStateRes := schemas.NewAccountState()
	for _, accData := range *accStateData {
		var voucher schemas.Voucher
		var exists bool
		var parsedDate *time.Time
		if voucher, exists = (*accStateRes.Vouchers)[accData.A]; !exists {
			categoryKey = fmt.Sprintf("%s - %s", accData.A, accData.D)
			var category string
			var exists bool
			if category, exists = categoryMap[categoryKey]; !exists {
				category = "SIN CATEGORIA"
			}
			(*accStateRes.Vouchers)[accData.A] = schemas.Voucher{
				ID:           accData.A,
				Type:         accData.TI,
				Denomination: accData.D,
				Category:     category,
				Holdings:     make([]schemas.Holding, 0, len(*accStateData)),
			}
			voucher = (*accStateRes.Vouchers)[accData.A]
		}
		if accData.F != "" {
			p, err := time.Parse(utils.ShortSlashDateLayout, accData.F)
			if err != nil {
				return nil, err
			}
			parsedDate = &p
		} else {
			parsedDate = nil
		}
		voucher.Holdings = append(voucher.Holdings, schemas.Holding{
			Currency:      accData.M,
			CurrencySign:  accData.MS,
			Value:         accData.N,
			DateRequested: date,
			Date:          parsedDate,
		})
		(*accStateRes.Vouchers)[accData.A] = voucher
	}

	return accStateRes, nil
}

func (c *AccountsController) GetBoletosDateRange(ctx context.Context, token, id string, startDate, endDate time.Time) (*schemas.AccountState, error) {

	account, err := c.GetAccountByID(ctx, token, id)
	if err != nil {
		return nil, err
	}
	boletos, err := c.ESCOClient.GetBoletos(token, account.ID, account.FI, strconv.Itoa(account.N), "-1", startDate, endDate, false)
	if err != nil {
		return nil, err
	}
	return c.parseBoletosToAccountState(&boletos)
}

func (c *AccountsController) parseBoletosToAccountState(boletos *[]esco.Boleto) (*schemas.AccountState, error) {
	var categoryKey string
	categoryMap := c.ESCOClient.GetCategoryMap()
	accStateRes := schemas.NewAccountState()
	for _, boleto := range *boletos {
		var voucher schemas.Voucher
		var exists bool
		var parsedDate *time.Time
		id := strings.Split(boleto.I, " - ")[0]
		categoryKey = boleto.I
		if voucher, exists = (*accStateRes.Vouchers)[id]; !exists {
			var category string
			var exists bool
			if category, exists = categoryMap[categoryKey]; !exists {
				category = "SIN CATEGORIA"
			}
			(*accStateRes.Vouchers)[id] = schemas.Voucher{
				ID:           id,
				Type:         boleto.T,
				Denomination: boleto.F,
				Category:     category,
				Transactions: make([]schemas.Transaction, 0, len(*boletos)),
			}
			voucher = (*accStateRes.Vouchers)[id]
		}
		if boleto.F != "" {
			// Parse the date as before
			p, err := time.Parse(utils.ShortSlashDateLayout, boleto.F)
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
			CurrencySign: boleto.NS,
			Value:        boleto.N,
			Date:         parsedDate,
		})
		(*accStateRes.Vouchers)[id] = voucher
	}

	return accStateRes, nil
}

func (c *AccountsController) GetLiquidacionesDateRange(ctx context.Context, token, id string, startDate, endDate time.Time) (*schemas.AccountState, error) {

	account, err := c.GetAccountByID(ctx, token, id)
	if err != nil {
		return nil, err
	}
	liquidaciones, err := c.ESCOClient.GetLiquidaciones(token, account.ID, account.FI, strconv.Itoa(account.N), "-1", startDate, endDate, false)
	if err != nil {
		return nil, err
	}
	return c.parseLiquidacionesToAccountState(&liquidaciones)
}

func (c *AccountsController) parseLiquidacionesToAccountState(liquidaciones *[]esco.Liquidacion) (*schemas.AccountState, error) {
	var categoryKey string
	categoryMap := c.ESCOClient.GetCategoryMap()
	accStateRes := schemas.NewAccountState()
	for _, liquidacion := range *liquidaciones {
		var voucher schemas.Voucher
		var exists bool
		var parsedDate *time.Time
		id := strings.Split(liquidacion.F, " - ")[0]
		categoryKey = fmt.Sprintf("%s / %s", liquidacion.F, id)
		if voucher, exists = (*accStateRes.Vouchers)[id]; !exists {
			var category string
			var exists bool
			if category, exists = categoryMap[categoryKey]; !exists {
				category = "SIN CATEGORIA"
			}
			(*accStateRes.Vouchers)[id] = schemas.Voucher{
				ID:           id,
				Type:         "",
				Denomination: categoryKey,
				Category:     category,
				Transactions: make([]schemas.Transaction, 0, len(*liquidaciones)),
			}
			voucher = (*accStateRes.Vouchers)[id]
		}
		if liquidacion.FL != "" {
			// Parse the date as before
			p, err := time.Parse(utils.ShortSlashDateLayout, liquidacion.FL)
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
			CurrencySign: liquidacion.MS,
			Value:        liquidacion.I,
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

func (c *AccountsController) GetMultiAccountStateWithTransactionsDateRange(ctx context.Context, token string, ids []string, startDate, endDate time.Time, interval time.Duration) ([]*schemas.AccountState, error) {
	logger := utils.LoggerFromContext(ctx)
	accountsStates := make([]*schemas.AccountState, 0, len(ids))
	var err error

	for _, id := range ids {
		var accountState *schemas.AccountState
		accountState, err = c.GetAccountStateWithTransactionsDateRange(ctx, token, id, startDate, endDate, interval)
		if err != nil {
			logger.Error(err)
			return nil, err
		}
		accountsStates = append(accountsStates, accountState)
	}

	return accountsStates, nil
}

func (c *AccountsController) GetMultiAccountStateByCategoryDateRange(ctx context.Context, token string, ids []string, startDate, endDate time.Time, interval time.Duration) (*schemas.AccountStateByCategory, error) {
	accountStates, err := c.GetMultiAccountStateWithTransactionsDateRange(ctx, token, ids, startDate, endDate, interval)
	if err != nil {
		logger := utils.LoggerFromContext(ctx)
		logger.Error(err)
		return nil, err
	}

	return c.CollapseAndGroupAccountsStates(accountStates), nil
}

func (c *AccountsController) CollapseAndGroupAccountsStates(accountsStates []*schemas.AccountState) *schemas.AccountStateByCategory {
	collapsedAccountState := collapseAccountStates(accountsStates)
	vouchersByCategory := groupAccountStateByCategory(&collapsedAccountState)
	totalsByDate := groupTotalHoldingsAndTransactionsByDate(&collapsedAccountState)
	return &schemas.AccountStateByCategory{
		VouchersByCategory:      vouchersByCategory,
		TotalHoldingsByDate:     totalsByDate.TotalHoldingsByDate,
		TotalTransactionsByDate: totalsByDate.TotalTransactionsByDate,
	}
}

// Group vouchers by category after collapsing, with sorting for consistent ordering
// In addition of calculating the total holding value
func groupAccountStateByCategory(state *schemas.AccountState) *map[string][]schemas.Voucher {
	vouchersByCategory := make(map[string][]schemas.Voucher)

	for _, voucher := range *state.Vouchers {
		category := voucher.Category
		vouchersByCategory[category] = append(vouchersByCategory[category], voucher)
	}

	// Sort each category's vouchers by ID for consistent ordering
	for category := range vouchersByCategory {
		sort.Slice(vouchersByCategory[category], func(i, j int) bool {
			return vouchersByCategory[category][i].ID < vouchersByCategory[category][j].ID
		})
	}

	return &vouchersByCategory
}

func groupTotalHoldingsAndTransactionsByDate(state *schemas.AccountState) *schemas.TotalHoldingsAndTransactionsByDate {
	totalHoldingsByDate := make(map[string]schemas.Holding)
	totalTransactionsByDate := make(map[string]schemas.Transaction)

	for _, voucher := range *state.Vouchers {
		if voucher.Category == "MEP" {
			continue
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
		}
	}

	return &schemas.TotalHoldingsAndTransactionsByDate{
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
