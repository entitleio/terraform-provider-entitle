package utils

// Shared filter for the list data sources (entitle_resources, entitle_roles,
// entitle_integrations): pagination, search/external ID, and the ids,
// ids_operator and after_id parameters of the list endpoints.

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/listvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/entitleio/terraform-provider-entitle/internal/client"
	"github.com/entitleio/terraform-provider-entitle/internal/validators"
)

// ListFilterModel is the filter block of entitle_resources and entitle_roles.
type ListFilterModel struct {
	Search      types.String `tfsdk:"search"`
	ExternalID  types.String `tfsdk:"external_id"`
	Page        types.Int64  `tfsdk:"page"`
	PerPage     types.Int64  `tfsdk:"per_page"`
	IDs         types.List   `tfsdk:"ids"`
	IDsOperator types.String `tfsdk:"ids_operator"`
	AfterID     types.String `tfsdk:"after_id"`
}

// IDsFilterModel is the filter block of entitle_integrations (no search / external ID).
type IDsFilterModel struct {
	Page        types.Int64  `tfsdk:"page"`
	PerPage     types.Int64  `tfsdk:"per_page"`
	IDs         types.List   `tfsdk:"ids"`
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
		"ids": schema.ListAttribute{
			ElementType: types.StringType,
			Optional:    true,
			MarkdownDescription: fmt.Sprintf("Return only the %s with these IDs (UUID), up to %d. "+
				"With `ids_operator = \"isNot\"`, return every %s except them.", noun, client.MaxListIDs, singular(noun)),
			Validators: []validator.List{
				listvalidator.SizeBetween(1, client.MaxListIDs),
				listvalidator.ValueStringsAre(validators.UUID{}),
			},
		},
		"ids_operator": schema.StringAttribute{
			Optional:            true,
			MarkdownDescription: "How `ids` is matched: `is` (default) returns the given " + noun + ", `isNot` returns every " + singular(noun) + " except them. Requires `ids`.",
			Validators: []validator.String{
				stringvalidator.OneOf(string(client.IDsOperatorIs), string(client.IDsOperatorIsNot)),
				stringvalidator.AlsoRequires(path.MatchRelative().AtParent().AtName("ids")),
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
func (m *ListFilterModel) HasIDs() bool { return m != nil && hasElements(m.IDs) }

// HasIDs reports whether a known, non-empty ids list is configured.
func (m *IDsFilterModel) HasIDs() bool { return m != nil && hasElements(m.IDs) }

func hasElements(l types.List) bool {
	return !l.IsNull() && (l.IsUnknown() || len(l.Elements()) > 0)
}

// Query converts the filter into list query parameters and validates the
// combinations the API rejects. A nil filter yields an empty query.
func (m *ListFilterModel) Query(ctx context.Context) (client.ListQuery, diag.Diagnostics) {
	if m == nil {
		return client.ListQuery{}, nil
	}
	q, diags := idsQuery(ctx, m.IDs, m.IDsOperator, m.AfterID, m.Page, m.PerPage)
	q.Search = m.Search.ValueString()
	q.ExternalID = m.ExternalID.ValueString()
	diags.Append(validateListQuery(q)...)
	return q, diags
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

func idsQuery(ctx context.Context, ids types.List, op, afterID types.String, page, perPage types.Int64) (client.ListQuery, diag.Diagnostics) {
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
