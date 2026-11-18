package bulk

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/entitleio/terraform-provider-entitle/internal/client"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/utils"
)

// NewExecutorFromProviderData builds an Executor from the value passed to a
// resource's Configure method. It returns (nil, nil) when providerData is nil
// (the provider is not configured yet).
func NewExecutorFromProviderData(providerData any) (*Executor, *client.ClientWithResponses, error) {
	if providerData == nil {
		return nil, nil, nil
	}

	c, ok := utils.ClientFromProviderData(providerData)
	if !ok {
		return nil, nil, fmt.Errorf("expected *utils.ProviderData, got: %T. Please report this issue to the provider developers", providerData)
	}

	return NewExecutor(c, utils.BulkConfigFromProviderData(providerData)), c, nil
}

// ResourcesUpdate returns a SubmitFunc that applies update to each chunk via
// PATCH /public/v1/resources/bulk.
func ResourcesUpdate(c *client.ClientWithResponses, update client.BulkResourceUpdate) SubmitFunc {
	return func(ctx context.Context, ids []string) (*client.BulkSubmitResponse, error) {
		return c.ResourcesBulkUpdateWithResponse(ctx, client.BulkUpdateResourcesRequest{
			Ids:    ids,
			Update: update,
		})
	}
}

// StringSet converts a known types.Set of strings into a slice. Null/unknown sets yield nil.
func StringSet(ctx context.Context, set types.Set) ([]string, diag.Diagnostics) {
	if set.IsNull() || set.IsUnknown() {
		return nil, nil
	}

	var out []string
	diags := set.ElementsAs(ctx, &out, false)
	return out, diags
}

// ToStringSet converts a slice into a types.Set of strings (empty, not null, for no items).
func ToStringSet(ctx context.Context, items []string) (types.Set, diag.Diagnostics) {
	if items == nil {
		items = []string{}
	}
	return types.SetValueFrom(ctx, types.StringType, items)
}
