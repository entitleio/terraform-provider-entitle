package workflowWebhooks

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
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
)

// httpsURLPattern matches HTTPS URLs. The Entitle API rejects webhook URLs that are
// not served over HTTPS, so the same restriction is applied during plan time.
var httpsURLPattern = regexp.MustCompile(`^https://\S+$`)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &WorkflowWebhookResource{}
var _ resource.ResourceWithImportState = &WorkflowWebhookResource{}

// NewWorkflowWebhookResource creates a new instance of WorkflowWebhookResource.
func NewWorkflowWebhookResource() resource.Resource {
	return &WorkflowWebhookResource{}
}

// WorkflowWebhookResource defines the resource implementation.
type WorkflowWebhookResource struct {
	client *client.ClientWithResponses
}

// Metadata is a function to set the TypeName for the Entitle Workflow Webhook resource.
func (r *WorkflowWebhookResource) Metadata(
	ctx context.Context,
	req resource.MetadataRequest,
	resp *resource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_workflow_webhook"
}

// Schema defines the schema for the Entitle Workflow Webhook resource.
func (r *WorkflowWebhookResource) Schema(
	ctx context.Context,
	req resource.SchemaRequest,
	resp *resource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: docs.WorkflowWebhookResourceMarkdownDescription,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Entitle Workflow Webhook identifier in uuid format",
				Description:         "Entitle Workflow Webhook identifier in uuid format",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the workflow webhook.",
				Description:         "The name of the workflow webhook.",
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
				},
			},
			"url": schema.StringAttribute{
				Required: true,
				MarkdownDescription: "The endpoint Entitle calls when the webhook is triggered. " +
					"Only `https` URLs are accepted.",
				Description: "The endpoint Entitle calls when the webhook is triggered. " +
					"Only https URLs are accepted.",
				Validators: []validator.String{
					stringvalidator.RegexMatches(
						httpsURLPattern,
						"must be an absolute URL using the https scheme",
					),
				},
			},
			"headers": schema.MapAttribute{
				Optional:    true,
				ElementType: types.StringType,
				MarkdownDescription: "HTTP headers sent with every webhook request, for example an " +
					"`Authorization` header. Values are treated as sensitive.",
				Description: "HTTP headers sent with every webhook request, for example an " +
					"Authorization header. Values are treated as sensitive.",
				Sensitive: true,
			},
		},
	}
}

// Configure is a function to set the client configuration for the WorkflowWebhookResource.
func (r *WorkflowWebhookResource) Configure(
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
			fmt.Sprintf("Expected *client.ClientWithResponses, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = cli
}

// Create is responsible for creating a new resource of type Entitle Workflow Webhook.
//
// It reads the Terraform plan data provided in req.Plan and maps it to the
// WorkflowWebhookResourceModel. Then it sends a request to the Entitle API to create
// the webhook. If the creation is successful, it saves the resource's data into
// Terraform state.
func (r *WorkflowWebhookResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var plan WorkflowWebhookResourceModel

	// Read Terraform plan data into the model
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	headers, diags := getHeadersAsPlanned(ctx, plan.Headers)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Call Entitle API to create the workflow webhook
	apiResp, err := r.client.WorkflowsWebhooksCreateWithResponse(ctx, client.WorkflowsWebhooksCreateJSONRequestBody{
		Name:    plan.Name.ValueString(),
		Url:     plan.URL.ValueString(),
		Headers: headers,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiConnection.Error(),
			fmt.Sprintf("Unable to create the workflow webhook, got error: %v", err),
		)
		return
	}

	err = utils.HTTPResponseToError(apiResp.HTTPResponse.StatusCode, apiResp.Body)
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiResponse.Error(),
			fmt.Sprintf(
				"Failed to create the Workflow Webhook, %s, status code: %d, %s",
				string(apiResp.Body),
				apiResp.HTTPResponse.StatusCode,
				err.Error(),
			),
		)
		return
	}

	// The API also documents an empty 201, guard against a missing result body.
	if apiResp.JSON200 == nil {
		resp.Diagnostics.AddError(
			utils.ErrApiResponse.Error(),
			fmt.Sprintf(
				"Missing result body while creating the Workflow Webhook, status code: %d, %s",
				apiResp.HTTPResponse.StatusCode,
				string(apiResp.Body),
			),
		)
		return
	}

	tflog.Trace(ctx, "created an Entitle Workflow Webhook resource")

	// Update the plan with the created workflow webhook data. Headers are kept as
	// planned, the attribute is optional and not computed so the applied value has
	// to match the configuration exactly.
	plan.ID = utils.TrimmedStringValue(apiResp.JSON200.Result.Id.String())
	plan.Name = utils.TrimmedStringValue(apiResp.JSON200.Result.Name)
	plan.URL = utils.TrimmedStringValue(apiResp.JSON200.Result.Url)

	// Save data into Terraform state
	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read is used to read an existing resource of type Entitle Workflow Webhook.
