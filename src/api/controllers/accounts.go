package controllers

import (
	"context"
	"fmt"
	"server/src/clients/esco"
	"server/src/schemas"
	"server/src/utils"
	"strconv"
	"sync"
	"time"
)

func (c *Controller) GetAllAccounts(ctx context.Context, filter string) ([]*schemas.AccountReponse, error) {
	if filter == "" {
		filter = "*"
	}
	accs, err := c.ESCOClient.BuscarCuentas(filter)
	if err != nil {
		return nil, err
	}
	accounts := make([]*schemas.AccountReponse, len(accs))
	for i, account := range accs {
		accounts[i] = &schemas.AccountReponse{ID: strconv.Itoa(account.N), CID: account.ID, FID: account.FI, Name: account.D}
	}
	return accounts, nil
}

func (c *Controller) GetAccountByID(ctx context.Context, id string) (*esco.CuentaSchema, error) {
	acc, err := c.ESCOClient.BuscarCuentas(id)
	if err != nil {
		return nil, err
	}
	if len(acc) != 1 {
		return nil, fmt.Errorf("more than 1 account received for given id %s", id)
	}
	return &acc[0], nil
}

func (c *Controller) GetAccountState(ctx context.Context, id string, date time.Time) (*schemas.AccountState, error) {

	account, err := c.GetAccountByID(ctx, id)
	if err != nil {
		return nil, err
	}
	accStateData, err := c.ESCOClient.GetEstadoCuenta(account.ID, account.FI, strconv.Itoa(account.N), date)
	if err != nil {
		return nil, err
	}
	return parseToAccountState(&accStateData)
}

func (c *Controller) GetAccountStateDateRange(ctx context.Context, id string, startDate, endDate time.Time) (*schemas.AccountState, error) {
	account, err := c.GetAccountByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Calculate the number of days between startDate and endDate
	numDays := int(endDate.Sub(startDate).Hours() / 24)
	vouchers := make(map[string]schemas.Voucher)

	var wg sync.WaitGroup
	var lock sync.Mutex
	for i := 0; i < numDays; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			date := startDate.AddDate(0, 0, i)
			accStateData, err := c.ESCOClient.GetEstadoCuenta(account.ID, account.FI, strconv.Itoa(account.N), date)
			if err != nil {
				return
			}
			accountState, err := parseToAccountState(&accStateData)
			if err != nil {
				return
			}
			lock.Lock()
			for key, value := range *accountState.Vouchers {
				if v, ok := vouchers[key]; ok {
					v.Holdings = append(v.Holdings, value.Holdings...)
					vouchers[key] = v
				} else {
					vouchers[key] = value
				}
			}
			lock.Unlock()
		}(i)
	}

	wg.Wait()

	return &schemas.AccountState{Vouchers: &vouchers}, nil
}

func parseToAccountState(accStateData *[]esco.EstadoCuentaSchema) (*schemas.AccountState, error) {
	accStateRes := schemas.NewAccountState()
	for _, accData := range *accStateData {
		var voucher schemas.Voucher
		var exists bool
		if voucher, exists = (*accStateRes.Vouchers)[accData.A]; !exists {
			(*accStateRes.Vouchers)[accData.A] = schemas.Voucher{
				ID:          accData.A,
				Type:        accData.TI,
				Description: accData.D,
				Holdings:    make([]schemas.Holding, 0, len(*accStateData)),
			}
			voucher = (*accStateRes.Vouchers)[accData.A]
		}
		date, err := time.Parse(utils.ShortSlashDateLayout, accData.F)
		if err != nil {
			return nil, err
		}
		voucher.Holdings = append(voucher.Holdings, schemas.Holding{
			Currency:     accData.M,
			CurrencySign: accData.MS,
			Value:        accData.N,
			Date:         date,
		})
		(*accStateRes.Vouchers)[accData.A] = voucher
	}

	return accStateRes, nil
}
