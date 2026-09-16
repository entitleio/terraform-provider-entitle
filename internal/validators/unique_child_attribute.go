package validators

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ validator.List = uniqueChildAttribute{}

type uniqueChildAttribute struct {
	name string
}

// UniqueChildAttribute checks that the given attribute is unique across all
// elements of a list of nested objects.
func UniqueChildAttribute(name string) validator.List {
	return uniqueChildAttribute{name: name}
}

func (v uniqueChildAttribute) Description(_ context.Context) string {
	return fmt.Sprintf("value of %q must be unique across all blocks", v.name)
}

func (v uniqueChildAttribute) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v uniqueChildAttribute) ValidateList(ctx context.Context, req validator.ListRequest, resp *validator.ListResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	seen := make(map[string]path.Path)

	for i, elem := range req.ConfigValue.Elements() {
		obj, ok := elem.(types.Object)
		if !ok || obj.IsNull() || obj.IsUnknown() {
			continue
		}

		val, ok := obj.Attributes()[v.name]
		if !ok || val.IsNull() || val.IsUnknown() {
			// Unknown until apply — nothing to compare yet.
			continue
		}

		p := req.Path.AtListIndex(i).AtName(v.name)
		key := val.String()

		if first, dup := seen[key]; dup {
			resp.Diagnostics.AddAttributeError(
				p,
				"Duplicate value",
				fmt.Sprintf("%s is already used at %s. Each block must have a unique %q.", key, first, v.name),
			)
			continue
		}
		seen[key] = p
	}
}
