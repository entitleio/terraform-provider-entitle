package directoryGroups

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/entitleio/terraform-provider-entitle/internal/client"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/utils"
)

// Ensure the data source satisfies the framework interface.
var _ datasource.DataSource = &DirectoryGroupsDataSource{}

// DirectoryGroupsDataSource defines the implementation of the list data source.
type DirectoryGroupsDataSource struct {
	client *client.ClientWithResponses
}

// NewDirectoryGroupsDataSource creates a new instance of DirectoryGroupsDataSource.
func NewDirectoryGroupsDataSource() datasource.DataSource {
	return &DirectoryGroupsDataSource{}
}

// DirectoryGroupsDataSourceModel describes the data source model.
type DirectoryGroupsDataSourceModel struct {
	Filter          *DirectoryGroupsListFilterModel `tfsdk:"filter"`
	DirectoryGroups []DirectoryGroupListItem        `tfsdk:"directory_groups"`
}

// DirectoryGroupsListFilterModel defines filter attributes.
type DirectoryGroupsListFilterModel struct {
	Search  types.String `tfsdk:"search"`
	Page    types.Int64  `tfsdk:"page"`
	PerPage types.Int64  `tfsdk:"per_page"`
}

// DirectoryGroupListItem represents a single directory group in the list.
type DirectoryGroupListItem struct {
	Id     types.String `tfsdk:"id"`
	Email  types.String `tfsdk:"email"`
	Name   types.String `tfsdk:"name"`
	Origin types.String `tfsdk:"origin"`
}

// Metadata sets the metadata for the data source.
func (d *DirectoryGroupsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_directory_groups"
}

// Schema defines the schema for the data source.
func (d *DirectoryGroupsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieve a list of Entitle DirectoryGroups filtered by optional search string.",
		Blocks: map[string]schema.Block{
			"filter": schema.SingleNestedBlock{
				MarkdownDescription: "Filter directoryGroups by optional search term.",
				Attributes: map[string]schema.Attribute{
					"search": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "Search string to filter directoryGroups.",
					},
					"page": schema.Int64Attribute{
						Optional:            true,
						MarkdownDescription: "Page number.",
					},
					"per_page": schema.Int64Attribute{
						Optional:            true,
						MarkdownDescription: "Number of results per page.",
					},
				},
			},
		},
		Attributes: map[string]schema.Attribute{
			"directory_groups": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "List of directoryGroups matching the filter.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":     schema.StringAttribute{Computed: true},
						"name":   schema.StringAttribute{Computed: true},
						"email":  schema.StringAttribute{Computed: true},
						"origin": schema.StringAttribute{Computed: true},
					},
				},
			},
		},
	}
}

// Configure sets the client used by the data source.
func (d *DirectoryGroupsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

// Read retrieves directoryGroups from the Entitle API using filters.
func (d *DirectoryGroupsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data DirectoryGroupsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var params client.DirectoryGroupsIndexParams

	if data.Filter != nil {
		s := data.Filter.Search.ValueString()
		if s != "" {
			params.Search = &s
		}

		page := int(data.Filter.Page.ValueInt64())
		if page > 0 {
			params.Page = &page
		}

		perPage := int(data.Filter.PerPage.ValueInt64())
		if perPage > 0 {
			params.PerPage = &perPage
		}
	}

	apiResp, err := d.client.DirectoryGroupsIndexWithResponse(ctx, &params)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list Directory Groups: %s", err))
		return
	}

	if err := utils.HTTPResponseToError(apiResp.HTTPResponse.StatusCode, apiResp.Body); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Failed to list Directory Groups: %s", err))
		return
	}

	directoryGroups := make([]DirectoryGroupListItem, len(apiResp.JSON200.Result))
	for i, r := range apiResp.JSON200.Result {
		directoryGroups[i] = DirectoryGroupListItem{
			Id:     types.StringValue(r.Id.String()),
			Name:   types.StringValue(r.Name),
			Email:  types.StringValue(r.Email),
			Origin: types.StringValue(r.Origin),
		}
	}

	data.DirectoryGroups = directoryGroups
	tflog.Trace(ctx, "Read entitle directory group list data source")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
