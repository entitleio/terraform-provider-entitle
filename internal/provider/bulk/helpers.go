package bulk

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
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

// AttrSkipMissingIDs is the name of the skip_missing_ids attribute of bulk resources.
const AttrSkipMissingIDs = "skip_missing_ids"

// SkipMissingIDsAttribute is the skip_missing_ids attribute of bulk resources.
// It only affects updates: removing what the resource added always skips
// entities that no longer exist. Null means false.
func SkipMissingIDsAttribute(entities string) schema.BoolAttribute {
	description := fmt.Sprintf("When `true`, %[1]s that no longer exist in Entitle (for example, deleted by a sync between plan and apply) "+
		"are skipped with a warning, and the change is applied to all other %[1]s. When `false` (default), the apply fails and names them. "+
		"Removing what this resource added always skips %[1]s that no longer exist.", entities)
	return schema.BoolAttribute{
		Optional:            true,
		MarkdownDescription: description,
		Description:         strings.ReplaceAll(description, "`", ""),
	}
}

// AddSkippedWarning warns about wanted IDs that were skipped because the API
// reported them as not found (they were deleted before apply). IDs skipped
// outside wanted (e.g. cleanup of entities removed from config) are expected
// and not reported. The IDs stay in state, which must match the plan; the next
// refresh forgets them.
func AddSkippedWarning(diags *diag.Diagnostics, entities string, wanted, skipped []string) {
	missing := Intersection(wanted, skipped)
	if len(missing) == 0 {
		return
	}
	diags.AddWarning(
		fmt.Sprintf("Skipped %d %s that no longer exist", len(missing), entities),
		fmt.Sprintf("These %s were not found in Entitle (they were probably deleted after the plan) and were skipped; "+
			"every other change was applied. They are dropped from state on the next refresh; "+
			"remove them from the configuration if they are listed there: %s", entities, strings.Join(missing, ", ")),
	)
}
