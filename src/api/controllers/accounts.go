package controllers

import (
	"context"
	"fmt"
	"server/src/clients/esco"
	"server/src/schemas"
	"server/src/utils"
	"sort"
	"strconv"
	"sync"
	"time"
)

type AccountsControllerI interface {
	GetAllAccounts(ctx context.Context, token, filter string) ([]*schemas.AccountReponse, error)
	GetAccountByID(ctx context.Context, token, id string) (*esco.CuentaSchema, error)
	GetAccountState(ctx context.Context, token, id string, date time.Time) (*schemas.AccountState, error)
	GetAccountStateDateRange(ctx context.Context, token, id string, startDate, endDate time.Time, interval time.Duration) (*schemas.AccountState, error)
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
	return c.parseToAccountState(&accStateData, &date)
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
			accountState, err := c.parseToAccountState(&accStateData, &date)
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

func (c *AccountsController) parseToAccountState(accStateData *[]esco.EstadoCuentaSchema, date *time.Time) (*schemas.AccountState, error) {
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

func sortHoldingsByDateRequested(voucher *schemas.Voucher) {
	sort.Slice(voucher.Holdings, func(i, j int) bool {
		return voucher.Holdings[i].DateRequested.Before(*voucher.Holdings[j].DateRequested)
	})
}
