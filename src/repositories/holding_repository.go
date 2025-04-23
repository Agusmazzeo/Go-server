package repositories

import (
	"context"
	"time"

	"server/src/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type HoldingRepository interface {
	GetByClientID(ctx context.Context, clientID string) ([]models.Holding, error)
	Create(ctx context.Context, h *models.Holding) error
}

type holdingRepo struct {
	db *pgxpool.Pool
}

func NewHoldingRepository(db *pgxpool.Pool) HoldingRepository {
	return &holdingRepo{db: db}
}

func (r *holdingRepo) GetByClientID(ctx context.Context, clientID string) ([]models.Holding, error) {
	rows, err := r.db.Query(ctx, `SELECT id, client_id, asset_id, quantity, value, date FROM holdings WHERE client_id = $1`, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var holdings []models.Holding
	for rows.Next() {
		var h models.Holding
		var date time.Time
		if err := rows.Scan(&h.ID, &h.ClientID, &h.AssetID, &h.Quantity, &h.Value, &date); err != nil {
			return nil, err
		}
		h.Date = date
		holdings = append(holdings, h)
	}
	return holdings, rows.Err()
}

func (r *holdingRepo) Create(ctx context.Context, h *models.Holding) error {
	err := r.db.QueryRow(ctx,
		`INSERT INTO holdings (client_id, asset_id, quantity, value, date)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id`,
		h.ClientID, h.AssetID, h.Quantity, h.Value, h.Date,
	).Scan(&h.ID)
	return err
}
