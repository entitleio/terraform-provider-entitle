package utils

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/entitleio/terraform-provider-entitle/internal/client"
)

func ConvertTerraformSetToAllowedDurations(ctx context.Context, terraformValue types.Set) ([]client.EnumAllowedDurations, diag.Diagnostics) {
	allowedDurations := make([]client.EnumAllowedDurations, 0)

	if !terraformValue.IsNull() && !terraformValue.IsUnknown() {
		for _, item := range terraformValue.Elements() {
			val, ok := item.(types.Number)
			if !ok {
				continue
			}

			val, diags := val.ToNumberValue(ctx)
			if diags.HasError() {
				return nil, diags
			}

			valFloat32, _ := val.ValueBigFloat().Float32()
			allowedDurations = append(allowedDurations, client.EnumAllowedDurations(valFloat32))
		}
	}

	return allowedDurations, nil
}
