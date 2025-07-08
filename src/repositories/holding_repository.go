package repositories

import (
	"context"
	"time"

	"server/src/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type HoldingRepository interface {
	GetByClientID(ctx context.Context, clientID string, startDate, endDate time.Time) ([]models.Holding, error)
	GetByClientIDs(ctx context.Context, clientIDs []string, startDate, endDate time.Time) ([]models.Holding, error)
	GetGroupedByCategoryAndDate(ctx context.Context, clientIDs []string, startDate, endDate time.Time) (map[string]map[string]float64, error)
	GetTotalByDate(ctx context.Context, clientIDs []string, startDate, endDate time.Time) (map[string]float64, error)
	Create(ctx context.Context, h *models.Holding, tx pgx.Tx) error
}

type holdingRepo struct {
	db *pgxpool.Pool
}

func NewHoldingRepository(db *pgxpool.Pool) HoldingRepository {
	return &holdingRepo{db: db}
}

func (r *holdingRepo) GetByClientID(ctx context.Context, clientID string, startDate, endDate time.Time) ([]models.Holding, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, client_id, asset_id, units, value, date, created_at, deleted, deleted_at
		FROM holdings
		WHERE client_id = $1 AND date BETWEEN $2 AND $3
		ORDER BY date DESC`,
		clientID, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var holdings []models.Holding
	for rows.Next() {
		var h models.Holding
		var date, createdAt time.Time
		var deletedAt *time.Time
		if err := rows.Scan(&h.ID, &h.ClientID, &h.AssetID, &h.Units, &h.Value, &date, &createdAt, &h.Deleted, &deletedAt); err != nil {
			return nil, err
		}
		h.Date = date
		h.CreatedAt = createdAt
		h.DeletedAt = deletedAt
		holdings = append(holdings, h)
	}
	return holdings, rows.Err()
}

func (r *holdingRepo) GetByClientIDs(ctx context.Context, clientIDs []string, startDate, endDate time.Time) ([]models.Holding, error) {
	if len(clientIDs) == 0 {
		return []models.Holding{}, nil
	}

	// Build the query with proper placeholders
	query := `SELECT h.id, h.client_id, h.asset_id, h.units, h.value, h.date, h.created_at, h.deleted, h.deleted_at
		FROM holdings h
		WHERE h.client_id = ANY($1) AND h.date BETWEEN $2 AND $3
		ORDER BY h.date DESC`

	rows, err := r.db.Query(ctx, query, clientIDs, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var holdings []models.Holding
	for rows.Next() {
		var h models.Holding
		var date, createdAt time.Time
		var deletedAt *time.Time
		if err := rows.Scan(&h.ID, &h.ClientID, &h.AssetID, &h.Units, &h.Value, &date, &createdAt, &h.Deleted, &deletedAt); err != nil {
			return nil, err
		}
		h.Date = date
		h.CreatedAt = createdAt
		h.DeletedAt = deletedAt
		holdings = append(holdings, h)
	}
	return holdings, rows.Err()
}

func (r *holdingRepo) GetGroupedByCategoryAndDate(ctx context.Context, clientIDs []string, startDate, endDate time.Time) (map[string]map[string]float64, error) {
	if len(clientIDs) == 0 {
		return make(map[string]map[string]float64), nil
	}

	query := `
		SELECT
			ac.name as category,
			DATE(h.date) as date,
			SUM(h.value) as total_value
		FROM holdings h
		JOIN assets a ON h.asset_id = a.id
		JOIN asset_categories ac ON a.category_id = ac.id
		WHERE h.client_id = ANY($1) AND h.date BETWEEN $2 AND $3
		GROUP BY ac.name, DATE(h.date)
		ORDER BY ac.name, DATE(h.date)`

	rows, err := r.db.Query(ctx, query, clientIDs, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]map[string]float64)
	for rows.Next() {
		var category string
		var date time.Time
		var totalValue float64
		if err := rows.Scan(&category, &date, &totalValue); err != nil {
			return nil, err
		}

		dateStr := date.Format("2006-01-02")
		if result[category] == nil {
			result[category] = make(map[string]float64)
		}
		result[category][dateStr] = totalValue
	}
	return result, rows.Err()
}

func (r *holdingRepo) GetTotalByDate(ctx context.Context, clientIDs []string, startDate, endDate time.Time) (map[string]float64, error) {
	if len(clientIDs) == 0 {
		return make(map[string]float64), nil
	}

	query := `
		SELECT
			DATE(h.date) as date,
			SUM(h.value) as total_value
		FROM holdings h
		WHERE h.client_id = ANY($1) AND h.date BETWEEN $2 AND $3
		GROUP BY DATE(h.date)
		ORDER BY DATE(h.date)`

	rows, err := r.db.Query(ctx, query, clientIDs, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string]float64)
	for rows.Next() {
		var date time.Time
		var totalValue float64
		if err := rows.Scan(&date, &totalValue); err != nil {
			return nil, err
		}

		dateStr := date.Format("2006-01-02")
		result[dateStr] = totalValue
	}
	return result, rows.Err()
}

func (r *holdingRepo) Create(ctx context.Context, h *models.Holding, tx pgx.Tx) error {
	query := `
		INSERT INTO holdings (client_id, asset_id, units, value, date)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (client_id, asset_id, date) DO UPDATE SET
			units = EXCLUDED.units,
			value = EXCLUDED.value
		RETURNING id`

	var err error
	if tx == nil {
		// If no transaction is provided, create a new one
		tx, err = r.db.Begin(ctx)
		if err != nil {
			return err
		}
		defer func() {
			if err != nil {
				_ = tx.Rollback(ctx)
			}
		}()

		err = tx.QueryRow(ctx, query,
			h.ClientID, h.AssetID, h.Units, h.Value, h.Date,
		).Scan(&h.ID)

		if err != nil {
			return err
		}

		return tx.Commit(ctx)
	}

	// Use the provided transaction
	return tx.QueryRow(ctx, query,
		h.ClientID, h.AssetID, h.Units, h.Value, h.Date,
	).Scan(&h.ID)
}
