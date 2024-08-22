package schemas

import "time"

type AccountReponse struct {
	ID   string
	Name string
	CID  string
	FID  string
}

type Holding struct {
	Currency      string
	CurrencySign  string
	Value         float64
	DateRequested *time.Time
	Date          *time.Time
}

type Voucher struct {
	ID          string
	Type        string
	Description string
	Holdings    []Holding
}

type AccountState struct {
	Vouchers *map[string]Voucher
}

func NewAccountState() *AccountState {
	return &AccountState{Vouchers: &map[string]Voucher{}}
}