// Package agentTokens provides the implementation of the Entitle Agent Token resource for Terraform.
// It defines the resource type, its schema, and the CRUD operations for managing Agent Tokens.
package agentTokens

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/entitleio/terraform-provider-entitle/docs"
	"github.com/entitleio/terraform-provider-entitle/internal/client"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/utils"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &AgentTokenResource{}
var _ resource.ResourceWithImportState = &AgentTokenResource{}

// rotationPath is where this resource keeps its rotation trigger. The token
// plan modifier reads it to decide whether a rotation is pending.
var rotationPath = path.Root(utils.DefaultRotationAttributeName)

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
	ID       types.String `tfsdk:"id"`
	Name     types.String `tfsdk:"name"`
	Token    types.String `tfsdk:"token"`
	Rotation types.String `tfsdk:"rotation"`
}

// Metadata sets the metadata for the resource.
func (r *AgentTokenResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_agent_token"
}

// Schema sets the schema for the resource.
func (r *AgentTokenResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: docs.AgentTokenResourceMarkdownDescription,
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
				MarkdownDescription: "The display name for the agent token.",
				Description:         "The display name for the agent token.",
			},
			"token": schema.StringAttribute{
				Computed:            true,
				Sensitive:           true,
				MarkdownDescription: "The token for the agent token. (sensitive)",
				Description:         "The token for the agent token. (sensitive)",
				// Carries the secret forward from state, because the API only
				// returns it on create and on rotate, except when a rotation is
				// pending, in which case it is planned as unknown and shows up
				// in the plan as "(sensitive value)".
				PlanModifiers: utils.RotatableSecretPlanModifiers(rotationPath),
			},
			utils.DefaultRotationAttributeName: utils.RotationSchemaAttribute("token"),
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
			"Failed to create agent token resource; required name variable missing",
		)
		return
	}

	// Send a request to the Entitle API to create the agent token.
	agentTokenResp, err := r.client.AgentTokensCreateWithResponse(ctx, client.AgentTokenCreateBodySchema{
		Name: name,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiConnection.Error(),
			fmt.Sprintf("Unable to create the agent token, got error: %v", err),
		)
		return
	}

	err = utils.HTTPResponseToError(agentTokenResp.HTTPResponse.StatusCode, agentTokenResp.Body)
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiResponse.Error(),
			fmt.Sprintf(
				"Failed to create the Agent Token, status code: %d, %s",
				agentTokenResp.HTTPResponse.StatusCode,
				err.Error(),
			),
		)
		return
	}

	if agentTokenResp.JSON200 == nil || agentTokenResp.JSON200.Result == nil {
		resp.Diagnostics.AddError(
			utils.ErrApiResponse.Error(),
			"The Agent Token create response contained no result",
		)
		return
	}

	// Write logs using the tflog package.
	tflog.Trace(ctx, "created an Entitle agent token resource")

	// Update the AgentTokenResourceModel with the created agent token data.
	// Rotation is a client-side trigger with no server representation, so it is
	// carried through from the plan untouched.
	plan = AgentTokenResourceModel{
		ID:       utils.TrimmedStringValue(agentTokenResp.JSON200.Result.Id.String()),
		Name:     utils.TrimmedStringValue(name),
		Token:    utils.TrimmedStringValue(agentTokenResp.JSON200.Result.Token),
		Rotation: plan.Rotation,
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
	uid, err := uuid.Parse(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Failed to parse the resource id (%s) to UUID, got error: %s", data.ID.ValueString(), err),
		)
		return
	}

	// Retrieve the agent token details from the Entitle API.
	agentTokenResp, err := r.client.AgentTokensShowWithResponse(ctx, uid)
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiConnection.Error(),
			fmt.Sprintf("Unable to get the agent token by the id (%s), got error: %s", uid.String(), err),
		)
		return
	}

	err = utils.HTTPResponseToError(agentTokenResp.HTTPResponse.StatusCode, agentTokenResp.Body)
	if err != nil {
		if errors.Is(err, utils.ErrNotFound) {
			tflog.Debug(ctx, "Resource no longer exists, removing from state")

			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			utils.ErrApiResponse.Error(),
			fmt.Sprintf(
				"Failed to get the Agent Token by the id (%s), status code: %d, %s",
				uid.String(),
				agentTokenResp.HTTPResponse.StatusCode,
				err.Error(),
			),
		)
		return
	}

	if agentTokenResp.JSON200 == nil || agentTokenResp.JSON200.Result == nil {
		resp.Diagnostics.AddError(
			utils.ErrApiResponse.Error(),
			fmt.Sprintf("The Agent Token read response for the id (%s) contained no result", uid.String()),
		)
		return
	}

	// Update the AgentTokenResourceModel with the retrieved data.
	//
	// The API never returns the secret on read, and rotation has no server-side
	// representation, so both are preserved from prior state. Overwriting either
	// here would produce a permanent diff.
	data = AgentTokenResourceModel{
		ID:       utils.TrimmedStringValue(agentTokenResp.JSON200.Result.Id.String()),
		Name:     utils.TrimmedStringValue(agentTokenResp.JSON200.Result.Name),
		Token:    data.Token,
		Rotation: data.Rotation,
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
//
// Two independent changes are handled here:
//
//   - a change to "name" is a PUT against /agentTokens/{id};
//   - a change to "rotation" is a POST against /agentTokens/{id}/rotate, which
//     issues a new secret for the same token. The resource id, and therefore any
//     grant or integration that references it, is unchanged.
//
// Both can happen in the same apply; the rename runs first. Neither response
// body is trusted for "name", which is Required and therefore always written
// back from the plan.
func (r *AgentTokenResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Create an instance of the AgentTokenResourceModel to store the resource data.
	var data, state AgentTokenResourceModel

	// Read Terraform plan data into the model.
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read prior state as well; rotation is decided by comparing the two.
	diags = req.State.Get(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Parse the unique identifier from the resource data.
	uid, err := uuid.Parse(data.ID.ValueString())
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

	// The secret is not returned by the update endpoint, so it defaults to the
	// value already in state and is only replaced if a rotation happens below.
	token := state.Token

	if !data.Name.Equal(state.Name) {
		// Send a request to the Entitle API to update the agent token.
		agentTokenResp, err := r.client.AgentTokensUpdateWithResponse(ctx, uid, client.AgentTokenCreateBodySchema{
			Name: name,
		})
		if err != nil {
			resp.Diagnostics.AddError(
				utils.ErrApiConnection.Error(),
				fmt.Sprintf("Unable to update agent token by the id (%s), got error: %s", uid.String(), err),
			)
			return
		}

		err = utils.HTTPResponseToError(agentTokenResp.HTTPResponse.StatusCode, agentTokenResp.Body)
		if err != nil {
			if errors.Is(err, utils.ErrNotFound) {
				tflog.Debug(ctx, "Resource no longer exists, removing from state")

				resp.State.RemoveResource(ctx)
				return
			}

			resp.Diagnostics.AddError(
				utils.ErrApiResponse.Error(),
				fmt.Sprintf(
					"Failed to update the Agent Token by the id (%s), status code: %d, %s",
					uid.String(),
					agentTokenResp.HTTPResponse.StatusCode,
					err.Error(),
				),
			)
			return
		}
	}

	if utils.RotationChanged(state.Rotation, data.Rotation) {
		tflog.Debug(ctx, "rotating an Entitle agent token", map[string]any{"id": uid.String()})

		rotateResp, err := r.client.AgentTokensRotateWithResponse(ctx, uid)
		if err != nil {
			resp.Diagnostics.AddError(
				utils.ErrApiConnection.Error(),
				fmt.Sprintf("Unable to rotate agent token by the id (%s), got error: %s", uid.String(), err),
			)
			return
		}

		err = utils.HTTPResponseToError(rotateResp.HTTPResponse.StatusCode, rotateResp.Body)
		if err != nil {
			if errors.Is(err, utils.ErrNotFound) {
				tflog.Debug(ctx, "Resource no longer exists, removing from state")

				resp.State.RemoveResource(ctx)
				return
			}

			resp.Diagnostics.AddError(
				utils.ErrApiResponse.Error(),
				fmt.Sprintf(
					"Failed to rotate the Agent Token by the id (%s), status code: %d, %s",
					uid.String(),
					rotateResp.HTTPResponse.StatusCode,
					err.Error(),
				),
			)
			return
		}

		// A rotation that reports success but returns no secret would silently
		// leave the old value in state, and the next apply would see no pending
		// change while the agent is already locked out. Fail loudly instead.
		if rotateResp.JSON200 == nil || rotateResp.JSON200.Result == nil || rotateResp.JSON200.Result.Token == "" {
			resp.Diagnostics.AddError(
				utils.ErrApiResponse.Error(),
				fmt.Sprintf(
					"The Agent Token (%s) was rotated but the API returned no new token value. "+
						"The old token is no longer valid and the new one cannot be recovered; "+
						"taint or recreate the resource to obtain a usable token.",
					uid.String(),
				),
			)
			return
		}

		token = utils.TrimmedStringValue(rotateResp.JSON200.Result.Token)

		tflog.Trace(ctx, "rotated an Entitle agent token resource")
	}

	// Update the AgentTokenResourceModel with the updated agent token data.
	//
	// "name" is taken from the plan rather than from either response body: it is
	// a Required attribute, so writing anything else here would fail the apply
	// with "Provider produced inconsistent result after apply".
	data = AgentTokenResourceModel{
		ID:       utils.TrimmedStringValue(uid.String()),
		Name:     data.Name,
		Token:    token,
		Rotation: data.Rotation,
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
	uid, err := uuid.Parse(data.ID.ValueString())
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
			utils.ErrApiConnection.Error(),
			fmt.Sprintf("Unable to delete agent token, id: (%s), got error: %v", uid.String(), err),
		)
		return
	}

	err = utils.HTTPResponseToError(httpResp.HTTPResponse.StatusCode, httpResp.Body, utils.WithIgnoreNotFound())
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiResponse.Error(),
			fmt.Sprintf(
				"Failed to delete the Agent by the id (%s), status code: %d, %s",
				uid.String(),
				httpResp.HTTPResponse.StatusCode,
				err.Error(),
			),
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
