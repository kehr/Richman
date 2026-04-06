package repo

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/richman/backend/internal/model"
)

// AssetRepo handles asset catalog data access operations.
type AssetRepo struct {
	pool *pgxpool.Pool
}

// NewAssetRepo creates a new AssetRepo.
func NewAssetRepo(pool *pgxpool.Pool) *AssetRepo {
	return &AssetRepo{pool: pool}
}

// ListAssets returns assets filtered by optional asset type and keyword.
func (r *AssetRepo) ListAssets(ctx context.Context, assetType, keyword string) ([]model.Asset, error) {
	query := `SELECT asset_catalog_id, code, name, name_en, asset_type, exchange, data_source, created_at, updated_at
		FROM asset_catalog
		WHERE is_deleted = 0`
	args := []interface{}{}
	argIdx := 1

	if assetType != "" {
		query += fmt.Sprintf(" AND asset_type = $%d", argIdx)
		args = append(args, assetType)
		argIdx++
	}

	if keyword != "" {
		query += fmt.Sprintf(" AND (code ILIKE $%d OR name ILIKE $%d OR name_en ILIKE $%d)", argIdx, argIdx, argIdx)
		args = append(args, "%"+keyword+"%")
		_ = argIdx // reserved for future filters
	}

	query += " ORDER BY asset_type, code"

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query assets: %w", err)
	}
	defer rows.Close()

	var assets []model.Asset
	for rows.Next() {
		var a model.Asset
		if err := rows.Scan(
			&a.AssetCatalogID, &a.Code, &a.Name, &a.NameEn,
			&a.AssetType, &a.Exchange, &a.DataSource,
			&a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan asset: %w", err)
		}
		assets = append(assets, a)
	}
	return assets, nil
}

// GetAssetByCode returns a single asset by its code. Returns nil if not found.
func (r *AssetRepo) GetAssetByCode(ctx context.Context, code string) (*model.Asset, error) {
	var a model.Asset
	err := r.pool.QueryRow(ctx,
		`SELECT asset_catalog_id, code, name, name_en, asset_type, exchange, data_source, created_at, updated_at
		 FROM asset_catalog
		 WHERE code = $1 AND is_deleted = 0`,
		code,
	).Scan(
		&a.AssetCatalogID, &a.Code, &a.Name, &a.NameEn,
		&a.AssetType, &a.Exchange, &a.DataSource,
		&a.CreatedAt, &a.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query asset by code: %w", err)
	}
	return &a, nil
}
