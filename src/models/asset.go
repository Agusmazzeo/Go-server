package models

import "time"

type Asset struct {
	ID         int        `db:"id"`
	ExternalID string     `db:"external_id"`
	Name       string     `db:"name"`
	AssetType  string     `db:"asset_type"`
	CategoryID int        `db:"category_id"`
	Currency   string     `db:"currency"`
	CreatedAt  time.Time  `db:"created_at"`
	Deleted    bool       `db:"deleted"`
	DeletedAt  *time.Time `db:"deleted_at"`
}

type AssetWithCategory struct {
	ID                  int        `db:"id"`
	ExternalID          string     `db:"external_id"`
	Name                string     `db:"name"`
	AssetType           string     `db:"asset_type"`
	CategoryID          int        `db:"category_id"`
	Currency            string     `db:"currency"`
	CategoryName        string     `db:"category_name"`
	CategoryDescription string     `db:"category_description"`
	CreatedAt           time.Time  `db:"created_at"`
	Deleted             bool       `db:"deleted"`
	DeletedAt           *time.Time `db:"deleted_at"`
}
