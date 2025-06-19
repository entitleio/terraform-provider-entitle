package validators

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// NewName creates new name validator
func NewName(min, max int) Name {
	return Name{
		min: min,
		max: max,
	}
}

type Name struct {
	min, max int
}

// Description satisfies the validator.String interface.
func (u Name) Description(ctx context.Context) string {
	return fmt.Sprintf("validating the name length between %d-%d", u.min, u.max)
}

// MarkdownDescription satisfies the validator.String interface.
func (u Name) MarkdownDescription(ctx context.Context) string {
	return fmt.Sprintf("validating the name length between %d-%d", u.min, u.max)
}

// ValidateString Validate satisfies the validator.String interface.
func (u Name) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	val := req.ConfigValue.ValueString()
	if len(val) < u.min || len(val) > u.max {
		resp.Diagnostics.AddError(
			"Name Validate Failed",
			fmt.Sprintf("validating the name length between %d-%d, name: (%s)", u.min, u.max, val),
		)
	}
}
