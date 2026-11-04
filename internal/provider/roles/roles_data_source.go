package roles

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/entitleio/terraform-provider-entitle/docs"
	"github.com/entitleio/terraform-provider-entitle/internal/client"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/utils"
	"github.com/entitleio/terraform-provider-entitle/internal/validators"
)

// Ensure the data source satisfies the framework interface.
var _ datasource.DataSource = &RolesDataSource{}

// RolesDataSource defines the implementation of the list data source.
type RolesDataSource struct {
	client *client.ClientWithResponses
}

// NewRolesDataSource creates a new instance of RolesDataSource.
func NewRolesDataSource() datasource.DataSource {
	return &RolesDataSource{}
}

// RolesDataSourceModel describes the data source model.
type RolesDataSourceModel struct {
	// Deprecated: use filter.resource_id and filter.integration_id.
	ResourceID    types.String      `tfsdk:"resource_id"`
	IntegrationID types.String      `tfsdk:"integration_id"`
	Filter        *RolesFilterModel `tfsdk:"filter"`
	Roles         []RoleListItem    `tfsdk:"roles"`
}

// RolesFilterModel is the filter block of entitle_roles. Pagination is handled
// internally: the data source always returns every matching role.
type RolesFilterModel struct {
	ResourceID            types.String `tfsdk:"resource_id"`
	IntegrationID         types.String `tfsdk:"integration_id"`
	Search                types.String `tfsdk:"search"`
	ExternalID            types.String `tfsdk:"external_id"`
	IDs                   types.Set    `tfsdk:"ids"`
	IDsOperator           types.String `tfsdk:"ids_operator"`
	Requestable           types.Bool   `tfsdk:"requestable"`
	WorkflowIDs           types.Set    `tfsdk:"workflow_ids"`
	WorkflowIDsOperator   types.String `tfsdk:"workflow_ids_operator"`
	ResourceTypes         types.Set    `tfsdk:"resource_types"`
	ResourceTypesOperator types.String `tfsdk:"resource_types_operator"`
	ResourceTags          types.Set    `tfsdk:"resource_tags"`
	ResourceTagsOperator  types.String `tfsdk:"resource_tags_operator"`
	ResourceTagsRelation  types.String `tfsdk:"resource_tags_relation"`

	// Deprecated: ignored, every page is read.
	Page    types.Int64 `tfsdk:"page"`
	PerPage types.Int64 `tfsdk:"per_page"`
}

// RoleListItem represents a single role in the list.
type RoleListItem struct {
	ID          types.String       `tfsdk:"id"`
	ExternalID  types.String       `tfsdk:"external_id"`
	Name        types.String       `tfsdk:"name"`
	Requestable types.Bool         `tfsdk:"requestable"`
	ResourceID  types.String       `tfsdk:"resource_id"`
	Workflow    *utils.IdNameModel `tfsdk:"workflow"`
}

// Metadata sets the metadata for the data source.
func (d *RolesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_roles"
}

