// Package agentTokens provides the implementation of the Entitle Agent Token data source for Terraform.
// This data source allows Terraform to query information about Agent Tokens from the Entitle API.
package agentTokens

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/entitleio/terraform-provider-entitle/internal/client"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/utils"
	"github.com/entitleio/terraform-provider-entitle/internal/validators"
)

// Ensure that the types defined by the provider satisfy framework interfaces.
var _ datasource.DataSource = &AgentTokenDataSource{}

// AgentTokenDataSource defines the implementation of the data source.
type AgentTokenDataSource struct {
	client *client.ClientWithResponses
}

// NewAgentTokenDataSource creates a new instance of AgentTokenDataSource.
func NewAgentTokenDataSource() datasource.DataSource {
	return &AgentTokenDataSource{}
}

// AgentTokenDataSourceModel describes the data source data model.
type AgentTokenDataSourceModel struct {
	ID   types.String `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
}

// Metadata sets the metadata for the data source.
func (d *AgentTokenDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_agent_token"
}

// Schema sets the schema for the data source.
func (d *AgentTokenDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Entitle AgentToken Description",
		Description:         "Entitle AgentToken Description",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Entitle AgentToken identifier in UUID format",
				Description:         "Entitle AgentToken identifier in UUID format",
				Validators: []validator.String{
					validators.UUID{},
				},
			},
			"name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Entitle AgentToken name",
				Description:         "Entitle AgentToken name",
			},
		},
	}
}

// Configure configures the data source with the provided client.
func (d *AgentTokenDataSource) Configure(
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

// Read retrieves data from the Entitle API and populates the data source state.
// This function is responsible for fetching details about an Agent Token from the Entitle API
// based on the provided Terraform configuration. It reads the configuration data into a model,
// sends a request to the Entitle API, and processes the API response. The retrieved data is then
// saved into Terraform state for further use in the Terraform plan.
func (d *AgentTokenDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	// Create a model to hold data from Terraform configuration
	var data AgentTokenDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Parse the Agent Token ID from the configuration model
	uid, err := uuid.Parse(data.ID.String())
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Failed to parse the resource ID (%s) to UUID, got error: %s", data.ID.String(), err),
		)
		return
	}

	// Fetch Agent Token details from the Entitle API
	agentTokenResp, err := d.client.AgentTokensShowWithResponse(ctx, uid)
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to get the Agent Token by the ID (%s), got error: %s", uid.String(), err),
		)
		return
	}

	// Check if the API response indicates success
	if agentTokenResp.StatusCode() != 200 || agentTokenResp.JSON200 == nil {
		errBody, _ := utils.GetErrorBody(agentTokenResp.Body)
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf(
				"Failed to get the Agent Token by the ID (%s), status code: %d%s",
				uid.String(),
				agentTokenResp.StatusCode(),
				errBody.GetMessage(),
			),
		)
		return
	}

	// Populate the data model with details from the API response
	data = AgentTokenDataSourceModel{
		ID:   utils.TrimmedStringValue(agentTokenResp.JSON200.Result.Id.String()),
		Name: utils.TrimmedStringValue(agentTokenResp.JSON200.Result.Name),
	}

	// Log a trace message indicating a successful read of the Entitle Agent Token data source
	tflog.Trace(ctx, "Read an Entitle Agent Token data source")

	// Save the retrieved data into Terraform state
	diagsResult := resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diagsResult...)
	if resp.Diagnostics.HasError() {
		return
	}
}
