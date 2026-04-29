package applications

import (
	"context"
	"fmt"

	"github.com/entitleio/terraform-provider-entitle/docs"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/entitleio/terraform-provider-entitle/internal/client"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/utils"
)

// Ensure the data source satisfies the framework interface.
var _ datasource.DataSource = &ApplicationsDataSource{}

// ApplicationsDataSource defines the list data source implementation.
type ApplicationsDataSource struct {
	client *client.ClientWithResponses
}

// NewApplicationsDataSource creates a new instance of ApplicationsDataSource.
func NewApplicationsDataSource() datasource.DataSource {
	return &ApplicationsDataSource{}
}

// ApplicationsDataSourceModel describes the data source model.
type ApplicationsDataSourceModel struct {
	Applications []utils.IdNameModel `tfsdk:"applications"`
}

// Metadata sets the metadata for the data source.
func (d *ApplicationsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_applications"
}

// Schema defines the schema for the data source.
func (d *ApplicationsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: docs.ApplicationDataSourceMarkdownDescription,
		Attributes: map[string]schema.Attribute{
			"applications": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "List of applications.",
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
func (d *ApplicationsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

// Read retrieves applications from the Entitle API using filters.
func (d *ApplicationsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ApplicationsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Call API
	apiResp, err := d.client.ApplicationsIndexWithResponse(ctx)
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiConnection.Error(),
			fmt.Sprintf("Unable to list applications: %s", err))
		return
	}

	if err := utils.HTTPResponseToError(apiResp.HTTPResponse.StatusCode, apiResp.Body); err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiResponse.Error(),
			fmt.Sprintf("Failed to list applications: %s", err))
		return
	}

	// Map API results
	data.Applications = make([]utils.IdNameModel, len(apiResp.JSON200.Result))
	for i, a := range apiResp.JSON200.Result {
		data.Applications[i] = utils.IdNameModel{
			ID:   utils.TrimmedStringValue(a.Id.String()),
			Name: utils.TrimmedStringValue(a.Name),
		}
	}

	tflog.Trace(ctx, "Read entitle applications list data source")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
