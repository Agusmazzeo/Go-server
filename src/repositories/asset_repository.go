package repositories

import (
	"context"
	"fmt"
	"strings"

	"server/src/models"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AssetRepository interface {
	GetAll(ctx context.Context) ([]models.Asset, error)
	GetByID(ctx context.Context, id int) (*models.Asset, error)
	GetByIDs(ctx context.Context, ids []int) ([]models.Asset, error)
	GetWithCategories(ctx context.Context) ([]models.AssetWithCategory, error)
	Create(ctx context.Context, asset *models.Asset, tx pgx.Tx) error
	CreateBatch(ctx context.Context, assets []models.Asset, tx pgx.Tx) error
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

func (r *assetRepo) GetByIDs(ctx context.Context, ids []int) ([]models.Asset, error) {
	if len(ids) == 0 {
		return []models.Asset{}, nil
	}

	query := `SELECT id, external_id, name, asset_type, category_id, currency FROM assets WHERE id = ANY($1)`
	rows, err := r.db.Query(ctx, query, ids)
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

func (r *assetRepo) GetWithCategories(ctx context.Context) ([]models.AssetWithCategory, error) {
	query := `
		SELECT
			a.id, a.external_id, a.name, a.asset_type, a.category_id, a.currency,
			ac.name as category_name, ac.description as category_description
		FROM assets a
		LEFT JOIN asset_categories ac ON a.category_id = ac.id
		ORDER BY a.external_id`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var assets []models.AssetWithCategory
	for rows.Next() {
		var asset models.AssetWithCategory
		var categoryName, categoryDescription *string
		if err := rows.Scan(&asset.ID, &asset.ExternalID, &asset.Name, &asset.AssetType, &asset.CategoryID, &asset.Currency, &categoryName, &categoryDescription); err != nil {
			return nil, err
		}
		if categoryName != nil {
			asset.CategoryName = *categoryName
		}
		if categoryDescription != nil {
			asset.CategoryDescription = *categoryDescription
		}
		assets = append(assets, asset)
	}
	return assets, rows.Err()
}

func (r *assetRepo) Create(ctx context.Context, asset *models.Asset, tx pgx.Tx) error {
	query := `
		INSERT INTO assets (external_id, name, asset_type, category_id, currency)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (external_id) DO UPDATE SET
			name = EXCLUDED.name,
			asset_type = EXCLUDED.asset_type,
			category_id = EXCLUDED.category_id,
			currency = EXCLUDED.currency
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
			asset.ExternalID, asset.Name, asset.AssetType, asset.CategoryID, asset.Currency,
		).Scan(&asset.ID)

		if err != nil {
			return err
		}

		return tx.Commit(ctx)
	}

	// Use the provided transaction
	return tx.QueryRow(ctx, query,
		asset.ExternalID, asset.Name, asset.AssetType, asset.CategoryID, asset.Currency,
	).Scan(&asset.ID)
}

func (r *assetRepo) CreateBatch(ctx context.Context, assets []models.Asset, tx pgx.Tx) error {
	if len(assets) == 0 {
		return nil
	}

	// Build the batch insert query
	query := `
		INSERT INTO assets (external_id, name, asset_type, category_id, currency)
		VALUES `

	// Build value placeholders and arguments
	args := make([]interface{}, 0, len(assets)*5)
	valueStrings := make([]string, 0, len(assets))

	for i, asset := range assets {
		valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d)",
			i*5+1, i*5+2, i*5+3, i*5+4, i*5+5))
		args = append(args, asset.ExternalID, asset.Name, asset.AssetType, asset.CategoryID, asset.Currency)
	}

	query += strings.Join(valueStrings, ",")
	query += `
		ON CONFLICT (external_id) DO UPDATE SET
			name = EXCLUDED.name,
			asset_type = EXCLUDED.asset_type,
			category_id = EXCLUDED.category_id,
			currency = EXCLUDED.currency`

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

		_, err = tx.Exec(ctx, query, args...)
		if err != nil {
			return err
		}

		return tx.Commit(ctx)
	}

	// Use the provided transaction
	_, err = tx.Exec(ctx, query, args...)
	return err
}
