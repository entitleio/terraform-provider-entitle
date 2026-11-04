package resources

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
var _ datasource.DataSource = &ResourcesDataSource{}

// ResourcesDataSource defines the list data source implementation.
type ResourcesDataSource struct {
	client *client.ClientWithResponses
}

// NewResourcesDataSource creates a new instance of ResourcesDataSource.
func NewResourcesDataSource() datasource.DataSource {
	return &ResourcesDataSource{}
}

// ResourcesDataSourceModel describes the data source model.
type ResourcesDataSourceModel struct {
	// Deprecated: use filter.integration_id.
	IntegrationID types.String            `tfsdk:"integration_id"`
	Filter        *ResourcesFilterModel   `tfsdk:"filter"`
	Resources     []ResourceListItemModel `tfsdk:"resources"`
}

// ResourcesFilterModel is the filter block of entitle_resources. Pagination is
// handled internally: the data source always returns every matching resource.
type ResourcesFilterModel struct {
	IntegrationID         types.String `tfsdk:"integration_id"`
	Search                types.String `tfsdk:"search"`
	ExternalID            types.String `tfsdk:"external_id"`
	IDs                   types.Set    `tfsdk:"ids"`
	IDsOperator           types.String `tfsdk:"ids_operator"`
	Requestable           types.Bool   `tfsdk:"requestable"`
	WorkflowIDs           types.Set    `tfsdk:"workflow_ids"`
	WorkflowIDsOperator   types.String `tfsdk:"workflow_ids_operator"`
	OwnerIDs              types.Set    `tfsdk:"owner_ids"`
	OwnerIDsOperator      types.String `tfsdk:"owner_ids_operator"`
	MaintainerIDs         types.Set    `tfsdk:"maintainer_ids"`
	MaintainerIDsOperator types.String `tfsdk:"maintainer_ids_operator"`
	Types                 types.Set    `tfsdk:"types"`
	TypesOperator         types.String `tfsdk:"types_operator"`
	Tags                  types.Set    `tfsdk:"tags"`
	TagsOperator          types.String `tfsdk:"tags_operator"`
	TagsRelation          types.String `tfsdk:"tags_relation"`

	// Deprecated: ignored, every page is read.
	Page    types.Int64 `tfsdk:"page"`
	PerPage types.Int64 `tfsdk:"per_page"`
}

// ResourceListItemModel represents a single resource in the list.
type ResourceListItemModel struct {
	ID          types.String       `tfsdk:"id"`
	ExternalID  types.String       `tfsdk:"external_id"`
	Name        types.String       `tfsdk:"name"`
	Integration *utils.IdNameModel `tfsdk:"integration"`
}

// Metadata sets the metadata for the data source.
func (d *ResourcesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resources"
}

