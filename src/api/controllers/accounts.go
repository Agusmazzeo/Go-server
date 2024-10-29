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
	accStateData, err := c.ESCOClient.GetEstadoCuenta(token, account.ID, account.FI, strconv.Itoa(account.N), "-1", date)
	if err != nil {
		return nil, err
	}
	return c.parseEstadoToAccountState(&accStateData, &date)
}

func (c *AccountsController) GetAccountStateWithTransactionsDateRange(ctx context.Context, token, id string, startDate, endDate time.Time, interval time.Duration) (*schemas.AccountState, error) {
	var err error
	var wg sync.WaitGroup
	wg.Add(3)

	var accountState *schemas.AccountState
	var liquidaciones *schemas.AccountState
	var boletos *schemas.AccountState
	go func() {
		accountState, err = c.GetAccountStateDateRange(ctx, token, id, startDate, endDate, interval)
		wg.Done()
	}()
	go func() {
		liquidaciones, err = c.GetLiquidacionesDateRange(ctx, token, id, startDate, endDate)
		wg.Done()
	}()
	go func() {
		boletos, err = c.GetBoletosDateRange(ctx, token, id, startDate, endDate)
		wg.Done()
	}()
	wg.Wait()
	if err != nil {
		return nil, err
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
	account, err := c.GetAccountByID(ctx, token, id)
	if err != nil {
		return nil, err
	}
	intervalHours := interval.Hours()
	// Calculate the number of days between startDate and endDate
	numDays := int(endDate.Sub(startDate).Hours()/intervalHours) + 1
	vouchers := make(map[string]schemas.Voucher)

	var wg sync.WaitGroup
	var voucherChan = make(chan *schemas.AccountState, numDays)
	for i := 0; i < numDays; i++ {
		wg.Add(1)
		go func(i int) {
			date := startDate.AddDate(0, 0, i*int(intervalHours/24))
			accStateData, err := c.ESCOClient.GetEstadoCuenta(token, account.ID, account.FI, strconv.Itoa(account.N), "-1", date)
			if err != nil {
				wg.Done()
				return
			}
			accountState, err := c.parseEstadoToAccountState(&accStateData, &date)
			if err != nil {
				wg.Done()
				return
			}
			voucherChan <- accountState
		}(i)
	}

	go func() {
		wg.Wait()
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
		wg.Done()
	}
	for _, voucher := range vouchers {
		sortHoldingsByDateRequested(&voucher)
	}
	return &schemas.AccountState{Vouchers: &vouchers}, nil
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
	boletos, err := c.ESCOClient.GetBoletos(token, account.ID, account.FI, strconv.Itoa(account.N), "-1", startDate, endDate)
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
	liquidaciones, err := c.ESCOClient.GetLiquidaciones(token, account.ID, account.FI, strconv.Itoa(account.N), "-1", startDate, endDate)
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
	accountsStates := make([]*schemas.AccountState, 0, len(ids))
	accountsStatesChan := make(chan *schemas.AccountState, len(ids))
	errChan := make(chan error, 1) // Create a buffered error channel to handle at most one error
	var wg sync.WaitGroup

	// Launch goroutines for each account ID
	for _, id := range ids {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			accountState, err := c.GetAccountStateWithTransactionsDateRange(ctx, token, id, startDate, endDate, interval)
			if err != nil {
				errChan <- err
				return
			}
			accountsStatesChan <- accountState
		}(id)
	}

	// Close the channels when all goroutines are done
	go func() {
		wg.Wait()
		close(accountsStatesChan)
		close(errChan)
	}()

	// Listen on the channels
	breakLoop := false
	for {
		select {
		case accountState, ok := <-accountsStatesChan:
			if ok {
				// Append account state to the result slice
				accountsStates = append(accountsStates, accountState)
			} else {
				// accountsStatesChan is closed, meaning wg is done
				breakLoop = true
			}
		case err, ok := <-errChan:
			if ok {
				// Return on the first error encountered
				return nil, err
			}
		}
		if breakLoop {
			break
		}
	}
	return accountsStates, nil
}

func (c *AccountsController) GetMultiAccountStateByCategoryDateRange(ctx context.Context, token string, ids []string, startDate, endDate time.Time, interval time.Duration) (*schemas.AccountStateByCategory, error) {
	accountStates, err := c.GetMultiAccountStateWithTransactionsDateRange(ctx, token, ids, startDate, endDate, interval)
	if err != nil {
		return nil, err
	}

	collapsedAccountState := collapseAccountStates(accountStates)

	return groupAccountStateByCategory(&collapsedAccountState), nil
}

func groupAccountStateByCategory(state *schemas.AccountState) *schemas.AccountStateByCategory {
	vouchersByCategory := make(map[string][]schemas.Voucher)

	for _, voucher := range *state.Vouchers {
		category := voucher.Category
		// Group vouchers by their category
		vouchersByCategory[category] = append(vouchersByCategory[category], voucher)
	}

	return &schemas.AccountStateByCategory{
		VouchersByCategory: &vouchersByCategory,
	}
}

func collapseAccountStates(states []*schemas.AccountState) schemas.AccountState {
	collapsed := make(map[string]schemas.Voucher)

	for _, state := range states {
		if state.Vouchers == nil {
			continue
		}
		for voucherID, voucher := range *state.Vouchers {
			if _, exists := collapsed[voucherID]; !exists {
				collapsed[voucherID] = schemas.Voucher{
					ID:           voucher.ID,
					Type:         voucher.Type,
					Denomination: voucher.Denomination,
					Category:     voucher.Category,
					Holdings:     []schemas.Holding{},
					Transactions: []schemas.Transaction{},
				}
			}
			collapsedVoucher := collapsed[voucherID]

			// Collapsing Holdings
			holdingMap := make(map[string]*schemas.Holding)
			for i := range collapsedVoucher.Holdings {
				holding := &collapsedVoucher.Holdings[i]
				key := holding.Currency + holding.DateRequested.Format("2006-01-02")
				holdingMap[key] = holding
			}
			for _, holding := range voucher.Holdings {
				key := holding.Currency + holding.DateRequested.Format("2006-01-02")
				if existing, found := holdingMap[key]; found {
					existing.Value += holding.Value
				} else {
					collapsedVoucher.Holdings = append(collapsedVoucher.Holdings, holding)
					holdingMap[key] = &collapsedVoucher.Holdings[len(collapsedVoucher.Holdings)-1]
				}
			}

			// Collapsing Transactions
			transactionMap := make(map[string]*schemas.Transaction)
			for i := range collapsedVoucher.Transactions {
				transaction := &collapsedVoucher.Transactions[i]
				key := transaction.Currency + transaction.Date.Format("2006-01-02")
				transactionMap[key] = transaction
			}
			for _, transaction := range voucher.Transactions {
				key := transaction.Currency + transaction.Date.Format("2006-01-02")
				if existing, found := transactionMap[key]; found {
					existing.Value += transaction.Value
				} else {
					collapsedVoucher.Transactions = append(collapsedVoucher.Transactions, transaction)
					transactionMap[key] = &collapsedVoucher.Transactions[len(collapsedVoucher.Transactions)-1]
				}
			}

			// Update the collapsed voucher in the map
			collapsed[voucherID] = collapsedVoucher
		}
	}

	return schemas.AccountState{Vouchers: &collapsed}
}
