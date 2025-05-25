package repositories

import (
	"context"
	"time"

	"server/src/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type HoldingRepository interface {
	GetByClientID(ctx context.Context, clientID string) ([]models.Holding, error)
	Create(ctx context.Context, h *models.Holding, tx pgx.Tx) error
}

type holdingRepo struct {
	db *pgxpool.Pool
}

func NewHoldingRepository(db *pgxpool.Pool) HoldingRepository {
	return &holdingRepo{db: db}
}

func (r *holdingRepo) GetByClientID(ctx context.Context, clientID string) ([]models.Holding, error) {
	rows, err := r.db.Query(ctx, `SELECT id, client_id, asset_id, units, value, date FROM holdings WHERE client_id = $1 order by date desc`, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var holdings []models.Holding
	for rows.Next() {
		var h models.Holding
		var date time.Time
		if err := rows.Scan(&h.ID, &h.ClientID, &h.AssetID, &h.Units, &h.Value, &date); err != nil {
			return nil, err
		}
		h.Date = date
		holdings = append(holdings, h)
	}
	return holdings, rows.Err()
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
