package accounts

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/entitleio/terraform-provider-entitle/docs"
	"github.com/entitleio/terraform-provider-entitle/internal/client"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/utils"
	"github.com/entitleio/terraform-provider-entitle/internal/validators"
)

// Ensure the data source satisfies the framework interface.
var _ datasource.DataSource = &AccountsDataSource{}

// AccountsDataSource defines the list data source implementation.
type AccountsDataSource struct {
	client *client.ClientWithResponses
}

// NewAccountsDataSource creates a new instance of AccountsDataSource.
func NewAccountsDataSource() datasource.DataSource {
	return &AccountsDataSource{}
}

// AccountsDataSourceModel describes the data source model.
type AccountsDataSourceModel struct {
	IntegrationID types.String                     `tfsdk:"integration_id"`
	Filter        *utils.PaginationWithSearchModel `tfsdk:"filter"`
	Accounts      []AccountListItem                `tfsdk:"accounts"`
}

// IntegrationModel matches the Terraform schema for integration.
type IntegrationModel struct {
	ID          types.String     `tfsdk:"id"`
	Name        types.String     `tfsdk:"name"`
	Application *utils.NameModel `tfsdk:"application"`
}

// AccountListItem represents a single account in the list.
type AccountListItem struct {
	ID          types.String      `tfsdk:"id"`
	Name        types.String      `tfsdk:"name"`
	Email       types.String      `tfsdk:"email"`
	EUID        types.String      `tfsdk:"euid"`
	Integration *IntegrationModel `tfsdk:"integration"`
}

// Metadata sets the metadata for the data source.
func (d *AccountsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_accounts"
}

// Schema defines the schema for the data source.
func (d *AccountsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: docs.AccountsDataSourceMarkdownDescription,
		Blocks: map[string]schema.Block{
			"filter": schema.SingleNestedBlock{
				MarkdownDescription: "Optional filter for accounts.",
				Attributes: map[string]schema.Attribute{
					"search": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "Search string to filter accounts.",
					},
					"page": schema.Int64Attribute{
						Optional:            true,
						MarkdownDescription: "Page number of results to return (starting from 1). Used together with `per_page` for pagination.",
					},
					"per_page": schema.Int64Attribute{
						Optional:            true,
						MarkdownDescription: "Number of results to return per page. Defaults to the API's configured page size if not specified.",
					},
				},
			},
		},
		Attributes: map[string]schema.Attribute{
			"integration_id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Filter accounts assigned to a specific integration ID (UUID).",
				Validators: []validator.String{
					validators.UUID{},
				},
			},
			"accounts": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "List of accounts matching the filter.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":    schema.StringAttribute{Computed: true},
						"name":  schema.StringAttribute{Computed: true},
						"email": schema.StringAttribute{Computed: true},
						"euid":  schema.StringAttribute{Computed: true},
						"integration": schema.SingleNestedAttribute{
							Computed: true,
							Attributes: map[string]schema.Attribute{
								"id":   schema.StringAttribute{Computed: true},
								"name": schema.StringAttribute{Computed: true},
								"application": schema.SingleNestedAttribute{
									Computed: true,
									Attributes: map[string]schema.Attribute{
										"name": schema.StringAttribute{Computed: true},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// Configure sets the client used by the data source.
func (d *AccountsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

// Read retrieves accounts from the Entitle API using filters.
func (d *AccountsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data AccountsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := client.AccountsIndexParams{
		IntegrationId: data.IntegrationID.ValueString(),
	}

	if data.Filter != nil {
		params.Search, params.Page, params.PerPage = data.Filter.GetValues()
	}

	// Call API
	apiResp, err := d.client.AccountsIndexWithResponse(ctx, &params)
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiConnection.Error(),
			fmt.Sprintf("Unable to list accounts: %s", err))
		return
	}

	if err := utils.HTTPResponseToError(apiResp.HTTPResponse.StatusCode, apiResp.Body); err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiResponse.Error(),
			fmt.Sprintf("Failed to list accounts: %s", err))
		return
	}

	// Map API results
	accounts := make([]AccountListItem, len(apiResp.JSON200.Result))
	for i, a := range apiResp.JSON200.Result {
		integration := &IntegrationModel{
			ID:   utils.TrimmedStringValue(a.Integration.Id.String()),
			Name: utils.TrimmedStringValue(a.Integration.Name),
			Application: &utils.NameModel{
				Name: utils.TrimmedStringValue(a.Integration.Application.Name),
			},
		}

		accounts[i] = AccountListItem{
			ID:          types.StringValue(a.Id.String()),
			Name:        types.StringValue(a.Name),
			Email:       types.StringValue(a.Email),
			EUID:        types.StringValue(a.Euid),
			Integration: integration,
		}
	}

	data.Accounts = accounts
	tflog.Trace(ctx, "Read entitle accounts list data source")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
