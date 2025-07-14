package accessReviewForwards

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/entitleio/terraform-provider-entitle/internal/client"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/utils"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &AccessReviewForwardResource{}
var _ resource.ResourceWithImportState = &AccessReviewForwardResource{}

func NewAccessReviewForwardResource() resource.Resource {
	return &AccessReviewForwardResource{}
}

// AccessReviewForwardResource defines the resource implementation.
type AccessReviewForwardResource struct {
	client *client.ClientWithResponses
}

// AccessReviewForwardResourceModel describes the resource data model.
type AccessReviewForwardResourceModel struct {
	ID        types.String        `tfsdk:"id" json:"id"`
	Forwarder *utils.IdEmailModel `tfsdk:"forwarder" json:"forwarder"`
	Target    *utils.IdEmailModel `tfsdk:"target" json:"target"`
}

// Metadata is a function to set the TypeName for the Entitle access review forward resource.
func (r *AccessReviewForwardResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_access_review_forward"
}

func (r *AccessReviewForwardResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Entitle Access Review Forward allows delegating access review responsibilities to another user. " +
			"This enables review tasks to be reassigned when the original reviewer is unavailable. " +
			"[Read more about access reviews](https://docs.beyondtrust.com/entitle/docs/access-review).",
		Description: "Entitle Access Review Forward allows delegating access review responsibilities to another user. " +
			"This enables review tasks to be reassigned when the original reviewer is unavailable. " +
			"[Read more about access reviews](https://docs.beyondtrust.com/entitle/docs/access-review).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Entitle Access Review Forward identifier in uuid format",
				Description:         "Entitle Access Review Forward identifier in uuid format",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"forwarder": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Required:            false,
						Optional:            true,
						Description:         "the forwarder user's identifier in uuid format",
						MarkdownDescription: "the forwarder user's identifier in uuid format",
					},
					"email": schema.StringAttribute{
						Computed:            true,
						Description:         "the forwarder user's email address",
						MarkdownDescription: "the forwarder user's email address",
					},
				},
				Required: false,
				Optional: true,
				Description: "Specifies the user who is delegating or forwarding their access review responsibilities. " +
					"This user must have review permissions for the items being forwarded.",
				MarkdownDescription: "Specifies the user who is delegating or forwarding their access review responsibilities. " +
					"This user must have review permissions for the items being forwarded.",
			},
			"target": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Required:            false,
						Optional:            true,
						Description:         "the taget user's identifier in uuid format",
						MarkdownDescription: "the taget user's identifier in uuid format",
					},
					"email": schema.StringAttribute{
						Computed:            true,
						Description:         "the taget user's email address",
						MarkdownDescription: "the taget user's email address",
					},
				},
				Required: false,
				Optional: true,
				Description: "Defines the user who will receive and be responsible for completing the forwarded access review tasks. " +
					"This user will temporarily assume the review responsibilities for the specified items.",
				MarkdownDescription: "Defines the user who will receive and be responsible for completing the forwarded access review tasks. " +
					"This user will temporarily assume the review responsibilities for the specified items.",
			},
		},
	}
}

// Configure is a function to set the client configuration for the AccessReviewForwardResource.
func (r *AccessReviewForwardResource) Configure(
	ctx context.Context,
	req resource.ConfigureRequest,
	resp *resource.ConfigureResponse,
) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	cli, ok := req.ProviderData.(*client.ClientWithResponses)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = cli
}

// Create is responsible for creating a new resource of type Entitle Access Review Forward.
//
// It reads the Terraform plan data provided in req.Plan and maps it to the AccessReviewForwardResourceModel.
// Then, it sends a request to the Entitle API to create the resource using API requests.
// If the creation is successful, it saves the resource's data into Terraform state.
func (r *AccessReviewForwardResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var err error
	var plan AccessReviewForwardResourceModel

	// Read Terraform plan data into the model
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Call Entitle API to create the resource
	apiResp, err := r.client.AccessReviewForwardsCreateWithResponse(ctx, client.AccessReviewForwardsCreateJSONRequestBody{
		Forwarder: client.UserEntitySchema{
			Id: plan.Forwarder.Id.ValueString(),
		},
		Target: client.UserEntitySchema{
			Id: plan.Target.Id.ValueString(),
		},
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to create the access review forward, got error: %v", err),
		)
		return
	}

	err = utils.HTTPResponseToError(apiResp.HTTPResponse.StatusCode, apiResp.Body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to create the Access Review Forward, got error: %s", err),
		)
		return
	}

	tflog.Trace(ctx, "created an Entitle Access Review Forward resource")

	// Update the plan with the new resource ID
	plan.ID = utils.TrimmedStringValue(apiResp.JSON200.Result[0].Id.String())

	// Save data into Terraform state
	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read is used to read an existing resource of type Entitle Access Review Forward.
