package validators

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var _ validator.String = &UUID{}

// UUID validator.String for uuid parser.
type UUID struct{}

// Description satisfies the validator.String interface.
func (u UUID) Description(ctx context.Context) string {
	return "validating the resource id is uuid formatted"
}

// MarkdownDescription satisfies the validator.String interface.
func (u UUID) MarkdownDescription(ctx context.Context) string {
	return "validating the resource id is uuid formatted"
}

// ValidateString Validate satisfies the validator.String interface.
func (u UUID) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	_, err := uuid.Parse(req.ConfigValue.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"UUID Validate failed",
			fmt.Sprintf("failed to parse UUID for resource, error: %v", err),
		)
	}
}

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
	val := req.ConfigValue.ValueString()
	if len(val) < 2 || len(val) > 50 {
		resp.Diagnostics.AddError(
			"Workflow Name Validate Failed",
			fmt.Sprintf("validating the resource id is workflow name length between 2-50, name: (%s)", val),
		)
	}
}