func resourcesFilterAttributes() map[string]schema.Attribute {
	shared := utils.ListFilterAttributes("resources", true)

	uuidSet := func(description string) schema.SetAttribute {
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

	return map[string]schema.Attribute{
		"integration_id": schema.StringAttribute{
			Optional:            true,
			MarkdownDescription: "Return only the resources of this integration. When omitted, resources of every integration are returned.",
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
			MarkdownDescription: "Return only resources whose name contains this term (case-insensitive). Exact matches are ranked first, " +
				"followed by case-insensitive matches and then partial matches. Can be combined with every other filter.",
		},
		"requestable": schema.BoolAttribute{
			Optional:            true,
			MarkdownDescription: "Return only resources that are (`true`) or are not (`false`) requestable.",
		},
		"workflow_ids": uuidSet("Return only resources with one of these workflow IDs (UUID) (see `workflow_ids_operator`)."),
		"workflow_ids_operator": schema.StringAttribute{
			Optional: true,
			MarkdownDescription: "How `workflow_ids` is matched: `is` returns resources with one of the given workflows, " +
				"`isNot` returns resources with none of them, including resources with no workflow of their own. Defaults to `is`.",
			Validators: []validator.String{
				stringvalidator.OneOf(utils.Operators...),
			},
		},
		"owner_ids": uuidSet("Return only resources owned by one of these users (UUID) (see `owner_ids_operator`)."),
		"owner_ids_operator": schema.StringAttribute{
			Optional: true,
			MarkdownDescription: "How `owner_ids` is matched: `is` returns resources owned by one of the given users, " +
				"`isNot` returns resources owned by none of them, including resources with no owner. Defaults to `is`.",
			Validators: []validator.String{
				stringvalidator.OneOf(utils.Operators...),
			},
		},
		"maintainer_ids": uuidSet("Return only resources maintained by one of these users or groups (UUID) (see `maintainer_ids_operator`)."),
		"maintainer_ids_operator": schema.StringAttribute{
			Optional: true,
			MarkdownDescription: "How `maintainer_ids` is matched: `contains` returns resources that have at least one of the given maintainers, " +
				"`notContains` returns resources that have none of them. Defaults to `contains`.",
			Validators: []validator.String{
				stringvalidator.OneOf(utils.ContainsOperators...),
			},
		},
		"types": schema.SetAttribute{
			ElementType:         types.StringType,
			Optional:            true,
			MarkdownDescription: "Return only resources whose type matches these values (see `types_operator`).",
			Validators:          []validator.Set{setvalidator.SizeAtLeast(1)},
		},
		"types_operator": schema.StringAttribute{
			Optional: true,
			MarkdownDescription: "How `types` is matched: `is` — the type is one of the values, `isNot` — it is none of them, " +
				"`contains` — it contains any of the values, `notContains` — it contains none of them. Defaults to `is`.",
			Validators: []validator.String{
				stringvalidator.OneOf(utils.TextOperators...),
			},
		},
		"tags": schema.SetAttribute{
			ElementType: types.StringType,
			Optional:    true,
			MarkdownDescription: "Return only resources with these tags, system (`tags`) or custom (`userDefinedTags`) " +
				"(see `tags_operator` and `tags_relation`).",
			Validators: []validator.Set{setvalidator.SizeAtLeast(1)},
		},
		"tags_operator": schema.StringAttribute{
			Optional: true,
			MarkdownDescription: "How `tags` is matched: `is` returns resources that have the tags, " +
				"`isNot` returns resources that do not have them, including resources with no tags. Defaults to `is`.",
			Validators: []validator.String{
				stringvalidator.OneOf(utils.Operators...),
			},
		},
		"tags_relation": schema.StringAttribute{
			Optional: true,
			MarkdownDescription: "How multiple `tags` are combined. With `is`: `or` matches resources that have any of the tags, " +
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
func (d *ResourcesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: docs.ResourcesDataSourceMarkdownDescription,
		Blocks: map[string]schema.Block{
			"filter": schema.SingleNestedBlock{
				MarkdownDescription: "Filters for the resources. All filters are combined with AND. " +
					"Every matching resource is returned; pagination is handled by the provider.",
				Attributes: resourcesFilterAttributes(),
			},
		},
		Attributes: map[string]schema.Attribute{
			"integration_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "**Deprecated.** Use `filter.integration_id`.",
				DeprecationMessage:  "Use filter.integration_id instead.",
				Validators: []validator.String{
					validators.UUID{},
				},
			},
			"resources": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "All resources matching the filter, sorted by ID, or by match rank and then ID when `filter.search` is set.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{Computed: true},
						"external_id": schema.StringAttribute{
							Computed:            true,
							Description:         "The external ID of the resource as assigned by the upstream integration.",
							MarkdownDescription: "The external ID of the resource as assigned by the upstream integration.",
						},
						"name": schema.StringAttribute{Computed: true},
						"integration": schema.SingleNestedAttribute{
							Computed: true,
							Attributes: map[string]schema.Attribute{
								"id":   schema.StringAttribute{Computed: true},
								"name": schema.StringAttribute{Computed: true},
							},
						},
					},
				},
			},
		},
	}
}

