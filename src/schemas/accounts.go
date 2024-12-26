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

type Transaction struct {
	Currency     string
	CurrencySign string
	Value        float64
	Date         *time.Time
}

type Voucher struct {
	ID           string
	Type         string
	Denomination string
	Category     string
	Holdings     []Holding
	Transactions []Transaction
}

type AccountState struct {
	Vouchers *map[string]Voucher
}

type TotalHoldingsAndTransactionsByDate struct {
	TotalHoldingsByDate     *map[string]Holding
	TotalTransactionsByDate *map[string]Transaction
}

type AccountStateByCategory struct {
	VouchersByCategory      *map[string][]Voucher
	TotalHoldingsByDate     *map[string]Holding
	TotalTransactionsByDate *map[string]Transaction
}

func NewAccountState() *AccountState {
	return &AccountState{Vouchers: &map[string]Voucher{}}
}
