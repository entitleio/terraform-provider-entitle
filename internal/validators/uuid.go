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