// Configure sets the client used by the data source.
func (d *ResourcesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
// empty query; the client fills in default operators. search is not sent: the
// API does not accept it together with the afterId cursor or the other filters,
// so it is applied locally (see utils.SearchByName).
func (f *ResourcesFilterModel) query(ctx context.Context) (client.ListQuery, diag.Diagnostics) {
	var (
		q     client.ListQuery
		diags diag.Diagnostics
	)
	if f == nil {
		return q, diags
	}

	q.IntegrationID = f.IntegrationID.ValueString()
	q.ExternalID = f.ExternalID.ValueString()
	q.IDs = utils.SetStrings(ctx, f.IDs, &diags)
	q.IDsOperator = client.IDsOperator(f.IDsOperator.ValueString())
	q.Requestable = f.Requestable.ValueBoolPointer()
	q.WorkflowIDs = utils.SetStrings(ctx, f.WorkflowIDs, &diags)
	q.WorkflowIDOperator = f.WorkflowIDsOperator.ValueString()
	q.OwnerIDs = utils.SetStrings(ctx, f.OwnerIDs, &diags)
	q.OwnerIDOperator = f.OwnerIDsOperator.ValueString()
	q.MaintainerIDs = utils.SetStrings(ctx, f.MaintainerIDs, &diags)
	q.MaintainerIDOperator = f.MaintainerIDsOperator.ValueString()
	q.Types = utils.SetStrings(ctx, f.Types, &diags)
	q.TypeOperator = f.TypesOperator.ValueString()
	q.Tags = utils.SetStrings(ctx, f.Tags, &diags)
	q.TagsOperator = f.TagsOperator.ValueString()
	q.TagsRelation = f.TagsRelation.ValueString()
	return q, diags
}

// Read retrieves every matching resource from the Entitle API, following the
// afterId cursor until the last page, then applies filter.search locally.
func (d *ResourcesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ResourcesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	query, diags := data.Filter.query(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	// The deprecated top-level integration_id conflicts with filter.integration_id,
	// so at most one of them is set.
	if query.IntegrationID == "" {
		query.IntegrationID = data.IntegrationID.ValueString()
	}

	items, apiResp, err := client.ListAll(ctx, query, d.client.ResourcesListWithResponse,
		func(r client.IntegrationResourceResultSchema) string { return r.Id.String() })
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiConnection.Error(),
			fmt.Sprintf("Unable to list resources: %s", err))
		return
	}

	if err := utils.HTTPResponseToError(apiResp.StatusCode(), apiResp.Body); err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiResponse.Error(),
			fmt.Sprintf("Failed to list resources: %s", err))
		return
	}

	if data.Filter != nil {
		items = utils.SearchByName(items, data.Filter.Search.ValueString(),
			func(r client.IntegrationResourceResultSchema) string { return r.Name })
	}

	resources := make([]ResourceListItemModel, len(items))
	for i, r := range items {
		var integration *utils.IdNameModel
		if r.Integration.Id.String() != "" {
			integration = &utils.IdNameModel{
				ID:   utils.TrimmedStringValue(r.Integration.Id.String()),
				Name: utils.TrimmedStringValue(r.Integration.Name),
			}
		}

		var externalID string
		if r.ExternalId != nil {
			externalID = *r.ExternalId
		}

		resources[i] = ResourceListItemModel{
			ID:          types.StringValue(r.Id.String()),
			ExternalID:  types.StringValue(externalID),
			Name:        types.StringValue(r.Name),
			Integration: integration,
		}
	}

	data.Resources = resources
	tflog.Trace(ctx, "Read entitle resources list data source", map[string]any{
		"integration_id": query.IntegrationID,
		"resource_count": len(resources),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
