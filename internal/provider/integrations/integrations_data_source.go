package integrations

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/entitleio/terraform-provider-entitle/docs"
	"github.com/entitleio/terraform-provider-entitle/internal/client"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/utils"
)

var _ datasource.DataSource = &IntegrationsDataSource{}

// NewIntegrationsDataSource creates a new entitle_integrations data source.
func NewIntegrationsDataSource() datasource.DataSource {
	return &IntegrationsDataSource{}
}

// IntegrationsDataSource lists integrations.
type IntegrationsDataSource struct {
	client *client.ClientWithResponses
}

// IntegrationsDataSourceModel describes the data source model.
type IntegrationsDataSourceModel struct {
	Filter       *utils.IDsFilterModel      `tfsdk:"filter"`
	Integrations []IntegrationListItemModel `tfsdk:"integrations"`
}

// IntegrationListItemModel is a single integration in the list.
type IntegrationListItemModel struct {
	ID          types.String     `tfsdk:"id"`
	Name        types.String     `tfsdk:"name"`
	Application *utils.NameModel `tfsdk:"application"`
}

// Metadata sets the data source type name.
func (d *IntegrationsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integrations"
}

// Schema defines the data source schema.
func (d *IntegrationsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: docs.IntegrationsDataSourceMarkdownDescription,
		Blocks: map[string]schema.Block{
			"filter": schema.SingleNestedBlock{
				MarkdownDescription: "Optional filter and pagination for integrations.",
				Attributes:          utils.ListFilterAttributes("integrations", false),
			},
		},
		Attributes: map[string]schema.Attribute{
			"integrations": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "List of integrations matching the filter.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The integration's unique identifier.",
						},
						"name": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "The integration's display name.",
						},
						"application": schema.SingleNestedAttribute{
							Computed:            true,
							MarkdownDescription: "The application the integration connects to.",
							Attributes: map[string]schema.Attribute{
								"name": schema.StringAttribute{
									Computed:            true,
									MarkdownDescription: "The application's name.",
								},
							},
						},
					},
				},
			},
		},
	}
}

// Configure sets the API client.
func (d *IntegrationsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := utils.ClientFromProviderData(req.ProviderData)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *utils.ProviderData, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)
		return
	}

	d.client = c
}

// Read lists integrations from the Entitle API.
func (d *IntegrationsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data IntegrationsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	query, diags := data.Filter.Query(ctx)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	apiResp, err := d.client.IntegrationsListWithResponse(ctx, query)
	if err != nil {
		resp.Diagnostics.AddError(utils.ErrApiConnection.Error(), fmt.Sprintf("Unable to list integrations: %s", err))
		return
	}
	if err := utils.HTTPResponseToError(apiResp.StatusCode(), apiResp.Body); err != nil {
		resp.Diagnostics.AddError(utils.ErrApiResponse.Error(), fmt.Sprintf("Failed to list integrations: %s", err))
		return
	}

	integrations := make([]IntegrationListItemModel, len(apiResp.Result))
	for i, it := range apiResp.Result {
		integrations[i] = IntegrationListItemModel{
			ID:          types.StringValue(it.Id.String()),
			Name:        utils.TrimmedStringValue(it.Name),
			Application: &utils.NameModel{Name: utils.TrimmedStringValue(it.Application.Name)},
		}
	}
	data.Integrations = integrations

	tflog.Trace(ctx, "Read entitle integrations list data source", map[string]any{
		"count": len(integrations),
		"query": query.Values().Encode(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
