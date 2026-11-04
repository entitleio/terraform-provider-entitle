package utils

// Shared filter parts of the list data sources (entitle_resources,
// entitle_roles, entitle_integrations): common attributes, operator values and
// helpers to read filter values.

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/entitleio/terraform-provider-entitle/internal/client"
	"github.com/entitleio/terraform-provider-entitle/internal/validators"
)

// Values accepted by the operator and relation attributes of list filters.
var (
	// Operators match a value list: is (one of them) or isNot (none of them).
	Operators = []string{"is", "isNot"}
	// TextOperators also match by substring: contains or notContains.
	TextOperators = []string{"is", "isNot", "contains", "notContains"}
	// Relations combine several values: and (all of them) or or (any of them).
	Relations = []string{"and", "or"}
	// ContainsOperators match a list attribute (e.g. maintainers) or a name:
	// contains (any of the values) or notContains (none of them).
	ContainsOperators = []string{"contains", "notContains"}
)

// PaginationDeprecation is the deprecation message of the page and per_page
// filter attributes of list data sources that read every page themselves.
const PaginationDeprecation = "Ignored. This data source reads every page and returns all matching items; remove this attribute."

// IDsAttributes returns the ids and ids_operator filter attributes. noun names
// the listed entities in descriptions (e.g. "resources").
func IDsAttributes(noun string) map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"ids": schema.SetAttribute{
			ElementType: types.StringType,
			Optional:    true,
			MarkdownDescription: fmt.Sprintf("Return only the %s with these IDs (UUID), up to %d. "+
				"With `ids_operator = \"isNot\"`, return every %s except them.", noun, client.MaxListIDs, singular(noun)),
			Validators: []validator.Set{
				setvalidator.SizeBetween(1, client.MaxListIDs),
				setvalidator.ValueStringsAre(validators.UUID{}),
			},
		},
		"ids_operator": schema.StringAttribute{
			Optional:            true,
			MarkdownDescription: "How `ids` is matched: `is` returns the given " + noun + ", `isNot` returns every " + singular(noun) + " except them. Defaults to `is`.",
			Validators: []validator.String{
				stringvalidator.OneOf(Operators...),
			},
		},
	}
}

// ExternalIDAttribute returns the external_id filter attribute.
func ExternalIDAttribute(noun string) schema.StringAttribute {
	return schema.StringAttribute{
		Optional:            true,
		MarkdownDescription: "Return only the " + noun + " with this external ID (the ID given by the third-party app).",
	}
}

// UUIDSetAttribute returns an optional, non-empty set of UUIDs.
func UUIDSetAttribute(description string) schema.SetAttribute {
	return schema.SetAttribute{
		ElementType:         types.StringType,
		Optional:            true,
		MarkdownDescription: description,
		Validators: []validator.Set{
			setvalidator.SizeAtLeast(1),
			setvalidator.ValueStringsAre(validators.UUID{}),
		},
	}
}

// DeprecatedPaginationAttributes returns the page and per_page attributes of
// list data sources that used to return one page; they are now ignored.
func DeprecatedPaginationAttributes() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"page": schema.Int64Attribute{
			Optional:            true,
			MarkdownDescription: "**Deprecated.** Ignored; all pages are read.",
			DeprecationMessage:  PaginationDeprecation,
		},
		"per_page": schema.Int64Attribute{
			Optional:            true,
			MarkdownDescription: "**Deprecated.** Ignored; all pages are read.",
			DeprecationMessage:  PaginationDeprecation,
		},
	}
}

func singular(noun string) string {
	if len(noun) > 1 && noun[len(noun)-1] == 's' {
		return noun[:len(noun)-1]
	}
	return noun
}

// SetStrings returns the elements of a known set; a null or unknown set yields nil.
func SetStrings(ctx context.Context, set types.Set, diags *diag.Diagnostics) []string {
	if set.IsNull() || set.IsUnknown() {
		return nil
	}

	var out []string
	diags.Append(set.ElementsAs(ctx, &out, false)...)
	return out
}
