// Package agentTokens provides the implementation of the Entitle Agent Token resource for Terraform.
// It defines the resource type, its schema, and the CRUD operations for managing Agent Tokens.
package agentTokens

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
var _ resource.Resource = &AgentTokenResource{}
var _ resource.ResourceWithImportState = &AgentTokenResource{}

// NewAgentTokenResource creates a new instance of the AgentTokenResource.
func NewAgentTokenResource() resource.Resource {
	return &AgentTokenResource{}
}

// AgentTokenResource defines the resource implementation.
type AgentTokenResource struct {
	client *client.ClientWithResponses
}

// AgentTokenResourceModel describes the resource data model.
type AgentTokenResourceModel struct {
	ID    types.String `tfsdk:"id"`
	Name  types.String `tfsdk:"name"`
	Token types.String `tfsdk:"token"`
}

// Metadata sets the metadata for the resource.
func (r *AgentTokenResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_agent_token"
}

// Schema sets the schema for the resource.
func (r *AgentTokenResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Defines the schema for an Entitle Agent Token resource. " +
			"[Read more about agents](https://docs.beyondtrust.com/entitle/docs/entitle-agent).",
		Description: "Defines the schema for an Entitle Agent Token resource. " +
			"[Read more about agents](https://docs.beyondtrust.com/entitle/docs/entitle-agent).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Entitle AgentToken identifier in UUID format",
				Description:         "Entitle AgentToken identifier in UUID format",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				Optional:            false,
				MarkdownDescription: "The display name for the agent token.",
				Description:         "The display name for the agent token.",
			},
			"token": schema.StringAttribute{
				Required:            false,
				Computed:            true,
				Optional:            true,
				Sensitive:           true,
				MarkdownDescription: "The token for the agent token. (sensitive)",
				Description:         "The token for the agent token. (sensitive)",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Configure configures the resource with the provided client.
func (r *AgentTokenResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.ClientWithResponses)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = c
}

// Create handles the creation of a new resource of type Entitle AgentToken.
//
// It reads the Terraform plan data, maps it to the AgentTokenResourceModel,
// sends a request to the Entitle API to create the resource, and saves the
// resource's data into Terraform state.
func (r *AgentTokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Create an instance of the AgentTokenResourceModel to store the resource data.
	var plan AgentTokenResourceModel

	// Read Terraform plan data into the model.
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Extract the name from the plan, required for creating the agent token.
	name := plan.Name.ValueString()
	if name == "" {
		resp.Diagnostics.AddError(
			"Client Error",
			"failed to create agent token resource; required name variable missing",
		)
		return
	}

	// Send a request to the Entitle API to create the agent token.
	agentTokenResp, err := r.client.AgentTokensCreateWithResponse(ctx, client.AgentTokenCreateBodySchema{
		Name: name,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to create the agent token, got error: %v", err),
		)
		return
	}

	// Check if the API request was successful.
	if agentTokenResp.StatusCode() != 200 || agentTokenResp.JSON200 == nil {
		errBody, _ := utils.GetErrorBody(agentTokenResp.Body)
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf(
				"failed to create the agent token, status code: %d%s",
				agentTokenResp.HTTPResponse.StatusCode,
				errBody.GetMessage(),
			),
		)
		return
	}

	// Write logs using the tflog package.
	tflog.Trace(ctx, "created an Entitle agent token resource")

	// Update the AgentTokenResourceModel with the created agent token data.
	plan = AgentTokenResourceModel{
		ID:    utils.TrimmedStringValue(agentTokenResp.JSON200.Id.String()),
		Name:  utils.TrimmedStringValue(name),
		Token: utils.TrimmedStringValue(agentTokenResp.JSON200.Token),
	}

	// Save the data into Terraform state.
	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read retrieves an existing resource of type Entitle AgentToken.
