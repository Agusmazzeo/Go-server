package repositories

import (
	"context"

	"server/src/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AssetCategoryRepository interface {
	GetAll(ctx context.Context) ([]models.AssetCategory, error)
	GetByID(ctx context.Context, id int) (*models.AssetCategory, error)
	Create(ctx context.Context, ac *models.AssetCategory) error
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
	var ac models.AssetCategory
	err := r.db.QueryRow(ctx,
		`SELECT id, name, description FROM asset_categories WHERE id = $1`, id,
	).Scan(&ac.ID, &ac.Name, &ac.Description)

	if err != nil {
		return nil, err
	}
	return &ac, nil
}

func (r *assetCategoryRepo) Create(ctx context.Context, ac *models.AssetCategory) error {
	err := r.db.QueryRow(ctx,
		`INSERT INTO asset_categories (name, description) VALUES ($1, $2) ON CONFLICT (name) DO NOTHING RETURNING id`,
		ac.Name, ac.Description,
	).Scan(&ac.ID)
	if err != nil {
		return err
	}
	return nil
}
