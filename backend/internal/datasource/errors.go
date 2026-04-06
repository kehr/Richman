package datasource

import "errors"

var (
	// ErrDataSourceUnavailable indicates the upstream data source is unreachable.
	ErrDataSourceUnavailable = errors.New("data source unavailable")

	// ErrInvalidResponse indicates the upstream returned an unparseable response.
	ErrInvalidResponse = errors.New("invalid response from data source")

	// ErrUnsupportedAssetType indicates the asset type is not supported.
	ErrUnsupportedAssetType = errors.New("unsupported asset type")
)
