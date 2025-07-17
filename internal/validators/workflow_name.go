package validators

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

type WorkflowName struct{}

// Description satisfies the validator.String interface.
func (u WorkflowName) Description(ctx context.Context) string {
	return "validating the resource id is workflow name length between 2-50"
}

// MarkdownDescription satisfies the validator.String interface.
func (u WorkflowName) MarkdownDescription(ctx context.Context) string {
	return "validating the resource id is workflow name length between 2-50"
}

// ValidateString Validate satisfies the validator.String interface.
func (u WorkflowName) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		// skip validation when the value is not known yet
		return
	}

	val := req.ConfigValue.ValueString()
	if len(val) < 2 || len(val) > 50 {
		resp.Diagnostics.AddError(
			"Workflow Name Validate Failed",
			fmt.Sprintf("validating the resource id is workflow name length between 2-50, name: (%s)", val),
		)
	}
}
