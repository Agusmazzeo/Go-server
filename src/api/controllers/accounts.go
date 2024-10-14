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
		if boleto, ok := (*boletos.Vouchers)[id]; ok {
			voucher.Transactions = append(voucher.Transactions, boleto.Transactions...)
			(*accountState.Vouchers)[id] = voucher
		}
		if liquidacion, ok := (*liquidaciones.Vouchers)[id]; ok {
			voucher.Transactions = append(voucher.Transactions, liquidacion.Transactions...)
			(*accountState.Vouchers)[id] = voucher
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
			(*accStateRes.Vouchers)[accData.A] = schemas.Voucher{
				ID:           accData.A,
				Type:         accData.TI,
				Denomination: accData.D,
				Category:     categoryMap[categoryKey],
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
			(*accStateRes.Vouchers)[id] = schemas.Voucher{
				ID:           id,
				Type:         boleto.T,
				Denomination: boleto.F,
				Category:     categoryMap[categoryKey],
				Transactions: make([]schemas.Transaction, 0, len(*boletos)),
			}
			voucher = (*accStateRes.Vouchers)[id]
		}
		if boleto.F != "" {
			p, err := time.Parse(utils.ShortSlashDateLayout, boleto.F)
			if err != nil {
				return nil, err
			}
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
			(*accStateRes.Vouchers)[id] = schemas.Voucher{
				ID:           id,
				Type:         "",
				Denomination: categoryKey,
				Category:     categoryMap[categoryKey],
				Transactions: make([]schemas.Transaction, 0, len(*liquidaciones)),
			}
			voucher = (*accStateRes.Vouchers)[id]
		}
		if liquidacion.FL != "" {
			p, err := time.Parse(utils.ShortSlashDateLayout, liquidacion.FL)
			if err != nil {
				return nil, err
			}
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
