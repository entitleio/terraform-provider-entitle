package validators

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// NewLowercase creates a new validator that ensures a string is all lowercase.
func NewLowercase() Lowercase {
	return Lowercase{}
}

type Lowercase struct{}

// Description satisfies the validator.String interface.
func (v Lowercase) Description(ctx context.Context) string {
	return "must be lowercase"
}

// MarkdownDescription satisfies the validator.String interface.
func (v Lowercase) MarkdownDescription(ctx context.Context) string {
	return "must be lowercase"
}

// ValidateString ensures the string is all lowercase.
func (v Lowercase) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	// Skip validation if the value is empty (not provided) or not known yet
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}

	val := req.ConfigValue.ValueString()
	if val != strings.ToLower(val) {
		resp.Diagnostics.AddError(
			"Lowercase Validation Failed",
			fmt.Sprintf("value must be all lowercase, got: (%s)", val),
		)
	}
}
