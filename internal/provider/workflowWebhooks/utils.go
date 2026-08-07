package workflowWebhooks

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// getHeadersAsPlanned converts the Terraform map of headers into the representation
// expected by the Entitle API.
//
// The API rejects a null headers value ("headers must be an object"), so an unset or
// unknown map is sent as an empty object rather than omitted. Sending an object also
// makes it possible to clear previously configured headers on update.
func getHeadersAsPlanned(ctx context.Context, headers types.Map) (*map[string]string, diag.Diagnostics) {
	var diags diag.Diagnostics

	result := make(map[string]string, len(headers.Elements()))

	if headers.IsNull() || headers.IsUnknown() {
		return &result, diags
	}

	diags.Append(headers.ElementsAs(ctx, &result, false)...)
	if diags.HasError() {
		return nil, diags
	}

	return &result, diags
}

// getHeaders converts the headers returned by the Entitle API into a Terraform map
// value. The API omits headers entirely when none are configured, which maps to a
// null value.
func getHeaders(ctx context.Context, headers *map[string]string) (types.Map, diag.Diagnostics) {
	var diags diag.Diagnostics

	if headers == nil || len(*headers) == 0 {
		return types.MapNull(types.StringType), diags
	}

	return types.MapValueFrom(ctx, types.StringType, *headers)
}
