package resources

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

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
	IntegrationID types.String            `tfsdk:"integration_id"`
	Filter        *ResourcesFilterModel   `tfsdk:"filter"`
	Resources     []ResourceListItemModel `tfsdk:"resources"`
}

// ResourcesFilterModel defines filter attributes.
type ResourcesFilterModel struct {
	Search types.String `tfsdk:"search"`
}

// ResourceListItemModel represents a single resource in the list.
type ResourceListItemModel struct {
	ID          types.String       `tfsdk:"id"`
	Name        types.String       `tfsdk:"name"`
	Integration *utils.IdNameModel `tfsdk:"integration"`
}

// Metadata sets the metadata for the data source.
func (d *ResourcesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resources"
}

// Schema defines the schema for the data source.
func (d *ResourcesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieve a list of Entitle Resources filtered by integration ID (mandatory) and optional search string.",
		Blocks: map[string]schema.Block{
			"filter": schema.SingleNestedBlock{
				MarkdownDescription: "Optional filter for resources.",
				Attributes: map[string]schema.Attribute{
					"search": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "Search string to filter resources.",
					},
				},
			},
		},
		Attributes: map[string]schema.Attribute{
			"integration_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Filter resources assigned to a specific integration ID (UUID).",
				Validators: []validator.String{
					validators.UUID{},
				},
			},
			"resources": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "List of resources matching the filter.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":   schema.StringAttribute{Computed: true},
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

// Read retrieves resources from the Entitle API using filters.
func (d *ResourcesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ResourcesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if data.IntegrationID.IsNull() || data.IntegrationID.ValueString() == "" {
		resp.Diagnostics.AddError("Invalid Configuration", "integration_id is mandatory")
		return
	}

	// Prepare optional search
	var search *string
	if data.Filter != nil && !data.Filter.Search.IsNull() && data.Filter.Search.ValueString() != "" {
		s := data.Filter.Search.ValueString()
		search = &s
	}

	// Call API
	params := client.ResourcesIndexParams{
		IntegrationId: data.IntegrationID.ValueString(),
		Search:        search,
	}
	apiResp, err := d.client.ResourcesIndexWithResponse(ctx, &params)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to list resources: %s", err))
		return
	}

	if err := utils.HTTPResponseToError(apiResp.HTTPResponse.StatusCode, apiResp.Body); err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Failed to list resources: %s", err))
		return
	}

	// Map API results
	resources := make([]ResourceListItemModel, len(apiResp.JSON200.Result))
	for i, r := range apiResp.JSON200.Result {
		var integration *utils.IdNameModel
		if r.Integration.Id.String() != "" {
			integration = &utils.IdNameModel{
				ID:   utils.TrimmedStringValue(r.Integration.Id.String()),
				Name: utils.TrimmedStringValue(r.Integration.Name),
			}
		}

		resources[i] = ResourceListItemModel{
			ID:          types.StringValue(r.Id.String()),
			Name:        types.StringValue(r.Name),
			Integration: integration,
		}
	}

	data.Resources = resources
	tflog.Trace(ctx, "Read entitle resources list data source")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
