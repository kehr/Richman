package model

import "time"

// Asset represents an entry in the asset catalog.
type Asset struct {
	AssetCatalogID int64     `json:"assetCatalogId"`
	Code           string    `json:"code"`
	Name           string    `json:"name"`
	NameEn         string    `json:"nameEn"`
	AssetType      string    `json:"assetType"`
	Exchange       string    `json:"exchange"`
	DataSource     string    `json:"dataSource"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}
