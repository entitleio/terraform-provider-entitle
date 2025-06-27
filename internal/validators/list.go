package validators

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

type MinLengthListValidator struct {
	Min int
}

func (v MinLengthListValidator) Description(_ context.Context) string {
	return fmt.Sprintf("List must have at least %d elements", v.Min)
}

func (v MinLengthListValidator) MarkdownDescription(_ context.Context) string {
	return v.Description(context.Background())
}

func (v MinLengthListValidator) ValidateList(_ context.Context, req validator.ListRequest, resp *validator.ListResponse) {
	if req.ConfigValue.IsUnknown() || req.ConfigValue.IsNull() {
		return
	}

	listVal := req.ConfigValue
	if len(listVal.Elements()) < v.Min {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			"Minimum List Length Not Met",
			fmt.Sprintf("Must have at least %d elements, but got %d.", v.Min, len(listVal.Elements())),
		)
	}
}
