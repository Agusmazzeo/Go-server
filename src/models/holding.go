package models

import (
	"time"
)

type Holding struct {
	ID        int        `db:"id"`
	ClientID  string     `db:"client_id"`
	AssetID   int        `db:"asset_id"`
	Quantity  float64    `db:"quantity"`
	Value     float64    `db:"value"`
	Date      time.Time  `db:"date"`
	CreatedAt time.Time  `db:"created_at"`
	Deleted   bool       `db:"deleted"`
	DeletedAt *time.Time `db:"deleted_at"`
}
