package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/entitleio/terraform-provider-entitle/internal/provider/utils"
)

func TestResolveBulkConfig(t *testing.T) {
	env := func(vals map[string]string) func(string) string {
		return func(k string) string { return vals[k] }
	}

	tests := []struct {
		name    string
		model   *BulkProviderModel
		env     map[string]string
		want    utils.BulkConfig
		wantErr bool
	}{
		{
			name: "no block, no env uses defaults",
			want: utils.DefaultBulkConfig(),
		},
		{
			name:  "empty block uses defaults",
			model: &BulkProviderModel{ChunkSize: types.Int64Null(), Parallelism: types.Int64Null()},
			want:  utils.DefaultBulkConfig(),
		},
		{
			name:  "config values",
			model: &BulkProviderModel{ChunkSize: types.Int64Value(500), Parallelism: types.Int64Value(8)},
			want:  utils.BulkConfig{ChunkSize: 500, Parallelism: 8},
		},
		{
			name: "env values",
			env:  map[string]string{envBulkChunkSize: "250", envBulkParallelism: "2"},
			want: utils.BulkConfig{ChunkSize: 250, Parallelism: 2},
		},
		{
			name:  "config overrides env",
			model: &BulkProviderModel{ChunkSize: types.Int64Value(100), Parallelism: types.Int64Null()},
			env:   map[string]string{envBulkChunkSize: "250", envBulkParallelism: "2"},
			want:  utils.BulkConfig{ChunkSize: 100, Parallelism: 2},
		},
		{
			name:    "invalid env",
			env:     map[string]string{envBulkChunkSize: "abc"},
			wantErr: true,
		},
		{
			name:    "non-positive env",
			env:     map[string]string{envBulkParallelism: "0"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, diags := resolveBulkConfig(tt.model, env(tt.env))
			if diags.HasError() != tt.wantErr {
				t.Fatalf("unexpected diagnostics: %v", diags)
			}
			if tt.wantErr {
				return
			}
			if got != tt.want {
				t.Fatalf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}
