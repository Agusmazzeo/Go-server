package controllers

import (
	"context"
	"server/src/schemas"
	"strconv"
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

func (c *Controller) GetAccountValues(ctx context.Context, filter string) ([]*schemas.AccountReponse, error) {
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
