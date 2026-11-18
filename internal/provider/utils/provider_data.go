package utils

import (
	"github.com/entitleio/terraform-provider-entitle/internal/client"
)

const (
	// DefaultBulkChunkSize is the default maximum number of items sent in a single bulk API request.
	DefaultBulkChunkSize int64 = 1000
	// DefaultBulkParallelism is the default maximum number of bulk API requests executed concurrently.
	DefaultBulkParallelism int64 = 4
)

// BulkConfig holds provider-wide limits applied to bulk operations.
type BulkConfig struct {
	// ChunkSize is the maximum number of items sent in a single bulk API request.
	ChunkSize int64
	// Parallelism is the maximum number of bulk API requests executed concurrently.
	Parallelism int64
}

// DefaultBulkConfig returns the bulk configuration used when the provider does not override it.
func DefaultBulkConfig() BulkConfig {
	return BulkConfig{
		ChunkSize:   DefaultBulkChunkSize,
		Parallelism: DefaultBulkParallelism,
	}
}

// ProviderData is passed from the provider's Configure method to every resource and data source.
type ProviderData struct {
	Client *client.ClientWithResponses
	Bulk   BulkConfig
}

// ClientFromProviderData extracts the Entitle API client from the value passed to a
// resource's or data source's Configure method.
func ClientFromProviderData(data any) (*client.ClientWithResponses, bool) {
	pd, ok := data.(*ProviderData)
	if !ok || pd == nil || pd.Client == nil {
		return nil, false
	}

	return pd.Client, true
}

// BulkConfigFromProviderData extracts the bulk configuration from the value passed to a
// resource's or data source's Configure method. Falls back to defaults when unavailable.
func BulkConfigFromProviderData(data any) BulkConfig {
	pd, ok := data.(*ProviderData)
	if !ok || pd == nil {
		return DefaultBulkConfig()
	}

	return pd.Bulk
}
