package repositories

import (
	"context"

	"server/src/models"

	"github.com/jackc/pgx/v5/pgxpool"
)

type AssetRepository interface {
	GetAll(ctx context.Context) ([]models.Asset, error)
	GetByID(ctx context.Context, id int) (*models.Asset, error)
	Create(ctx context.Context, asset *models.Asset) error
}

type assetRepo struct {
	db *pgxpool.Pool
}

func NewAssetRepository(db *pgxpool.Pool) AssetRepository {
	return &assetRepo{db: db}
}

func (r *assetRepo) GetAll(ctx context.Context) ([]models.Asset, error) {
	rows, err := r.db.Query(ctx, `SELECT id, external_id, name, asset_type, category_id, currency FROM assets`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assets []models.Asset
	for rows.Next() {
		var asset models.Asset
		if err := rows.Scan(&asset.ID, &asset.ExternalID, &asset.Name, &asset.AssetType, &asset.CategoryID, &asset.Currency); err != nil {
			return nil, err
		}
		assets = append(assets, asset)
	}
	return assets, rows.Err()
}

func (r *assetRepo) GetByID(ctx context.Context, id int) (*models.Asset, error) {
	var asset models.Asset
	err := r.db.QueryRow(ctx, `SELECT id, external_id, name, asset_type, category_id, currency FROM assets WHERE id = $1`, id).
		Scan(&asset.ID, &asset.ExternalID, &asset.Name, &asset.AssetType, &asset.CategoryID, &asset.Currency)
	if err != nil {
		return nil, err
	}
	return &asset, nil
}

func (r *assetRepo) Create(ctx context.Context, asset *models.Asset) error {
	err := r.db.QueryRow(ctx,
		`INSERT INTO assets (external_id, name, asset_type, category_id, currency)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (external_id) DO NOTHING
		 RETURNING id`,
		asset.ExternalID, asset.Name, asset.AssetType, asset.CategoryID, asset.Currency,
	).Scan(&asset.ID)
	return err
}
