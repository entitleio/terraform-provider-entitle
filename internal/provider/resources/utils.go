package resources

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/entitleio/terraform-provider-entitle/internal/client"
)

func ConvertTerraformSetToAllowedDurations(ctx context.Context, terraformValue types.Set) (*[]client.EnumAllowedDurations, diag.Diagnostics) {
	// If the value is null, return nil to indicate that the API should use workflow defaults
	if terraformValue.IsNull() {
		return nil, nil
	}

	// If the value is unknown, return empty slice
	if terraformValue.IsUnknown() {
		allowedDurations := make([]client.EnumAllowedDurations, 0)
		return &allowedDurations, nil
	}

	// Convert the terraform set to a slice of allowed durations
	allowedDurations := make([]client.EnumAllowedDurations, 0)
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

	return &allowedDurations, nil
}
