package accessReviewForwards

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/entitleio/terraform-provider-entitle/internal/provider/utils"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

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
		MarkdownDescription: "Entitle Access Review Forward allows delegating access review responsibilities to another user. " +
			"This enables review tasks to be reassigned when the original reviewer is unavailable. " +
			"[Read more about access reviews](https://docs.beyondtrust.com/entitle/docs/access-review).",
		Description: "Entitle Access Review Forward allows delegating access review responsibilities to another user. " +
			"This enables review tasks to be reassigned when the original reviewer is unavailable. " +
			"[Read more about access reviews](https://docs.beyondtrust.com/entitle/docs/access-review).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Entitle Access Review Forwardidentifier in uuid format",
				Description:         "Entitle Access Review Forwardidentifier in uuid format",
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
			fmt.Sprintf("failed to parse the resource id (%s) to UUID, got error: %s", data.ID.String(), err),
		)
		return
	}

	// Log the start of the access review forward GET operation with the resource ID
	tflog.Debug(ctx, "run access review forward GET by id", map[string]interface{}{
		"uid": uid.String(),
	})

	// Send a request to the Entitle API to get the access review forward data by ID
	apiResp, err := d.client.AccessReviewForwardsShowWithResponse(ctx, uid)
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to get the access review forward by the id (%s), got error: %s", uid.String(), err),
		)
		return
	}

	// Check the HTTP response status code for errors
	if apiResp.HTTPResponse.StatusCode != 200 {
		errBody, _ := utils.GetErrorBody(apiResp.Body)
		if apiResp.HTTPResponse.StatusCode == http.StatusUnauthorized ||
			(apiResp.HTTPResponse.StatusCode == http.StatusBadRequest && strings.Contains(errBody.GetMessage(), "is not a valid uuid")) {
			resp.Diagnostics.AddError(
				"Client Error",
				"unauthorized token, update the entitle token and retry please",
			)
			return
		}

		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf(
				"failed to get the access review forward by the id (%s), status code: %d%s",
				uid.String(),
				apiResp.HTTPResponse.StatusCode,
				errBody.GetMessage(),
			),
		)
		return
	}

	responseSchema := apiResp.JSON200.Result[0]
	forwarderEmailBytes, err := responseSchema.Forwarder.Email.MarshalJSON()
	if err != nil {
		resp.Diagnostics.AddError(
			"No data",
			fmt.Sprintf("failed to get forwarder user email bytes, error: %v", err),
		)

		return
	}

	targetEmailBytes, err := responseSchema.Forwarder.Email.MarshalJSON()
	if err != nil {
		resp.Diagnostics.AddError(
			"No data",
			fmt.Sprintf("failed to get target user email bytes, error: %v", err),
		)

		return
	}

	data = AccessReviewForwardDataSourceModel{
		ID: utils.TrimmedStringValue(responseSchema.Id.String()),
		Forwarder: &utils.IdEmailModel{
			Id:    utils.TrimmedStringValue(responseSchema.Forwarder.Id.String()),
			Email: utils.TrimmedStringValue(string(forwarderEmailBytes)),
		},
		Target: &utils.IdEmailModel{
			Id:    utils.TrimmedStringValue(responseSchema.Target.Id.String()),
			Email: utils.TrimmedStringValue(string(targetEmailBytes)),
		},
	}

	// Save the converted data into Terraform state
	diagsResult := resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diagsResult...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Log a trace indicating the successful saving of the data source
	tflog.Trace(ctx, "saved entitle access review forward data source successfully!")
}