//
// It retrieves the webhook from the Entitle API, maps it to the
// WorkflowWebhookResourceModel and saves it to Terraform state. When the webhook no
// longer exists it is removed from state so Terraform can plan its recreation.
func (r *WorkflowWebhookResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var data WorkflowWebhookResourceModel

	// Read Terraform prior state data into the model
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Parse the resource ID from the model
	uid, err := uuid.Parse(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Failed to parse the resource id (%s) to UUID, got error: %s", data.ID.ValueString(), err),
		)
		return
	}

	// Call Entitle API to get the workflow webhook by ID
	apiResp, err := r.client.WorkflowsWebhooksShowWithResponse(ctx, uid)
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiConnection.Error(),
			fmt.Sprintf("Unable to get the workflow webhook by the id (%s), got error: %s", uid.String(), err),
		)
		return
	}

	err = utils.HTTPResponseToError(apiResp.HTTPResponse.StatusCode, apiResp.Body)
	if err != nil {
		if errors.Is(err, utils.ErrNotFound) {
			tflog.Debug(ctx, "Resource no longer exists, removing from state")

			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			utils.ErrApiResponse.Error(),
			fmt.Sprintf(
				"Failed to get the Workflow Webhook by the id (%s), status code: %d, %s",
				uid.String(),
				apiResp.HTTPResponse.StatusCode,
				err.Error(),
			),
		)
		return
	}

	responseSchema := apiResp.JSON200.Result

	headers, diags := getHeaders(ctx, responseSchema.Headers)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data = WorkflowWebhookResourceModel{
		ID:      utils.TrimmedStringValue(responseSchema.Id.String()),
		Name:    utils.TrimmedStringValue(responseSchema.Name),
		URL:     utils.TrimmedStringValue(responseSchema.Url),
		Headers: headers,
	}

	// Save updated data into Terraform state
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update handles updates to an existing resource of type Entitle Workflow Webhook.
//
// It reads the updated Terraform plan data, sends it to the Entitle API and saves the
// updated webhook data into Terraform state.
func (r *WorkflowWebhookResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var plan WorkflowWebhookResourceModel

	// Read Terraform plan data into the model
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Parse the resource ID from the model
	uid, err := uuid.Parse(plan.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Failed to parse the resource id (%s) to UUID, got error: %s", plan.ID.ValueString(), err),
		)
		return
	}

	headers, diags := getHeadersAsPlanned(ctx, plan.Headers)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := plan.Name.ValueString()
	url := plan.URL.ValueString()

	// Call Entitle API to update the workflow webhook
	apiResp, err := r.client.WorkflowsWebhooksUpdateWithResponse(ctx, uid, client.WorkflowsWebhooksUpdateJSONRequestBody{
		Name:    &name,
		Url:     &url,
		Headers: headers,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiConnection.Error(),
			fmt.Sprintf("Unable to update the workflow webhook by the id (%s), got error: %s", uid.String(), err),
		)
		return
	}

	err = utils.HTTPResponseToError(apiResp.HTTPResponse.StatusCode, apiResp.Body)
	if err != nil {
		if errors.Is(err, utils.ErrNotFound) {
			tflog.Debug(ctx, "Resource no longer exists, removing from state")

			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			utils.ErrApiResponse.Error(),
			fmt.Sprintf(
				"Failed to update the Workflow Webhook by the id (%s), status code: %d, %s",
				uid.String(),
				apiResp.HTTPResponse.StatusCode,
				err.Error(),
			),
		)
		return
	}

	// Update the plan with the updated workflow webhook data. Headers are kept as
	// planned, the attribute is optional and not computed so the applied value has
	// to match the configuration exactly.
	plan.ID = utils.TrimmedStringValue(apiResp.JSON200.Result.Id.String())
	plan.Name = utils.TrimmedStringValue(apiResp.JSON200.Result.Name)
	plan.URL = utils.TrimmedStringValue(apiResp.JSON200.Result.Url)

	// Save the updated data into Terraform state
	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete is responsible for deleting an existing resource of type Entitle Workflow Webhook.
//
// It reads the resource's data from Terraform state, extracts the unique identifier
// and sends a delete request to the Entitle API.
func (r *WorkflowWebhookResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var data WorkflowWebhookResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Parse the resource ID from the model
	uid, err := uuid.Parse(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to parse uuid of the workflow webhook, id: (%s), got error: %v", data.ID.ValueString(), err),
		)
		return
	}

	// Call Entitle API to delete the workflow webhook resource
	apiResp, err := r.client.WorkflowsWebhooksDestroyWithResponse(ctx, uid)
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiConnection.Error(),
			fmt.Sprintf("Unable to delete workflow webhook, id: (%s), got error: %v", uid.String(), err),
		)
		return
	}

	err = utils.HTTPResponseToError(apiResp.HTTPResponse.StatusCode, apiResp.Body, utils.WithIgnoreNotFound())
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiResponse.Error(),
			fmt.Sprintf(
				"Unable to delete Workflow Webhook, id: (%s), status code: %v, %s",
				uid.String(),
				apiResp.HTTPResponse.StatusCode,
				err.Error(),
			),
		)
		return
	}
}

// ImportState is used to import an existing resource's state into Terraform.
//
// It extracts the resource's identifier from the import request and sets it in
// Terraform state using resource.ImportStatePassthroughID.
func (r *WorkflowWebhookResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
