package models

import "time"

type SyncLog struct {
	ID        int       `db:"id"`
	ClientID  string    `db:"client_id"`
	SyncDate  time.Time `db:"sync_date"`
	CreatedAt time.Time `db:"created_at"`
}