//
// It retrieves the resource's data from the provider API requests,
// maps it to the AgentTokenResourceModel, and saves the data to Terraform state.
func (r *AgentTokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Create an instance of the AgentTokenResourceModel to store the resource data.
	var data AgentTokenResourceModel

	// Read Terraform prior state data into the model.
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Parse the resource ID into a UUID for API request.
	uid, err := uuid.Parse(data.ID.String())
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("failed to parse the resource id (%s) to UUID, got error: %s", data.ID.String(), err),
		)
		return
	}

	// Retrieve the agent token details from the Entitle API.
	agentTokenResp, err := r.client.AgentTokensShowWithResponse(ctx, uid)
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to get the agent token by the id (%s), got error: %s", uid.String(), err),
		)
		return
	}

	// Check if the API request was successful.
	if agentTokenResp.StatusCode() != 200 || agentTokenResp.JSON200 == nil {
		errBody, _ := utils.GetErrorBody(agentTokenResp.Body)
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf(
				"failed to get the agent token by the id (%s), status code: %d%s",
				uid.String(),
				agentTokenResp.StatusCode(),
				errBody.GetMessage(),
			),
		)
		return
	}

	// Update the AgentTokenResourceModel with the retrieved data.
	data = AgentTokenResourceModel{
		ID:    utils.TrimmedStringValue(agentTokenResp.JSON200.Id.String()),
		Name:  utils.TrimmedStringValue(agentTokenResp.JSON200.Name),
		Token: data.Token,
	}

	// Save the updated data into Terraform state.
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update handles updates to an existing resource of type Entitle AgentToken.
//
// It reads the updated Terraform plan data, sends a request to the Entitle API
// to update the resource, and saves the updated resource data into Terraform state.
func (r *AgentTokenResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Create an instance of the AgentTokenResourceModel to store the resource data.
	var data AgentTokenResourceModel

	// Read Terraform plan data into the model.
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Parse the unique identifier from the resource data.
	uid, err := uuid.Parse(data.ID.String())
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("failed to parse the given id to UUID format, got error: %v", err),
		)
		return
	}

	// Extract the name from the resource data; it's required for updating the agent token.
	var name string
	if data.Name.ValueString() == "" {
		resp.Diagnostics.AddError(
			"Client Error",
			"missing the name variable for Entitle agent token",
		)
		return
	}

	name = data.Name.ValueString()

	// Send a request to the Entitle API to update the agent token.
	agentTokenResp, err := r.client.AgentTokensUpdateWithResponse(ctx, uid, client.AgentTokenCreateBodySchema{
		Name: name,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to update agent token by the id (%s), got error: %s", uid.String(), err),
		)
		return
	}

	// Check if the API request was successful.
	if agentTokenResp.StatusCode() != 200 || agentTokenResp.JSON200 == nil {
		errBody, _ := utils.GetErrorBody(agentTokenResp.Body)
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf(
				"failed to update the agent token by the id (%s), status code: %d%s",
				uid.String(),
				agentTokenResp.StatusCode(),
				errBody.GetMessage(),
			),
		)
		return
	}

	// Update the AgentTokenResourceModel with the updated agent token data.
	data = AgentTokenResourceModel{
		ID:    utils.TrimmedStringValue(agentTokenResp.JSON200.Id.String()),
		Name:  utils.TrimmedStringValue(agentTokenResp.JSON200.Name),
		Token: data.Token,
	}

	// Save the updated data into Terraform state.
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete is responsible for deleting an existing resource of type Entitle AgentToken.
//
// It reads the resource's data from Terraform state, extracts the unique identifier,
// and sends a request to delete the resource using API requests. If the deletion
// is successful, it removes the resource from Terraform state.
func (r *AgentTokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Create an instance of the AgentTokenResourceModel to store the resource data.
	var data AgentTokenResourceModel

	// Read Terraform prior state data into the model.
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	// Check for errors in reading Terraform state data.
	if resp.Diagnostics.HasError() {
		return
	}

	// Parse the unique identifier from the resource data.
	uid, err := uuid.Parse(data.ID.String())
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("failed to parse the given id to UUID format, got error: %v", err),
		)
		return
	}

	// Send a request to the Entitle API to delete the agent token.
	httpResp, err := r.client.AgentTokensDestroyWithResponse(ctx, uid)
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to delete agent token, id: (%s), got error: %v", data.ID.String(), err),
		)
		return
	}

	// Check if the API request was successful.
	if httpResp.HTTPResponse.StatusCode != 200 {
		errBody, _ := utils.GetErrorBody(httpResp.Body)
		if httpResp.HTTPResponse.StatusCode == http.StatusUnauthorized ||
			(httpResp.HTTPResponse.StatusCode == http.StatusBadRequest && strings.Contains(errBody.GetMessage(), "is not a valid uuid")) {
			resp.Diagnostics.AddError(
				"Client Error",
				"unauthorized token, update the entitle token and retry please",
			)
			return
		}

		if httpResp.HTTPResponse.StatusCode == http.StatusUnauthorized ||
			httpResp.HTTPResponse.StatusCode == http.StatusBadRequest {
			resp.Diagnostics.AddError(
				"Client Error",
				"unauthorized token, update the entitle token and retry please",
			)
			return
		}

		if errBody.ID == "resource.notFound" {
			return
		}

		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to delete agent token, id: (%s), status code: %v%s", data.ID.String(), httpResp.HTTPResponse.StatusCode, errBody.GetMessage()),
		)
		return
	}
}

// ImportState is used to import an existing resource's state into Terraform.
//
// It extracts the resource's identifier from the import request and sets
// it in Terraform state using resource.ImportStatePassthroughID.
func (r *AgentTokenResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
