package directoryGroups

import (
	"context"
	"fmt"
	"net/http"
	"strings"

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
var _ datasource.DataSource = &DirectoryGroupDataSource{}

// DirectoryGroupDataSource defines the data source implementation.
type DirectoryGroupDataSource struct {
	client *client.ClientWithResponses
}

// NewDirectoryGroupDataSource creates a new instance of DirectoryGroupDataSource.
func NewDirectoryGroupDataSource() datasource.DataSource {
	return &DirectoryGroupDataSource{}
}

// DirectoryGroupDataSourceModel describes the data source data model.
type DirectoryGroupDataSourceModel struct {
	Id     types.String `tfsdk:"id"`
	Email  types.String `tfsdk:"email"`
	Name   types.String `tfsdk:"name"`
	Origin types.String `tfsdk:"origin"`
}

// Metadata sets the metadata for the data source.
func (d *DirectoryGroupDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_directory_group"
}

// Schema defines the data source schema.
func (d *DirectoryGroupDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A Directory Group in Entitle represents a collection of users from your organization's identity provider (IdP). " +
			"These groups are typically synchronized from systems like Azure AD, Okta, or Google Workspace, " +
			"and are used for managing access permissions and policies.",
		Description: "A Directory Group in Entitle represents a collection of users from your organization's identity provider (IdP). " +
			"These groups are typically synchronized from systems like Azure AD, Okta, or Google Workspace, " +
			"and are used for managing access permissions and policies.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Entitle Directory Group identifier in uuid format",
				Description:         "Entitle Directory Group identifier in uuid format",
			},
			"email": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The email address associated with the Directory Group, used as a primary identifier for synchronization with the IdP",
				Description:         "The email address associated with the Directory Group, used as a primary identifier for synchronization with the IdP",
				Validators: []validator.String{
					validators.Email{},
				},
			},
			"name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Entitle Directory Group name",
				Description:         "Entitle Directory Group name",
			},
			"origin": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The source identity provider (IdP) from which this group was synchronized",
				Description:         "The source identity provider (IdP) from which this group was synchronized",
			},
		},
	}
}

// Configure configures the data source with the provided client.
func (d *DirectoryGroupDataSource) Configure(
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
func (d *DirectoryGroupDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data DirectoryGroupDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	email := data.Email.ValueString()
	if email == "" {
		resp.Diagnostics.AddError(
			"Client Error",
			"Directory Group email not provided",
		)
	}

	params := client.DirectoryGroupsIndexParams{
		Page:    nil,
		PerPage: nil,
		Search:  &email,
	}

	resourceResp, err := d.client.DirectoryGroupsIndexWithResponse(ctx, &params)
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to get the Directory Group by the email (%s), got error: %s", email, err),
		)
		return
	}

	if resourceResp.HTTPResponse.StatusCode != 200 {
		errBody, _ := utils.GetErrorBody(resourceResp.Body)
		if resourceResp.HTTPResponse.StatusCode == http.StatusUnauthorized ||
			(resourceResp.HTTPResponse.StatusCode == http.StatusBadRequest && strings.Contains(errBody.GetMessage(), "is not a valid uuid")) {
			resp.Diagnostics.AddError(
				"Client Error",
				"unauthorized token, update the entitle token and retry please",
			)
			return
		}

		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf(
				"failed to get the Directory Group by the email (%s), status code: %d",
				email,
				resourceResp.HTTPResponse.StatusCode,
			),
		)
		return
	}

	results := resourceResp.JSON200.Result
	if len(results) == 0 {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf(
				"failed to get the Directory Group by the email (%s), no results found",
				email,
			),
		)
		return
	}

	data = DirectoryGroupDataSourceModel{
		Id:     utils.TrimmedStringValue(results[0].Id.String()),
		Email:  utils.TrimmedStringValue(results[0].Email),
		Name:   utils.TrimmedStringValue(results[0].Name),
		Origin: utils.TrimmedStringValue(results[0].Origin),
	}

	tflog.Trace(ctx, "read a entitle directory group data source")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)

	diagsResult := resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diagsResult...)
	if resp.Diagnostics.HasError() {
		return
	}
}
