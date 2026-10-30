package roles

import (
	"context"
	"fmt"

	"github.com/entitleio/terraform-provider-entitle/internal/validators"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/entitleio/terraform-provider-entitle/docs"
	"github.com/entitleio/terraform-provider-entitle/internal/client"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/utils"
)

// Ensure the data source satisfies the framework interface.
var _ datasource.DataSource = &RolesDataSource{}
var _ datasource.DataSourceWithValidateConfig = &RolesDataSource{}

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
	ResourceID    types.String           `tfsdk:"resource_id"`
	IntegrationID types.String           `tfsdk:"integration_id"`
	Filter        *utils.ListFilterModel `tfsdk:"filter"`
	Roles         []RoleListItem         `tfsdk:"roles"`
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

// Schema defines the schema for the data source.
func (d *RolesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: docs.RolesDataSourceMarkdownDescription,
		Blocks: map[string]schema.Block{
			"filter": schema.SingleNestedBlock{
				MarkdownDescription: "Pagination and filter roles by optional search term.",
				Attributes:          utils.ListFilterAttributes("roles", true),
			},
		},
		Attributes: map[string]schema.Attribute{
			"resource_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Filter roles assigned to a specific resource ID. One of `resource_id`, `integration_id` or `filter.ids` is required.",
				Validators: []validator.String{
					validators.UUID{},
				},
			},
			"integration_id": schema.StringAttribute{
				Optional:            true,
				MarkdownDescription: "Filter roles assigned to a specific integration ID. One of `resource_id`, `integration_id` or `filter.ids` is required.",
				Validators: []validator.String{
					validators.UUID{},
				},
			},
			"roles": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "List of roles matching the filter.",
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

// ValidateConfig requires resource_id, integration_id or filter.ids.
func (d *RolesDataSource) ValidateConfig(ctx context.Context, req datasource.ValidateConfigRequest, resp *datasource.ValidateConfigResponse) {
	var data RolesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ResourceID.IsNull() && data.IntegrationID.IsNull() && !data.Filter.HasIDs() {
		resp.Diagnostics.AddAttributeError(path.Root("resource_id"), "Missing Filter",
			"One of resource_id, integration_id or filter.ids must be set.")
	}
}

// Read retrieves roles from the Entitle API using filters.
func (d *RolesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data RolesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	query, diags := data.Filter.Query(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	query.ResourceID = data.ResourceID.ValueString()
	query.IntegrationID = data.IntegrationID.ValueString()

	apiResp, err := d.client.RolesListWithResponse(ctx, query)
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

	roles := make([]RoleListItem, len(apiResp.Result))
	for i, r := range apiResp.Result {
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
	tflog.Trace(ctx, "Read entitle roles list data source")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
