package accessRequestForwards

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/entitleio/terraform-provider-entitle/docs"
	"github.com/entitleio/terraform-provider-entitle/internal/client"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/utils"
	"github.com/entitleio/terraform-provider-entitle/internal/validators"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &AccessRequestForwardResource{}
var _ resource.ResourceWithImportState = &AccessRequestForwardResource{}

func NewAccessRequestForwardResource() resource.Resource {
	return &AccessRequestForwardResource{}
}

// AccessRequestForwardResource defines the resource implementation.
type AccessRequestForwardResource struct {
	client *client.ClientWithResponses
}

// AccessRequestForwardResourceModel describes the resource data model.
type AccessRequestForwardResourceModel struct {
	ID        types.String        `tfsdk:"id" json:"id"`
	Forwarder *utils.IdEmailModel `tfsdk:"forwarder" json:"forwarder"`
	Target    *utils.IdEmailModel `tfsdk:"target" json:"target"`
}

// Metadata is a function to set the TypeName for the Entitle Access Request Forward resource.
func (r *AccessRequestForwardResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_access_request_forward"
}

func (r *AccessRequestForwardResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: docs.AccessRequestForwardResourceMarkdownDescription,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Entitle Access Request Forward identifier in uuid format",
				Description:         "Entitle Access Request Forward identifier in uuid format",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"forwarder": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "the forwarder user's identifier in uuid format",
						MarkdownDescription: "the forwarder user's identifier in uuid format",
						Validators: []validator.String{
							validators.UUID{},
						},
					},
					"email": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "the forwarder user's email address",
						MarkdownDescription: "the forwarder user's email address",
						Validators: []validator.String{
							validators.Email{},
						},
					},
				},
				Required: true,
				Description: "Specifies the user who is delegating or forwarding their access request responsibilities. " +
					"This user must have request permissions for the items being forwarded.",
				MarkdownDescription: "Specifies the user who is delegating or forwarding their access request responsibilities. " +
					"This user must have request permissions for the items being forwarded.",
			},
			"target": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "the target user's identifier in uuid format",
						MarkdownDescription: "the target user's identifier in uuid format",
						Validators: []validator.String{
							validators.UUID{},
						},
					},
					"email": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "the target user's email address",
						MarkdownDescription: "the target user's email address",
						Validators: []validator.String{
							validators.Email{},
						},
					},
				},
				Required: true,
				Description: "Defines the user who will receive and be responsible for completing the forwarded access request tasks. " +
					"This user will temporarily assume the request responsibilities for the specified items.",
				MarkdownDescription: "Defines the user who will receive and be responsible for completing the forwarded access request tasks. " +
					"This user will temporarily assume the request responsibilities for the specified items.",
			},
		},
	}
}