func rolesFilterAttributes() map[string]schema.Attribute {
	shared := utils.ListFilterAttributes("roles", true)

	return map[string]schema.Attribute{
		"resource_id": schema.StringAttribute{
			Optional: true,
			MarkdownDescription: "Return the roles of this resource. One of `resource_id`, `integration_id` or `ids` is required; " +
				"`resource_id` takes precedence over `integration_id`.",
			Validators: []validator.String{
				validators.UUID{},
				stringvalidator.ConflictsWith(path.MatchRoot("resource_id")),
			},
		},
		"integration_id": schema.StringAttribute{
			Optional: true,
			MarkdownDescription: "Return every role across all resources of this integration. " +
				"One of `resource_id`, `integration_id` or `ids` is required.",
			Validators: []validator.String{
				validators.UUID{},
				stringvalidator.ConflictsWith(path.MatchRoot("integration_id")),
			},
		},
		"ids":          shared["ids"],
		"ids_operator": shared["ids_operator"],
		"external_id":  shared["external_id"],
		"search": schema.StringAttribute{
			Optional: true,
			MarkdownDescription: "Return only roles whose name contains this term (case-insensitive). Exact matches are ranked first, " +
				"followed by case-insensitive matches and then partial matches. Can be combined with every other filter.",
		},
		"requestable": schema.BoolAttribute{
			Optional:            true,
			MarkdownDescription: "Return only roles that are (`true`) or are not (`false`) requestable.",
		},
		"workflow_ids": schema.SetAttribute{
			ElementType:         types.StringType,
			Optional:            true,
			MarkdownDescription: "Return only roles with one of these workflow IDs (UUID) (see `workflow_ids_operator`).",
			Validators: []validator.Set{
				setvalidator.SizeAtLeast(1),
				setvalidator.ValueStringsAre(validators.UUID{}),
			},
		},
		"workflow_ids_operator": schema.StringAttribute{
			Optional:            true,
			MarkdownDescription: "How `workflow_ids` is matched: `is` returns roles with one of the given workflows, `isNot` returns roles with none of them. Defaults to `is`.",
			Validators: []validator.String{
				stringvalidator.OneOf(utils.Operators...),
			},
		},
		"resource_types": schema.SetAttribute{
			ElementType:         types.StringType,
			Optional:            true,
			MarkdownDescription: "Return only roles whose resource type matches these values (see `resource_types_operator`).",
			Validators:          []validator.Set{setvalidator.SizeAtLeast(1)},
		},
		"resource_types_operator": schema.StringAttribute{
			Optional: true,
			MarkdownDescription: "How `resource_types` is matched: `is` — the resource type is one of the values, `isNot` — it is none of them, " +
				"`contains` — it contains any of the values, `notContains` — it contains none of them. Defaults to `is`.",
			Validators: []validator.String{
				stringvalidator.OneOf(utils.TextOperators...),
			},
		},
		"resource_tags": schema.SetAttribute{
			ElementType:         types.StringType,
			Optional:            true,
			MarkdownDescription: "Return only roles whose resource has these tags (see `resource_tags_operator` and `resource_tags_relation`).",
			Validators:          []validator.Set{setvalidator.SizeAtLeast(1)},
		},
		"resource_tags_operator": schema.StringAttribute{
			Optional: true,
			MarkdownDescription: "How `resource_tags` is matched: `is` returns roles whose resource has the tags, " +
				"`isNot` returns roles whose resource does not have them, including resources with no tags. Defaults to `is`.",
			Validators: []validator.String{
				stringvalidator.OneOf(utils.Operators...),
			},
		},
		"resource_tags_relation": schema.StringAttribute{
			Optional: true,
			MarkdownDescription: "How multiple `resource_tags` are combined. With `is`: `or` matches resources that have any of the tags, " +
				"`and` matches resources that have all of them. With `isNot`: `or` matches resources that have none of the tags, " +
				"`and` matches resources that are missing at least one of them. Defaults to `or`.",
			Validators: []validator.String{
				stringvalidator.OneOf(utils.Relations...),
			},
		},
		"page": schema.Int64Attribute{
			Optional:            true,
			MarkdownDescription: "**Deprecated.** Ignored; all pages are read.",
			DeprecationMessage:  utils.PaginationDeprecation,
		},
		"per_page": schema.Int64Attribute{
			Optional:            true,
			MarkdownDescription: "**Deprecated.** Ignored; all pages are read.",
			DeprecationMessage:  utils.PaginationDeprecation,
		},
	}
}

// Schema defines the schema for the data source.
func (d *RolesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: docs.RolesDataSourceMarkdownDescription,
		Blocks: map[string]schema.Block{
			"filter": schema.SingleNestedBlock{
				MarkdownDescription: "Filters for the roles. One of `resource_id`, `integration_id` or `ids` is required. " +
					"All filters are combined with AND. Every matching role is returned; pagination is handled by the provider.",
				Attributes: rolesFilterAttributes(),
			},
		},
		Attributes: map[string]schema.Attribute{
			"resource_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "**Deprecated.** Use `filter.resource_id`.",
				DeprecationMessage:  "Use filter.resource_id instead.",
				Validators: []validator.String{
					validators.UUID{},
					// Root attributes are always validated, also without a filter
					// block, so the scope requirement lives here.
					stringvalidator.AtLeastOneOf(
						path.MatchRoot("resource_id"),
						path.MatchRoot("integration_id"),
						path.MatchRoot("filter").AtName("resource_id"),
						path.MatchRoot("filter").AtName("integration_id"),
						path.MatchRoot("filter").AtName("ids"),
					),
				},
			},
			"integration_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "**Deprecated.** Use `filter.integration_id`.",
				DeprecationMessage:  "Use filter.integration_id instead.",
				Validators: []validator.String{
					validators.UUID{},
				},
			},
			"roles": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "All roles matching the filter, sorted by ID, or by match rank and then ID when `filter.search` is set.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":          schema.StringAttribute{Computed: true},
						"name":        schema.StringAttribute{Computed: true},
						"external_id": schema.StringAttribute{Computed: true},
						"requestable": schema.BoolAttribute{Computed: true},
						"resource_id": schema.StringAttribute{
							Computed:            true,
							Description:         "The unique identifier of the resource the role belongs to.",
							MarkdownDescription: "The unique identifier of the resource the role belongs to.",
						},
						"workflow": schema.SingleNestedAttribute{
							Attributes: map[string]schema.Attribute{
								"id": schema.StringAttribute{
									Computed:            true,
									Description:         "Workflow's unique identifier.",
									MarkdownDescription: "Workflow's unique identifier.",
								},
								"name": schema.StringAttribute{
									Computed:            true,
									Description:         "workflow's name",
									MarkdownDescription: "workflow's name",
								},
							},
							Computed: true,
						},
					},
				},
			},
		},
	}
}

