package repositories

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type SyncLogRepository interface {
	MarkClientForDate(ctx context.Context, clientID string, syncDate time.Time) error
	GetLastSyncDate(ctx context.Context, clientID string) (*time.Time, error)
	MarkClientForDates(ctx context.Context, clientID string, syncDates []time.Time) error
	GetSyncedDates(ctx context.Context, clientID string, startDate time.Time, endDate time.Time) ([]time.Time, error)
	CleanupSyncLogs(ctx context.Context, clientID string, startDate time.Time, endDate time.Time) error
}

type syncLogRepo struct {
	DB *pgxpool.Pool
}

func NewSyncLogRepository(db *pgxpool.Pool) SyncLogRepository {
	return &syncLogRepo{DB: db}
}

func (r *syncLogRepo) MarkClientForDate(ctx context.Context, clientID string, syncDate time.Time) error {
	query := `
		INSERT INTO sync_logs (client_id, sync_date)
		VALUES ($1, $2)
		ON CONFLICT (client_id, sync_date) DO NOTHING`

	var err error

	_, err = r.DB.Exec(ctx, query, clientID, syncDate)
	if err != nil {
		return err
	}
	return nil
}

func (r *syncLogRepo) GetLastSyncDate(ctx context.Context, clientID string) (*time.Time, error) {
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

func (r *syncLogRepo) MarkClientForDates(ctx context.Context, clientID string, syncDates []time.Time) error {
	if len(syncDates) == 0 {
		return nil
	}

	// Start a transaction
	tx, err := r.DB.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Build the query with multiple value pairs and conflict handling
	query := `
		INSERT INTO sync_logs (client_id, sync_date)
		VALUES `

	args := make([]interface{}, 0, len(syncDates)*2)
	valueStrings := make([]string, 0, len(syncDates))

	for i, date := range syncDates {
		valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d)", i*2+1, i*2+2))
		args = append(args, clientID, date)
	}

	query += strings.Join(valueStrings, ",")
	query += " ON CONFLICT (client_id, sync_date) DO NOTHING"

	// Execute the single insert
	_, err = tx.Exec(ctx, query, args...)
	if err != nil {
		return err
	}

	// Commit the transaction
	return tx.Commit(ctx)
}

func (r *syncLogRepo) CleanupSyncLogs(ctx context.Context, clientID string, startDate time.Time, endDate time.Time) error {
	_, err := r.DB.Exec(ctx, `
		DELETE FROM sync_logs
		WHERE client_id = $1
		AND sync_date >= $2
		AND sync_date <= $3
	`, clientID, startDate, endDate)
	if err != nil {
		return err
	}
	return nil
}

func (r *syncLogRepo) GetSyncedDates(ctx context.Context, clientID string, startDate time.Time, endDate time.Time) ([]time.Time, error) {
	rows, err := r.DB.Query(ctx, `
		SELECT sync_date
		FROM sync_logs
		WHERE client_id = $1
		AND sync_date >= $2
		AND sync_date < $3
		ORDER BY sync_date ASC
	`, clientID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dates []time.Time
	for rows.Next() {
		var date time.Time
		if err := rows.Scan(&date); err != nil {
			return nil, err
		}
		dates = append(dates, date)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return dates, nil
}