// Configure is a function to set the client configuration for the AccessRequestForwardResource.
func (r *AccessRequestForwardResource) Configure(
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

// Create is responsible for creating a new resource of type Entitle Access Request Forward.
//
// It reads the Terraform plan data provided in req.Plan and maps it to the AccessRequestForwardResourceModel.
// Then, it sends a request to the Entitle API to create the resource using API requests.
// If the creation is successful, it saves the resource's data into Terraform state.
func (r *AccessRequestForwardResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var err error
	var plan AccessRequestForwardResourceModel

	// Read Terraform plan data into the model
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	forwarderID := plan.Forwarder.Id.ValueString()
	if forwarderID == "" {
		forwarderID = plan.Forwarder.Email.ValueString()
	}

	if forwarderID == "" {
		resp.Diagnostics.AddError(
			"Client Error",
			"Forwarder id not provided",
		)
		return
	}

	targetID := plan.Target.Id.ValueString()
	if targetID == "" {
		targetID = plan.Target.Email.ValueString()
	}

	if targetID == "" {
		resp.Diagnostics.AddError(
			"Client Error",
			"Target id not provided",
		)
		return
	}

	if targetID == forwarderID {
		resp.Diagnostics.AddError(
			"Client Error",
			"Target id is the same as Forwarder id",
		)
		return
	}

	// Call Entitle API to create the resource
	apiResp, err := r.client.AccessRequestForwardsCreateWithResponse(ctx, client.AccessRequestForwardsCreateJSONRequestBody{
		Forwarder: client.UserEntitySchema{
			Id: forwarderID,
		},
		Target: client.UserEntitySchema{
			Id: targetID,
		},
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to create the access request forward, got error: %v", err),
		)
		return
	}

	err = utils.HTTPResponseToError(apiResp.HTTPResponse.StatusCode, apiResp.Body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf(
				"Failed to create the Access Request Forward, %s, status code: %d, %s",
				string(apiResp.Body),
				apiResp.HTTPResponse.StatusCode,
				err.Error(),
			))
		return
	}

	tflog.Trace(ctx, "created an Entitle Access Request Forward resource")

	// Update the plan with the new resource ID
	plan.ID = utils.TrimmedStringValue(apiResp.JSON200.Result.Id.String())

	plan.Forwarder.Id = utils.TrimmedStringValue(apiResp.JSON200.Result.Forwarder.Id.String())
	plan.Forwarder.Email = utils.GetEmailStringValue(apiResp.JSON200.Result.Forwarder.Email)

	plan.Target.Id = utils.TrimmedStringValue(apiResp.JSON200.Result.Target.Id.String())
	plan.Target.Email = utils.GetEmailStringValue(apiResp.JSON200.Result.Target.Email)

	// Save data into Terraform state
	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read is used to read an existing resource of type Entitle Access Request Forward.
//
// It retrieves the resource's data from the provider API requests.
// The retrieved data is then mapped to the AccessRequestForwardResourceModel,
// and the data is saved to Terraform state.
func (r *AccessRequestForwardResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var data AccessRequestForwardResourceModel

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
	apiResp, err := r.client.AccessRequestForwardsShowWithResponse(ctx, uid)
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiConnection.Error(),
			fmt.Sprintf("Unable to get the access request forward by the id (%s), got error: %s", uid.String(), err),
		)
		return
	}

	err = utils.HTTPResponseToError(apiResp.HTTPResponse.StatusCode, apiResp.Body)
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiResponse.Error(),
			fmt.Sprintf(
				"Failed to get the Access Request Forward by the id (%s), status code: %d, %s",
				uid.String(),
				apiResp.HTTPResponse.StatusCode,
				err.Error(),
			),
		)
		return
	}

	responseSchema := apiResp.JSON200.Result

	data = AccessRequestForwardResourceModel{
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

	// Save updated data into Terraform state
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update handles updates to an existing resource of type Entitle Access Request Forward.
//
// It reads the updated Terraform plan data provided in req.Plan and maps it to the AccessRequestForwardResourceModel.
// Then, it sends a request to the Entitle API to update the resource using API requests.
// If the update is successful, it saves the updated resource data into Terraform state.
func (r *AccessRequestForwardResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var data AccessRequestForwardResourceModel

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
}

// Delete is responsible for deleting an existing resource of type Entitle Access Request Forward.
//
// It reads the resource's data from Terraform state, extracts the unique identifier,
// and sends a request to delete the resource using API requests.
// If the deletion is successful, it removes the resource from Terraform state.
func (r *AccessRequestForwardResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var data AccessRequestForwardResourceModel

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
			fmt.Sprintf("Unable to parse uuid of the access request forward, id: (%s), got error: %v", data.ID.String(), err),
		)
		return
	}

	// Call Entitle API to delete the access request forward resource
	httpResp, err := r.client.AccessRequestForwardsDestroyWithResponse(ctx, parsedUUID)
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiConnection.Error(),
			fmt.Sprintf("Unable to delete access request forward, id: (%s), got error: %v", data.ID.String(), err),
		)
		return
	}

	err = utils.HTTPResponseToError(httpResp.HTTPResponse.StatusCode, httpResp.Body)
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiResponse.Error(),
			fmt.Sprintf(
				"Unable to delete Access Request Forward, id: (%s), status code: %v, %s",
				data.ID.String(),
				httpResp.HTTPResponse.StatusCode,
				err.Error(),
			),
		)
		return
	}
}

// ImportState this function is used to import an existing resource's state into Terraform.
//
// It extracts the resource's identifier from the import request and sets
// it in Terraform state using resource.ImportStatePassthroughID.
func (r *AccessRequestForwardResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