// Configure sets the client used by the data source.
func (d *RolesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := utils.ClientFromProviderData(req.ProviderData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.ClientWithResponses, got: %T", req.ProviderData),
		)
		return
	}

	d.client = c
}

// query converts the filter into list query parameters. A nil filter yields an
// empty query; the client fills in default operators. search is not sent: the API does not accept it together with the
// afterId cursor or the other filters, so it is applied locally (see utils.SearchByName).
func (f *RolesFilterModel) query(ctx context.Context) (client.ListQuery, diag.Diagnostics) {
	var (
		q     client.ListQuery
		diags diag.Diagnostics
	)
	if f == nil {
		return q, diags
	}

	q.ResourceID = f.ResourceID.ValueString()
	q.IntegrationID = f.IntegrationID.ValueString()
	q.ExternalID = f.ExternalID.ValueString()
	q.IDs = utils.SetStrings(ctx, f.IDs, &diags)
	q.IDsOperator = client.IDsOperator(f.IDsOperator.ValueString())
	q.Requestable = f.Requestable.ValueBoolPointer()
	q.WorkflowIDs = utils.SetStrings(ctx, f.WorkflowIDs, &diags)
	q.WorkflowIDOperator = f.WorkflowIDsOperator.ValueString()
	q.ResourceTypes = utils.SetStrings(ctx, f.ResourceTypes, &diags)
	q.ResourceTypeOperator = f.ResourceTypesOperator.ValueString()
	q.ResourceTags = utils.SetStrings(ctx, f.ResourceTags, &diags)
	q.ResourceTagsOperator = f.ResourceTagsOperator.ValueString()
	q.ResourceTagsRelation = f.ResourceTagsRelation.ValueString()
	return q, diags
}

// Read retrieves every matching role from the Entitle API, following the
// afterId cursor until the last page, then applies filter.search locally.
func (d *RolesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data RolesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	query, diags := data.Filter.query(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	// The deprecated top-level attributes conflict with the filter ones, so at
	// most one of each pair is set.
	if query.ResourceID == "" {
		query.ResourceID = data.ResourceID.ValueString()
	}
	if query.IntegrationID == "" {
		query.IntegrationID = data.IntegrationID.ValueString()
	}

	items, apiResp, err := client.ListAll(ctx, query, d.client.RolesListWithResponse,
		func(r client.IntegrationResourceRoleListItemResponseSchema) string { return r.Id.String() })
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiConnection.Error(),
			fmt.Sprintf("Unable to list roles: %s", err))
		return
	}

	if err := utils.HTTPResponseToError(apiResp.StatusCode(), apiResp.Body); err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiResponse.Error(),
			fmt.Sprintf("Failed to list roles: %s", err))
		return
	}

	if data.Filter != nil {
		items = utils.SearchByName(items, data.Filter.Search.ValueString(),
			func(r client.IntegrationResourceRoleListItemResponseSchema) string { return r.Name })
	}

	roles := make([]RoleListItem, len(items))
	for i, r := range items {
		roles[i] = RoleListItem{
			ID:          types.StringValue(r.Id.String()),
			Name:        types.StringValue(r.Name),
			ExternalID:  types.StringValue(r.GetExternalID()),
			Requestable: types.BoolPointerValue(r.Requestable),
			ResourceID:  types.StringValue(r.Resource.Id.String()),
		}

		if r.Workflow != nil {
			roles[i].Workflow = new(utils.IdNameModel{
				ID:   utils.TrimmedStringValue(r.Workflow.Id.String()),
				Name: utils.TrimmedStringValue(r.Workflow.Name),
			})
		}
	}

	data.Roles = roles
	tflog.Trace(ctx, "Read entitle roles list data source", map[string]any{"count": len(roles)})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
