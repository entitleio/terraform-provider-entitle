package validators

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// NewSetMinLength creates a new set length validator.
func NewSetMinLength(minLength int) SetMinLength {
	return SetMinLength{
		minLength: minLength,
	}
}

type SetMinLength struct {
	minLength int
}

// Description satisfies the validator.Set interface.
func (u SetMinLength) Description(ctx context.Context) string {
	return fmt.Sprintf("Validating the set length - minimum %d", u.minLength)
}

// MarkdownDescription satisfies the validator.Set interface.
func (u SetMinLength) MarkdownDescription(ctx context.Context) string {
	return fmt.Sprintf("Validating the set length - minimum %d", u.minLength)
}

// ValidateSet Validate satisfies the validator.Set interface.
func (u SetMinLength) ValidateSet(ctx context.Context, req validator.SetRequest, resp *validator.SetResponse) {
	// Skip validation if value is empty (not provided) or not known yet
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}

	if len(req.ConfigValue.Elements()) < u.minLength {
		resp.Diagnostics.AddError(
			"Set length Validate Failed",
			fmt.Sprintf("Incorrect %s set length - minimum %d, got %d", req.Path, u.minLength, len(req.ConfigValue.Elements())),
		)
	}
}

// NewListMinLength creates a new set length validator.
func NewListMinLength(minLength int) ListMinLength {
	return ListMinLength{
		minLength: minLength,
	}
}

type ListMinLength struct {
	minLength int
}

// Description satisfies the validator.List interface.
func (u ListMinLength) Description(ctx context.Context) string {
	return fmt.Sprintf("Validating the set length - minimum %d", u.minLength)
}

// MarkdownDescription satisfies the validator.List interface.
func (u ListMinLength) MarkdownDescription(ctx context.Context) string {
	return fmt.Sprintf("Validating the set length - minimum %d", u.minLength)
}

// ValidateList Validate satisfies the validator.List interface.
func (u ListMinLength) ValidateList(ctx context.Context, req validator.ListRequest, resp *validator.ListResponse) {
	// Skip validation if value is empty (not provided) or not known yet
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}

	if len(req.ConfigValue.Elements()) < u.minLength {
		resp.Diagnostics.AddError(
			"List length Validate Failed",
			fmt.Sprintf("Incorrect %s list length - minimum %d, got %d", req.Path, u.minLength, len(req.ConfigValue.Elements())),
		)
	}
}
