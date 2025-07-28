package models

import (
	"time"
)

type Transaction struct {
	ID              int        `db:"id"`
	ClientID        string     `db:"client_id"`
	AssetID         int        `db:"asset_id"`
	TransactionType string     `db:"transaction_type"`
	Units           float64    `db:"units"`
	PricePerUnit    float64    `db:"price_per_unit"`
	TotalValue      float64    `db:"total_value"`
	Date            time.Time  `db:"date"`
	CreatedAt       time.Time  `db:"created_at"`
	Deleted         bool       `db:"deleted"`
	DeletedAt       *time.Time `db:"deleted_at"`
}
