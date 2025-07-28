package models

import "time"

type AssetCategory struct {
	ID          int        `db:"id"`
	Name        string     `db:"name"`
	Description string     `db:"description"`
	CreatedAt   time.Time  `db:"created_at"`
	Deleted     bool       `db:"deleted"`
	DeletedAt   *time.Time `db:"deleted_at"`
}
