package users

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

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &UserDataSource{}

// UserDataSource defines the data source implementation.
type UserDataSource struct {
	client *client.ClientWithResponses
}

// NewUserDataSource creates a new instance of UserDataSource.
func NewUserDataSource() datasource.DataSource {
	return &UserDataSource{}
}

// UserDataSourceModel describes the data source data model.
type UserDataSourceModel struct {
	Id         types.String `tfsdk:"id"`
	Email      types.String `tfsdk:"email"`
	GivenName  types.String `tfsdk:"given_name"`
	FamilyName types.String `tfsdk:"family_name"`
	CreatedAt  types.String `tfsdk:"created_at"`
}

// Metadata sets the metadata for the data source.
func (d *UserDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

// Schema defines the data source schema.
func (d *UserDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Defines an Entitle User, which represents organization's employee. [Read more about users](https://docs.beyondtrust.com/entitle/docs/users).",
		Description:         "Defines an Entitle User, which represents organization's employee. [Read more about users](https://docs.beyondtrust.com/entitle/docs/users).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Entitle User identifier in uuid format",
				Description:         "Entitle User identifier in uuid format",
			},
			"email": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Entitle User email address (identifier)",
				Description:         "Entitle User email address (identifier)",
				Validators: []validator.String{
					validators.Email{},
				},
			},
			"given_name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Entitle User given name",
				Description:         "Entitle User given name",
			},
			"family_name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Entitle User family name",
				Description:         "Entitle User family name",
			},
			"created_at": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Entitle User creation time",
				Description:         "Entitle User creation time",
			},
		},
	}
}

// Configure configures the data source with the provided client.
func (d *UserDataSource) Configure(
	ctx context.Context,
	req datasource.ConfigureRequest,
	resp *datasource.ConfigureResponse,
) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.ClientWithResponses)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = c
}

// Read retrieves data from the provider and populates the data source model.
func (d *UserDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data UserDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	email := data.Email.ValueString()
	if email == "" {
		resp.Diagnostics.AddError(
			"Client Error",
			"User email not provided",
		)
	}

	params := client.UsersIndexParams{
		Page:    nil,
		PerPage: nil,
		Search:  &email,
	}

	resourceResp, err := d.client.UsersIndexWithResponse(ctx, &params)
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to get the User by the email (%s), got error: %s", email, err),
		)
		return
	}

	err = utils.HTTPResponseToError(resourceResp.HTTPResponse.StatusCode, resourceResp.Body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to get the User by the email (%s), got error: %s", email, err),
		)
		return
	}

	results := resourceResp.JSON200.Result
	if len(results) == 0 {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf(
				"Failed to get the User by the email (%s), no results found",
				email,
			),
		)
		return
	}

	data = UserDataSourceModel{
		Id:         utils.TrimmedStringValue(results[0].Id.String()),
		Email:      utils.TrimmedStringValue(results[0].Email),
		FamilyName: utils.TrimmedStringValue(results[0].FamilyName),
		GivenName:  utils.TrimmedStringValue(results[0].GivenName),
		CreatedAt:  utils.TrimmedStringValue(results[0].CreatedAt.GoString()),
	}

	tflog.Trace(ctx, "read a entitle user data source")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	diagsResult := resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diagsResult...)
	if resp.Diagnostics.HasError() {
		return
	}
}