//
// It retrieves the resource's data from the provider API requests.
// The retrieved data is then mapped to the AccessReviewForwardResourceModel,
// and the data is saved to Terraform state.
func (r *AccessReviewForwardResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var data AccessReviewForwardResourceModel

	// Read Terraform prior state data into the model
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Parse the resource ID from the model
	uid, err := uuid.Parse(data.ID.String())
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Failed to parse the resource id (%s) to UUID, got error: %s", data.ID.String(), err),
		)
		return
	}

	// Call Entitle API to get the resource by ID
	apiResp, err := r.client.AccessReviewForwardsShowWithResponse(ctx, uid)
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to get the access review forward by the id (%s), got error: %s", uid.String(), err),
		)
		return
	}

	// Handle API response status
	if apiResp.HTTPResponse.StatusCode != 200 {
		errBody, _ := utils.GetErrorBody(apiResp.Body)
		if apiResp.HTTPResponse.StatusCode == http.StatusUnauthorized ||
			(apiResp.HTTPResponse.StatusCode == http.StatusBadRequest && strings.Contains(errBody.GetMessage(), "is not a valid uuid")) {
			resp.Diagnostics.AddError(
				"Client Error",
				"Unauthorized token, update the entitle token and retry please",
			)
			return
		}

		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf(
				"Failed to get the access review forward by the id (%s), status code: %d%s",
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
		diags.AddError(
			"No data",
			fmt.Sprintf("Failed to get forwarder user email bytes, error: %v", err),
		)

		return
	}

	targetEmailBytes, err := responseSchema.Forwarder.Email.MarshalJSON()
	if err != nil {
		diags.AddError(
			"No data",
			fmt.Sprintf("Failed to get target user email bytes, error: %v", err),
		)

		return
	}

	data = AccessReviewForwardResourceModel{
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

	// Save updated data into Terraform state
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update handles updates to an existing resource of type Entitle Access Review Forward.
//
// It reads the updated Terraform plan data provided in req.Plan and maps it to the AccessReviewForwardResourceModel.
// Then, it sends a request to the Entitle API to update the resource using API requests.
// If the update is successful, it saves the updated resource data into Terraform state.
func (r *AccessReviewForwardResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var data AccessReviewForwardResourceModel

	// Read Terraform plan data into the model
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Parse the resource ID from the model
	uid, err := uuid.Parse(data.ID.String())
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Failed to parse the resource id (%s) to UUID, got error: %s", data.ID.String(), err),
		)
		return
	}

	resp.Diagnostics.AddError(
		"Client Error",
		fmt.Sprintf("Update not available for the resource id (%s)", uid.String()),
	)
	return
}

// Delete is responsible for deleting an existing resource of type Entitle Access Review Forward.
//
// It reads the resource's data from Terraform state, extracts the unique identifier,
// and sends a request to delete the resource using API requests.
// If the deletion is successful, it removes the resource from Terraform state.
func (r *AccessReviewForwardResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var data AccessReviewForwardResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Parse the resource ID from the model
	parsedUUID, err := uuid.Parse(data.ID.String())
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to parse uuid of the access review forward, id: (%s), got error: %v", data.ID.String(), err),
		)
		return
	}

	// Call Entitle API to delete the access review forward resource
	httpResp, err := r.client.AccessReviewForwardsDestroyWithResponse(ctx, parsedUUID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to delete access review forward, id: (%s), got error: %v", data.ID.String(), err),
		)
		return
	}

	err = utils.HTTPResponseToError(httpResp.HTTPResponse.StatusCode, httpResp.Body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf(
				"Unable to delete Access Review Forward, id: (%s), status code: %v, %s",
				data.ID.String(),
				httpResp.HTTPResponse.StatusCode,
				err.Error(),
			))
		return
	}
}

// ImportState this function is used to import an existing resource's state into Terraform.
//
// It extracts the resource's identifier from the import request and sets
// it in Terraform state using resource.ImportStatePassthroughID.
func (r *AccessReviewForwardResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
