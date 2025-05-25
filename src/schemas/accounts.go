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
	Units         float64
	DateRequested *time.Time
	Date          *time.Time
}

type Transaction struct {
	Currency     string
	CurrencySign string
	Value        float64
	Units        float64
	Date         *time.Time
}

type Asset struct {
	ID           string
	Type         string
	Denomination string
	Category     string
	Holdings     []Holding
	Transactions []Transaction
}

type AccountState struct {
	Assets *map[string]Asset
}

type TotalHoldingsAndTransactionsByDate struct {
	TotalHoldingsByDate     *map[string]Holding
	TotalTransactionsByDate *map[string]Transaction
}

type AccountStateByCategory struct {
	AssetsByCategory        *map[string][]Asset
	CategoryAssets          *map[string]Asset
	TotalHoldingsByDate     *map[string]Holding
	TotalTransactionsByDate *map[string]Transaction
}

// SyncRequest represents a request to sync account data
type SyncRequest struct {
	AccountID string    `json:"account_id"`
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
}

func NewAccountState() *AccountState {
	return &AccountState{Assets: &map[string]Asset{}}
}
