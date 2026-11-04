package utils

// Shared filter parts of the list data sources (entitle_resources,
// entitle_roles, entitle_integrations): the ids, ids_operator, search and
// external_id attributes, operator values, and the entitle_integrations filter.

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
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
	// ContainsOperators match a list attribute (e.g. maintainers): contains (any
	// of the values) or notContains (none of them).
	ContainsOperators = []string{"contains", "notContains"}
)

// PaginationDeprecation is the deprecation message of the page and per_page
// filter attributes of list data sources that read every page themselves.
const PaginationDeprecation = "Ignored. This data source reads every page and returns all matching items; remove this attribute."

// IDsFilterModel is the filter block of entitle_integrations (no search / external ID).
type IDsFilterModel struct {
	Page        types.Int64  `tfsdk:"page"`
	PerPage     types.Int64  `tfsdk:"per_page"`
	IDs         types.Set    `tfsdk:"ids"`
	IDsOperator types.String `tfsdk:"ids_operator"`
	AfterID     types.String `tfsdk:"after_id"`
}

// ListFilterAttributes returns the filter attributes. noun names the listed
// entities in descriptions (e.g. "resources"); withSearch adds search and external_id.
func ListFilterAttributes(noun string, withSearch bool) map[string]schema.Attribute {
	afterIDLimits := "Cannot be combined with a `page` other than 1."
	if withSearch {
		afterIDLimits = "Cannot be combined with a `page` other than 1 or with `search`."
	}

	attrs := map[string]schema.Attribute{
		"page": schema.Int64Attribute{
			Optional:            true,
			MarkdownDescription: "Page number of results to return (starting from 1). Used together with `per_page` for pagination.",
			Validators:          []validator.Int64{int64validator.AtLeast(1)},
		},
		"per_page": schema.Int64Attribute{
			Optional:            true,
			MarkdownDescription: "Number of results to return per page (at most 100). Defaults to the API's configured page size if not specified.",
			Validators:          []validator.Int64{int64validator.Between(1, client.MaxListPerPage)},
		},
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
		"after_id": schema.StringAttribute{
			Optional: true,
			MarkdownDescription: "Cursor for reading the whole list without skipping or repeating items while it changes. " +
				"Results are sorted by ID; pass the ID of the last item of the previous page to get the items after it. " +
				afterIDLimits,
			Validators: []validator.String{validators.UUID{}},
		},
	}

	if withSearch {
		attrs["search"] = schema.StringAttribute{
			Optional:            true,
			MarkdownDescription: "Search string to filter " + noun + ". Cannot be combined with `ids` unless `external_id` is also set.",
		}
		attrs["external_id"] = schema.StringAttribute{
			Optional:            true,
			MarkdownDescription: "External ID to filter " + noun + ".",
		}
	}
	return attrs
}

func singular(noun string) string {
	if len(noun) > 1 && noun[len(noun)-1] == 's' {
		return noun[:len(noun)-1]
	}
	return noun
}

// HasIDs reports whether a known, non-empty ids list is configured.
func (m *IDsFilterModel) HasIDs() bool { return m != nil && hasElements(m.IDs) }

func hasElements(l types.Set) bool {
	return !l.IsNull() && (l.IsUnknown() || len(l.Elements()) > 0)
}

// Query converts the filter into list query parameters. A nil filter yields an empty query.
func (m *IDsFilterModel) Query(ctx context.Context) (client.ListQuery, diag.Diagnostics) {
	if m == nil {
		return client.ListQuery{}, nil
	}
	q, diags := idsQuery(ctx, m.IDs, m.IDsOperator, m.AfterID, m.Page, m.PerPage)
	diags.Append(validateListQuery(q)...)
	return q, diags
}

func idsQuery(ctx context.Context, ids types.Set, op, afterID types.String, page, perPage types.Int64) (client.ListQuery, diag.Diagnostics) {
	var diags diag.Diagnostics
	var q client.ListQuery

	if !ids.IsNull() && !ids.IsUnknown() {
		diags.Append(ids.ElementsAs(ctx, &q.IDs, false)...)
	}
	q.IDsOperator = client.IDsOperator(op.ValueString())
	q.AfterID = afterID.ValueString()
	q.Page = int(page.ValueInt64())
	q.PerPage = int(perPage.ValueInt64())
	return q, diags
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

// validateListQuery reports parameter combinations the list endpoints reject.
func validateListQuery(q client.ListQuery) diag.Diagnostics {
	var diags diag.Diagnostics
	filter := path.Root("filter")

	if q.AfterID != "" && q.Page > 1 {
		diags.AddAttributeError(filter.AtName("after_id"), "Invalid Filter",
			"after_id cannot be combined with a page other than 1.")
	}
	if q.AfterID != "" && q.Search != "" {
		diags.AddAttributeError(filter.AtName("after_id"), "Invalid Filter",
			"after_id cannot be combined with search.")
	}
	if len(q.IDs) > 0 && q.Search != "" && q.ExternalID == "" {
		diags.AddAttributeError(filter.AtName("ids"), "Invalid Filter",
			"ids cannot be combined with search unless external_id is also set.")
	}
	return diags
}
