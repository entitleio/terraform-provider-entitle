package roles

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/entitleio/terraform-provider-entitle/internal/client"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/utils"
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
	ResourceID types.String          `tfsdk:"resource_id"`
	Filter     *RolesListFilterModel `tfsdk:"filter"`
	Roles      []RoleListItem        `tfsdk:"roles"`
}

// RolesListFilterModel defines filter attributes.
type RolesListFilterModel struct {
	Search types.String `tfsdk:"search"`
}

// RoleListItem represents a single role in the list.
type RoleListItem struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

// Metadata sets the metadata for the data source.
func (d *RolesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_roles"
}

// Schema defines the schema for the data source.
func (d *RolesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieve a list of Entitle Roles filtered by resource ID (mandatory) and optional search string.",
		Blocks: map[string]schema.Block{
			"filter": schema.SingleNestedBlock{
				MarkdownDescription: "Filter roles by resource ID (mandatory) and optional search term.",
				Attributes: map[string]schema.Attribute{
					"search": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "Search string to filter roles.",
					},
				},
			},
		},
		Attributes: map[string]schema.Attribute{
			"resource_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Filter roles assigned to a specific resource ID.",
			},
			"roles": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "List of roles matching the filter.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":   schema.StringAttribute{Computed: true},
						"name": schema.StringAttribute{Computed: true},
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

	c, ok := req.ProviderData.(*client.ClientWithResponses)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.ClientWithResponses, got: %T", req.ProviderData),
		)
		return
	}

	d.client = c
}

// Read retrieves roles from the Entitle API using filters.
func (d *RolesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data RolesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.ResourceID.IsNull() || data.ResourceID.ValueString() == "" {
		resp.Diagnostics.AddError("Invalid Configuration", "filter.resource_id is mandatory")
		return
	}

	resourceID := data.ResourceID.ValueString()
	// inside Read() after validating resourceID
	uid, err := uuid.Parse(resourceID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid Resource ID",
			fmt.Sprintf("Failed to parse filter.resource_id (%s) as UUID: %s", resourceID, err),
		)
		return
	}

	var search *string
	if data.Filter != nil && !data.Filter.Search.IsNull() && data.Filter.Search.ValueString() != "" {
		s := data.Filter.Search.ValueString()
		search = &s
	}

	params := client.RolesIndexParams{
		Page:       nil,
		PerPage:    nil,
		ResourceId: uid,
		Search:     search,
	}

	apiResp, err := d.client.RolesIndexWithResponse(ctx, &params)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list roles: %s", err))
		return
	}

	if err := utils.HTTPResponseToError(apiResp.HTTPResponse.StatusCode, apiResp.Body); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Failed to list roles: %s", err))
		return
	}

	roles := make([]RoleListItem, len(apiResp.JSON200.Result))
	for i, r := range apiResp.JSON200.Result {
		roles[i] = RoleListItem{
			ID:   types.StringValue(r.Id.String()),
			Name: types.StringValue(r.Name),
		}
	}

	data.Roles = roles
	tflog.Trace(ctx, "Read entitle roles list data source")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
