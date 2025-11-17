package accessReviewForwards

import (
	"context"
	"fmt"

	"github.com/entitleio/terraform-provider-entitle/docs"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/utils"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/entitleio/terraform-provider-entitle/internal/client"
	"github.com/entitleio/terraform-provider-entitle/internal/validators"
)

// Ensure that the provider-defined types fully satisfy the framework interfaces.
var _ datasource.DataSource = &AccessReviewForwardDataSource{}

// AccessReviewForwardDataSource defines the data source implementation for the Terraform provider.
type AccessReviewForwardDataSource struct {
	client *client.ClientWithResponses
}

// NewAccessReviewForwardDataSource creates a new instance of the AccessReviewForwardDataSource.
func NewAccessReviewForwardDataSource() datasource.DataSource {
	return &AccessReviewForwardDataSource{}
}

// AccessReviewForwardDataSourceModel defines the data model.
type AccessReviewForwardDataSourceModel struct {
	ID        types.String        `tfsdk:"id" json:"id"`
	Forwarder *utils.IdEmailModel `tfsdk:"forwarder" json:"forwarder"`
	Target    *utils.IdEmailModel `tfsdk:"target" json:"target"`
}

// Metadata sets the data source's metadata, such as its type name.
func (d *AccessReviewForwardDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_access_review_forward"
}

// Schema defines the expected structure of the data source.
func (d *AccessReviewForwardDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: docs.AccessReviewForwardDataSourceMarkdownDescription,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Entitle Access Review Forward identifier in uuid format",
				Description:         "Entitle Access Review Forward identifier in uuid format",
				Validators: []validator.String{
					validators.UUID{},
				},
			},
			"forwarder": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Computed:            true,
						Description:         "the forwarder user's identifier in uuid format",
						MarkdownDescription: "the forwarder user's identifier in uuid format",
					},
					"email": schema.StringAttribute{
						Computed:            true,
						Description:         "the forwarder user's email address",
						MarkdownDescription: "the forwarder user's email address",
					},
				},
				Computed: true,
				Description: "Specifies the user who is delegating or forwarding their access review responsibilities. " +
					"This user must have review permissions for the items being forwarded.",
				MarkdownDescription: "Specifies the user who is delegating or forwarding their access review responsibilities. " +
					"This user must have review permissions for the items being forwarded.",
			},
			"target": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Computed:            true,
						Description:         "the target user's identifier in uuid format",
						MarkdownDescription: "the target user's identifier in uuid format",
					},
					"email": schema.StringAttribute{
						Computed:            true,
						Description:         "the target user's email address",
						MarkdownDescription: "the target user's email address",
					},
				},
				Computed: true,
				Description: "Defines the user who will receive and be responsible for completing the forwarded access review tasks. " +
					"This user will temporarily assume the review responsibilities for the specified items.",
				MarkdownDescription: "Defines the user who will receive and be responsible for completing the forwarded access review tasks. " +
					"This user will temporarily assume the review responsibilities for the specified items.",
			},
		},
	}
}

// Configure configures the data source with the provider's client.
func (d *AccessReviewForwardDataSource) Configure(
	ctx context.Context,
	req datasource.ConfigureRequest,
	resp *datasource.ConfigureResponse,
) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	cli, ok := req.ProviderData.(*client.ClientWithResponses)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = cli
}

// Read reads data from the external source and sets it in Terraform state.
// It retrieves the configuration data from Terraform, sends a request to the Entitle API to get the access review forward data,
// converts the API response to the data source model, and saves it into Terraform state.
func (d *AccessReviewForwardDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data AccessReviewForwardDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Parse the resource ID from the data source model
	uid, err := uuid.Parse(data.ID.String())
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Failed to parse the resource id (%s) to UUID, got error: %s", data.ID.String(), err),
		)
		return
	}

	// Send a request to the Entitle API to get the access review forward data by ID
	apiResp, err := d.client.AccessReviewForwardsShowWithResponse(ctx, uid)
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiConnection.Error(),
			fmt.Sprintf("Unable to get the access review forward by the id (%s), got error: %s", uid.String(), err),
		)
		return
	}

	err = utils.HTTPResponseToError(apiResp.HTTPResponse.StatusCode, apiResp.Body)
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiResponse.Error(),
			fmt.Sprintf(
				"Failed to get the Access Review Forward by the id (%s), status code: %d, %s",
				uid.String(),
				apiResp.HTTPResponse.StatusCode,
				err.Error(),
			))
		return
	}

	responseSchema := apiResp.JSON200.Result

	data = AccessReviewForwardDataSourceModel{
		ID: utils.TrimmedStringValue(responseSchema.Id.String()),
		Forwarder: &utils.IdEmailModel{
			Id:    utils.TrimmedStringValue(responseSchema.Forwarder.Id.String()),
			Email: utils.TrimmedStringValue(string(responseSchema.Forwarder.Email)),
		},
		Target: &utils.IdEmailModel{
			Id:    utils.TrimmedStringValue(responseSchema.Target.Id.String()),
			Email: utils.TrimmedStringValue(string(responseSchema.Target.Email)),
		},
	}

	// Save the converted data into Terraform state
	diagsResult := resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diagsResult...)
	if resp.Diagnostics.HasError() {
		return
	}
}
