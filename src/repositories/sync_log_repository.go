package repositories

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type SyncLogRepository struct {
	DB *pgxpool.Pool
}

func NewSyncLogRepository(db *pgxpool.Pool) *SyncLogRepository {
	return &SyncLogRepository{DB: db}
}

func (r *SyncLogRepository) Insert(ctx context.Context, clientID string, syncDate time.Time) error {
	_, err := r.DB.Exec(ctx, `
		INSERT INTO sync_logs (client_id, sync_date)
		VALUES ($1, $2)
	`, clientID, syncDate)
	return err
}

func (r *SyncLogRepository) GetLastSyncDate(ctx context.Context, clientID string) (*time.Time, error) {
	var syncDate time.Time
	err := r.DB.QueryRow(ctx, `
		SELECT sync_date
		FROM sync_logs
		WHERE client_id = $1
		ORDER BY sync_date DESC
		LIMIT 1
	`, clientID).Scan(&syncDate)
	if err != nil {
		if err.Error() == "no rows in result set" {
			return nil, nil
		}
		return nil, err
	}
	return &syncDate, nil
}
