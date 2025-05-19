package repositories

import (
	"context"
	"time"

	"server/src/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TransactionRepository interface {
	GetByClientID(ctx context.Context, clientID string) ([]models.Transaction, error)
	Create(ctx context.Context, t *models.Transaction, tx pgx.Tx) error
}

type transactionRepo struct {
	db *pgxpool.Pool
}

func NewTransactionRepository(db *pgxpool.Pool) TransactionRepository {
	return &transactionRepo{db: db}
}

func (r *transactionRepo) GetByClientID(ctx context.Context, clientID string) ([]models.Transaction, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, client_id, asset_id, transaction_type, units, price_per_unit, total_value, date FROM transactions WHERE client_id = $1`,
		clientID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions []models.Transaction
	for rows.Next() {
		var t models.Transaction
		var date time.Time
		if err := rows.Scan(&t.ID, &t.ClientID, &t.AssetID, &t.TransactionType, &t.Units, &t.PricePerUnit, &t.TotalValue, &date); err != nil {
			return nil, err
		}
		t.Date = date
		transactions = append(transactions, t)
	}
	return transactions, rows.Err()
}

func (r *transactionRepo) Create(ctx context.Context, t *models.Transaction, tx pgx.Tx) error {
	query := `
		INSERT INTO transactions (client_id, asset_id, transaction_type, units, price_per_unit, total_value, date)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
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
			t.ClientID, t.AssetID, t.TransactionType, t.Units, t.PricePerUnit, t.TotalValue, t.Date,
		).Scan(&t.ID)

		if err != nil {
			return err
		}

		return tx.Commit(ctx)
	}

	// Use the provided transaction
	return tx.QueryRow(ctx, query,
		t.ClientID, t.AssetID, t.TransactionType, t.Units, t.PricePerUnit, t.TotalValue, t.Date,
	).Scan(&t.ID)
}
