package schemas

import (
	"fmt"
	"time"
)

// Date represents a date in YYYY-MM-DD format
type Date struct {
	time.Time
}

// ToTime returns the underlying time.Time value
func (d Date) ToTime() time.Time {
	return d.Time
}

// UnmarshalJSON implements json.Unmarshaler interface
func (d *Date) UnmarshalJSON(data []byte) error {
	// Remove quotes from the JSON string
	str := string(data)
	if len(str) >= 2 && str[0] == '"' && str[len(str)-1] == '"' {
		str = str[1 : len(str)-1]
	}

	// Parse the date in YYYY-MM-DD format
	parsed, err := time.Parse("2006-01-02", str)
	if err != nil {
		return fmt.Errorf("invalid date format, expected YYYY-MM-DD: %v", err)
	}

	d.Time = parsed
	return nil
}

// MarshalJSON implements json.Marshaler interface
func (d Date) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, d.Format("2006-01-02"))), nil
}

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
	AccountID string `json:"accountID"`
	StartDate Date   `json:"startDate"`
	EndDate   Date   `json:"endDate"`
}

func NewAccountState() *AccountState {
	return &AccountState{Assets: &map[string]Asset{}}
}
