package workflowWebhooks

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/entitleio/terraform-provider-entitle/docs"
	"github.com/entitleio/terraform-provider-entitle/internal/client"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/utils"
	"github.com/entitleio/terraform-provider-entitle/internal/validators"
)

// Ensure that the provider-defined types fully satisfy the framework interfaces.
var _ datasource.DataSource = &WorkflowWebhookDataSource{}

// WorkflowWebhookDataSource defines the data source implementation.
type WorkflowWebhookDataSource struct {
	client *client.ClientWithResponses
}

// NewWorkflowWebhookDataSource creates a new instance of WorkflowWebhookDataSource.
func NewWorkflowWebhookDataSource() datasource.DataSource {
	return &WorkflowWebhookDataSource{}
}

// Metadata sets the data source's metadata, such as its type name.
func (d *WorkflowWebhookDataSource) Metadata(
	ctx context.Context,
	req datasource.MetadataRequest,
	resp *datasource.MetadataResponse,
) {
	resp.TypeName = req.ProviderTypeName + "_workflow_webhook"
}

// Schema sets the schema for the data source.
func (d *WorkflowWebhookDataSource) Schema(
	ctx context.Context,
	req datasource.SchemaRequest,
	resp *datasource.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: docs.WorkflowWebhookDataSourceMarkdownDescription,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Entitle Workflow Webhook identifier in uuid format. If not provided then name will be used to get entity.",
				Description:         "Entitle Workflow Webhook identifier in uuid format. If not provided then name will be used to get entity.",
				Validators: []validator.String{
					validators.UUID{},
					stringvalidator.ExactlyOneOf(
						path.MatchRoot("id"),
						path.MatchRoot("name"),
					),
				},
			},
			"name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The workflow webhook's name. Querying by name is case sensitive.",
				Description:         "The workflow webhook's name. Querying by name is case sensitive.",
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(
						path.MatchRoot("id"),
						path.MatchRoot("name"),
					),
				},
			},
			"url": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The endpoint Entitle calls when the webhook is triggered.",
				Description:         "The endpoint Entitle calls when the webhook is triggered.",
			},
			"headers": schema.MapAttribute{
				Computed:            true,
				ElementType:         types.StringType,
				MarkdownDescription: "HTTP headers sent with every webhook request.",
				Description:         "HTTP headers sent with every webhook request.",
				Sensitive:           true,
			},
		},
	}
}

// Configure configures the data source with the provided client.
func (d *WorkflowWebhookDataSource) Configure(
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
			fmt.Sprintf("Expected *client.ClientWithResponses, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = c
}

// Read retrieves a workflow webhook from the Entitle API and populates the data
// source state. The webhook is looked up by its identifier, or by name when no
// identifier is given.
func (d *WorkflowWebhookDataSource) Read(
	ctx context.Context,
	req datasource.ReadRequest,
	resp *datasource.ReadResponse,
) {
	var data WorkflowWebhookDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var uid openapi_types.UUID
	if data.ID.ValueString() == "" {
		name := data.Name.ValueString()

		id, err := d.getWorkflowWebhookIDByName(ctx, name)
		if err != nil {
			resp.Diagnostics.AddError("Workflow Webhook not found", fmt.Sprintf(
				"Failed to get the Workflow Webhook by the name (%s), %s",
				name,
				err.Error(),
			))

			return
		}

		uid = *id
	} else {
		uid = uuid.MustParse(data.ID.String())
	}

	// Fetch the workflow webhook details from the Entitle API
	webhookResp, err := d.client.WorkflowsWebhooksShowWithResponse(ctx, uid)
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiConnection.Error(),
			fmt.Sprintf("Unable to get the workflow webhook by the id (%s), got error: %s", uid.String(), err),
		)
		return
	}

	err = utils.HTTPResponseToError(webhookResp.HTTPResponse.StatusCode, webhookResp.Body)
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiResponse.Error(),
			fmt.Sprintf(
				"Failed to get the Workflow Webhook by the id (%s), status code: %d, %s",
				uid.String(),
				webhookResp.HTTPResponse.StatusCode,
				err.Error(),
			),
		)
		return
	}

	responseSchema := webhookResp.JSON200.Result

	headers, diags := getHeaders(ctx, responseSchema.Headers)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Populate the data model with details from the API response
	data = WorkflowWebhookDataSourceModel{
		ID:      utils.TrimmedStringValue(responseSchema.Id.String()),
		Name:    utils.TrimmedStringValue(responseSchema.Name),
		URL:     utils.TrimmedStringValue(responseSchema.Url),
		Headers: headers,
	}

	tflog.Trace(ctx, "read an entitle workflow webhook data source")

	// Save the retrieved data into Terraform state
	diagsResult := resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diagsResult...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// getWorkflowWebhookIDByName resolves a workflow webhook name to its identifier.
//
// The workflow webhooks endpoint is not paginated, so the whole list is fetched as a
// single page and matched by exact, case sensitive name.
func (d *WorkflowWebhookDataSource) getWorkflowWebhookIDByName(
	ctx context.Context,
	name string,
) (*openapi_types.UUID, error) {
	fetch := func(ctx context.Context, page int) ([]client.WorkflowsWebhookResponseSchema, int, error) {
		webhooksResp, err := d.client.WorkflowsWebhooksIndexWithResponse(ctx)
		if err != nil {
			return nil, 0, fmt.Errorf("%s, %w", utils.ErrApiConnection.Error(), err)
		}

		if err := utils.HTTPResponseToError(
			webhooksResp.HTTPResponse.StatusCode,
			webhooksResp.Body,
		); err != nil {
			return nil, 0, fmt.Errorf("%s, %w", utils.ErrApiResponse.Error(), err)
		}

		if webhooksResp.JSON200 == nil {
			return nil, 0, utils.ErrNotFound
		}

		return webhooksResp.JSON200.Result, 1, nil
	}

	return utils.FindIDByName(ctx, name, fetch)
}
