package users

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

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &UsersDataSource{}

// UsersDataSource defines the data source implementation.
type UsersDataSource struct {
	client *client.ClientWithResponses
}

func NewUsersDataSource() datasource.DataSource {
	return &UsersDataSource{}
}

// UsersDataSourceModel describes the top-level data source state.
type UsersDataSourceModel struct {
	Filter *utils.PaginationWithSearchModel `tfsdk:"filter"`
	Users  []UserModel                      `tfsdk:"users"`
}

// UserModel describes a single user in the list.
type UserModel struct {
	Id         types.String `tfsdk:"id"`
	Email      types.String `tfsdk:"email"`
	GivenName  types.String `tfsdk:"given_name"`
	FamilyName types.String `tfsdk:"family_name"`
	CreatedAt  types.String `tfsdk:"created_at"`
}

func (d *UsersDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_users"
}

func (d *UsersDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieve a list of Entitle Users with optional filters. [Read more about users](https://docs.beyondtrust.com/entitle/docs/users).",
		Attributes: map[string]schema.Attribute{
			"users": schema.ListNestedAttribute{
				Computed:            true,
				MarkdownDescription: "List of users matching the filter.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "User ID",
						},
						"email": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "User email address",
						},
						"given_name": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "User given name",
						},
						"family_name": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "User family name",
						},
						"created_at": schema.StringAttribute{
							Computed:            true,
							MarkdownDescription: "User creation time",
						},
					},
				},
			},
		},
		Blocks: map[string]schema.Block{
			"filter": schema.SingleNestedBlock{
				MarkdownDescription: "Filter options for searching users.",
				Attributes: map[string]schema.Attribute{
					"search": schema.StringAttribute{
						Optional:            true,
						MarkdownDescription: "Free-text search term (matches email or full name).",
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
	}
}

func (d *UsersDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *UsersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data UsersDataSourceModel

	// Load config into model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var params client.UsersIndexParams

	if data.Filter != nil {
		params.Search, params.Page, params.PerPage = data.Filter.GetValues()
	}

	resourceResp, err := d.client.UsersIndexWithResponse(ctx, &params)
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ApiConnectionError.Error(),
			fmt.Sprintf("Unable to list users, got error: %s", err),
		)
		return
	}

	err = utils.HTTPResponseToError(resourceResp.HTTPResponse.StatusCode, resourceResp.Body)
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ApiResponseError.Error(),
			fmt.Sprintf("Unable to list users, got error: %s", err),
		)
		return
	}

	results := resourceResp.JSON200.Result
	users := make([]UserModel, 0, len(results))

	for _, u := range results {
		users = append(users, UserModel{
			Id:         utils.TrimmedStringValue(u.Id.String()),
			Email:      utils.TrimmedStringValue(u.Email),
			GivenName:  utils.TrimmedStringValue(u.GivenName),
			FamilyName: utils.TrimmedStringValue(u.FamilyName),
			CreatedAt:  utils.TrimmedStringValue(u.CreatedAt.GoString()),
		})
	}

	data.Users = users

	tflog.Trace(ctx, "read entitle users data source")

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
