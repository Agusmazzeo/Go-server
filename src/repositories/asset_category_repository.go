package repositories

import (
	"context"

	"server/src/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AssetCategoryRepository interface {
	GetAll(ctx context.Context) ([]models.AssetCategory, error)
	GetByID(ctx context.Context, id int) (*models.AssetCategory, error)
	GetByName(ctx context.Context, name string) (*models.AssetCategory, error)

	Create(ctx context.Context, ac *models.AssetCategory, tx pgx.Tx) error
}

type assetCategoryRepo struct {
	db *pgxpool.Pool
}

func NewAssetCategoryRepository(db *pgxpool.Pool) AssetCategoryRepository {
	return &assetCategoryRepo{db: db}
}

func (r *assetCategoryRepo) GetAll(ctx context.Context) ([]models.AssetCategory, error) {
	rows, err := r.db.Query(ctx, `SELECT id, name, description FROM asset_categories`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var categories []models.AssetCategory
	for rows.Next() {
		var ac models.AssetCategory
		if err := rows.Scan(&ac.ID, &ac.Name, &ac.Description); err != nil {
			return nil, err
		}
		categories = append(categories, ac)
	}

	return categories, rows.Err()
}

func (r *assetCategoryRepo) GetByID(ctx context.Context, id int) (*models.AssetCategory, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, name, description FROM asset_categories WHERE id = $1`, id,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, nil
	}

	var ac models.AssetCategory
	if err := rows.Scan(&ac.ID, &ac.Name, &ac.Description); err != nil {
		return nil, err
	}

	return &ac, nil
}

func (r *assetCategoryRepo) GetByName(ctx context.Context, name string) (*models.AssetCategory, error) {
	rows, err := r.db.Query(ctx,
		`SELECT id, name, description FROM asset_categories WHERE name = $1`, name,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, nil
	}

	var ac models.AssetCategory
	if err := rows.Scan(&ac.ID, &ac.Name, &ac.Description); err != nil {
		return nil, err
	}

	return &ac, nil
}

func (r *assetCategoryRepo) Create(ctx context.Context, ac *models.AssetCategory, tx pgx.Tx) error {
	query := `
		INSERT INTO asset_categories (name, description)
		VALUES ($1, $2)
		ON CONFLICT (name) DO UPDATE SET description = EXCLUDED.description
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

		err = tx.QueryRow(ctx, query, ac.Name, ac.Description).Scan(&ac.ID)
		if err != nil {
			return err
		}

		return tx.Commit(ctx)
	}

	// Use the provided transaction
	return tx.QueryRow(ctx, query, ac.Name, ac.Description).Scan(&ac.ID)
}
