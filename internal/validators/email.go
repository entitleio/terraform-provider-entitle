package validators

import (
	"context"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var _ validator.String = &Email{}
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

// Email validator.String for email parser.
type Email struct{}

// Description satisfies the validator.String interface.
func (u Email) Description(ctx context.Context) string {
	return "validating the resource email is properly formatted"
}

// MarkdownDescription satisfies the validator.String interface.
func (u Email) MarkdownDescription(ctx context.Context) string {
	return "validating the resource email is properly formatted"
}

// ValidateString Validate satisfies the validator.String interface.
func (u Email) ValidateString(ctx context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsUnknown() {
		// skip validation when the value is not known yet
		return
	}

	if !emailRegex.MatchString(req.ConfigValue.ValueString()) {
		resp.Diagnostics.AddError(
			"Email Validate failed",
			"Failed to parse Email for resource, invalid format",
		)
	}
}
