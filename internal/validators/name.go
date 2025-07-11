package validators

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// NewName creates new name validator.
func NewName(minLength, maxLength int) Name {
	return Name{
		minLength: minLength,
		maxLength: maxLength,
	}
}

type Name struct {
	minLength, maxLength int
}

// Description satisfies the validator.String interface.
func (u Name) Description(ctx context.Context) string {
	return fmt.Sprintf("validating the name length between %d-%d", u.minLength, u.maxLength)
}

// MarkdownDescription satisfies the validator.String interface.
func (u Name) MarkdownDescription(ctx context.Context) string {
	return fmt.Sprintf("validating the name length between %d-%d", u.minLength, u.maxLength)
}

// ValidateString Validate satisfies the validator.String interface.
func (u Name) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsUnknown() {
		// skip validation when the value is not known yet
		return
	}

	val := req.ConfigValue.ValueString()
	if len(val) < u.minLength || len(val) > u.maxLength {
		resp.Diagnostics.AddError(
			"Name Validate Failed",
			fmt.Sprintf("validating the name length between %d-%d, name: (%s)", u.minLength, u.maxLength, val),
		)
	}
}
