package repositories

import (
	"context"
	"time"

	"server/src/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type TransactionRepository interface {
	GetByClientID(ctx context.Context, clientID string, startDate, endDate time.Time) ([]models.Transaction, error)
	GetByClientIDs(ctx context.Context, clientIDs []string, startDate, endDate time.Time) ([]models.Transaction, error)
	GetGroupedByCategoryAndDate(ctx context.Context, clientIDs []string, startDate, endDate time.Time) (map[string]map[string]float64, error)
	GetTotalByDate(ctx context.Context, clientIDs []string, startDate, endDate time.Time) (map[string]float64, error)
	Create(ctx context.Context, t *models.Transaction, tx pgx.Tx) error
}

type transactionRepo struct {
	db *pgxpool.Pool
}

func NewTransactionRepository(db *pgxpool.Pool) TransactionRepository {
	return &transactionRepo{db: db}
}

func (r *transactionRepo) GetByClientID(ctx context.Context, clientID string, startDate, endDate time.Time) ([]models.Transaction, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, client_id, asset_id, transaction_type, units, price_per_unit, total_value, date, created_at, deleted, deleted_at
		FROM transactions
		WHERE client_id = $1 AND date BETWEEN $2 AND $3
		ORDER BY date DESC`,
		clientID, startDate, endDate,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions []models.Transaction
	for rows.Next() {
		var t models.Transaction
		var date, createdAt time.Time
		var deletedAt *time.Time
		if err := rows.Scan(&t.ID, &t.ClientID, &t.AssetID, &t.TransactionType, &t.Units, &t.PricePerUnit, &t.TotalValue, &date, &createdAt, &t.Deleted, &deletedAt); err != nil {
			return nil, err
		}
		t.Date = date
		t.CreatedAt = createdAt
		t.DeletedAt = deletedAt
		transactions = append(transactions, t)
	}
	return transactions, rows.Err()
}

func (r *transactionRepo) GetByClientIDs(ctx context.Context, clientIDs []string, startDate, endDate time.Time) ([]models.Transaction, error) {
	if len(clientIDs) == 0 {
		return []models.Transaction{}, nil
	}

	query := `SELECT t.id, t.client_id, t.asset_id, t.transaction_type, t.units, t.price_per_unit, t.total_value, t.date, t.created_at, t.deleted, t.deleted_at
		FROM transactions t
		WHERE t.client_id = ANY($1) AND t.date BETWEEN $2 AND $3
		ORDER BY t.date DESC`

	rows, err := r.db.Query(ctx, query, clientIDs, startDate, endDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var transactions []models.Transaction
	for rows.Next() {
		var t models.Transaction
		var date, createdAt time.Time
		var deletedAt *time.Time
		if err := rows.Scan(&t.ID, &t.ClientID, &t.AssetID, &t.TransactionType, &t.Units, &t.PricePerUnit, &t.TotalValue, &date, &createdAt, &t.Deleted, &deletedAt); err != nil {
			return nil, err
		}
		t.Date = date
		t.CreatedAt = createdAt
		t.DeletedAt = deletedAt
		transactions = append(transactions, t)
	}
	return transactions, rows.Err()
}

func (r *transactionRepo) GetGroupedByCategoryAndDate(ctx context.Context, clientIDs []string, startDate, endDate time.Time) (map[string]map[string]float64, error) {
	if len(clientIDs) == 0 {
		return make(map[string]map[string]float64), nil
	}

	query := `
		SELECT
			ac.name as category,
			DATE(t.date) as date,
			SUM(t.total_value) as total_value
		FROM transactions t
		JOIN assets a ON t.asset_id = a.id
		JOIN asset_categories ac ON a.category_id = ac.id
		WHERE t.client_id = ANY($1) AND t.date BETWEEN $2 AND $3
		GROUP BY ac.name, DATE(t.date)
		ORDER BY ac.name, DATE(t.date)`

	rows, err := r.db.Query(ctx, query, clientIDs, startDate, endDate)
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

func (r *transactionRepo) GetTotalByDate(ctx context.Context, clientIDs []string, startDate, endDate time.Time) (map[string]float64, error) {
	if len(clientIDs) == 0 {
		return make(map[string]float64), nil
	}

	query := `
		SELECT
			DATE(t.date) as date,
			SUM(t.total_value) as total_value
		FROM transactions t
		WHERE t.client_id = ANY($1) AND t.date BETWEEN $2 AND $3
		GROUP BY DATE(t.date)
		ORDER BY DATE(t.date)`

	rows, err := r.db.Query(ctx, query, clientIDs, startDate, endDate)
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
