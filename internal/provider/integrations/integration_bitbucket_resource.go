package integrations

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/entitleio/terraform-provider-entitle/docs"
	"github.com/entitleio/terraform-provider-entitle/internal/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &IntegrationBitbucketResource{}
var _ resource.ResourceWithImportState = &IntegrationBitbucketResource{}

func NewIntegrationBitbucketResource() resource.Resource {
	return &IntegrationBitbucketResource{}
}

// IntegrationBitbucketResource defines the resource implementation.
type IntegrationBitbucketResource struct {
	client *client.ClientWithResponses
}

type BitbucketJiraCredentialsModel struct {
	URL  types.String `tfsdk:"url"`
	Key  types.String `tfsdk:"key"`
	User types.String `tfsdk:"user"`
}

type BitbucketConnectionModel struct {
	Email           types.String                   `tfsdk:"email"`
	AppToken        types.String                   `tfsdk:"app_token"`
	JiraCredentials *BitbucketJiraCredentialsModel `tfsdk:"jira_credentials"`
}

// IntegrationBitbucketResourceModel describes the resource data model.
type IntegrationBitbucketResourceModel struct {
	BaseIntegrationResourceModel
	Connection *BitbucketConnectionModel `tfsdk:"connection_data"`
}

func (r *IntegrationBitbucketResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration_bitbucket"
}

func (r *IntegrationBitbucketResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: docs.IntegrationBitbucketResourceMarkdownDescription,
		Attributes: func() map[string]schema.Attribute {
			m := GetBaseIntegrationResourceAttributes(applicationBitbucket)

			m["connection_data"] = schema.SingleNestedAttribute{
				Description: "Bitbucket connection credentials, including optional Jira credentials used for " +
					"email-based user matching.",
				Attributes: map[string]schema.Attribute{
					"email": schema.StringAttribute{
						Description: "The Atlassian account email address used to authenticate with Bitbucket. " +
							"Copy this from the Atlassian Email settings page.",
						Required:  true,
						WriteOnly: true,
					},
					"app_token": schema.StringAttribute{
						Description: "An Atlassian API token with Bitbucket scopes. Create one from the " +
							"Atlassian API Tokens page with the following scopes: `admin:workspace:bitbucket`, " +
							"`admin:repository:bitbucket`, `read:workspace:bitbucket`, `read:repository:bitbucket`, " +
							"`read:permission:bitbucket`, `write:permission:bitbucket`, `delete:permission:bitbucket`.",
						Required:  true,
						Sensitive: true,
						WriteOnly: true,
					},
					"jira_credentials": schema.SingleNestedAttribute{
						Description: "Optional Jira credentials Entitle uses to look up user email addresses, " +
							"since Bitbucket's API does not expose them. Provide this to enable Entitle to " +
							"automatically match Bitbucket users to identities by email.",
						Attributes: map[string]schema.Attribute{
							"url": schema.StringAttribute{
								Description: "Your Jira instance URL (e.g. \"https://your-domain.atlassian.net\").",
								Required:    true,
								WriteOnly:   true,
							},
							"key": schema.StringAttribute{
								Description: "A Jira API token used to authenticate with the Jira API.",
								Required:    true,
								Sensitive:   true,
								WriteOnly:   true,
							},
							"user": schema.StringAttribute{
								Description: "The Jira account email address associated with the API token.",
								Required:    true,
								WriteOnly:   true,
							},
						},
						Optional:  true,
						WriteOnly: true,
					},
				},
				Required:  true,
				WriteOnly: true,
			}

			return m
		}(),
	}
}

func (r *IntegrationBitbucketResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	configureIntegrationResource(req.ProviderData, &r.client, &resp.Diagnostics)
}

// Create this function is responsible for creating a new resource of type Entitle Integration.
//
// Its reads the Terraform plan data provided in req.Plan and maps it to the IntegrationBitbucketResourceModel.
// And sends a request to the Entitle API to create the resource using API requests.
// If the creation is successful, it saves the resource's data into Terraform state.
func (r *IntegrationBitbucketResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan IntegrationBitbucketResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	parsedConnectionJson := parseBitbucketConnectionJson(plan.Connection)

	newBase, diags := CreateIntegration(ctx, r.client, plan.BaseIntegrationResourceModel, applicationBitbucket, parsedConnectionJson)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, IntegrationBitbucketResourceModel{
		BaseIntegrationResourceModel: newBase,
		Connection:                   plan.Connection,
	})...)
}

// Read this function is used to read an existing resource of type Entitle Integration.
//
// It retrieves the resource's data from the provider API requests.
// The retrieved data is then mapped to the IntegrationBitbucketResourceModel,
// and the data is saved to Terraform state.
func (r *IntegrationBitbucketResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data IntegrationBitbucketResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	newBase, _, ok := ReadIntegration(ctx, r.client, data.BaseIntegrationResourceModel, resp)
	if !ok {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, IntegrationBitbucketResourceModel{
		BaseIntegrationResourceModel: newBase,
		Connection:                   data.Connection,
	})...)
}

// Update this function handles updates to an existing resource of type Entitle Integration.
//
// It reads the updated Terraform plan data provided in req.Plan and maps it to the IntegrationBitbucketResourceModel.
// And sends a request to the Entitle API to update the resource using API requests.
// If the update is successful, it saves the updated resource data into Terraform state.
func (r *IntegrationBitbucketResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data IntegrationBitbucketResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var parsedConnectionJson *map[string]interface{}
	if data.Connection != nil {
		parsedConnectionJson = new(parseBitbucketConnectionJson(data.Connection))
	}

	newBase := UpdateIntegration(ctx, r.client, data.BaseIntegrationResourceModel, applicationBitbucket, parsedConnectionJson, resp)
	if newBase == nil {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, IntegrationBitbucketResourceModel{
		BaseIntegrationResourceModel: *newBase,
		Connection:                   data.Connection,
	})...)
}

func parseBitbucketConnectionJson(m *BitbucketConnectionModel) map[string]interface{} {
	var connectionModel BitbucketConnectionModel
	if m != nil {
		connectionModel = *m
	}

	jsonSchema := map[string]interface{}{
		"email":     connectionModel.Email.ValueString(),
		"app_token": connectionModel.AppToken.ValueString(),
	}

	if connectionModel.JiraCredentials != nil {
		jsonSchema["jira_credentials"] = map[string]interface{}{
			"url":  connectionModel.JiraCredentials.URL.ValueString(),
			"key":  connectionModel.JiraCredentials.Key.ValueString(),
			"user": connectionModel.JiraCredentials.User.ValueString(),
		}
	}

	return jsonSchema
}

// Delete this function is responsible for deleting an existing resource of type Entitle Bitbucket Integration.
//
// It reads the resource's data from Terraform state, extracts the unique identifier,
// and sends a request to delete the resource using API requests.
// If the deletion is successful, it removes the resource from Terraform state.
func (r *IntegrationBitbucketResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data IntegrationBitbucketResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	DeleteIntegration(ctx, r.client, data.BaseIntegrationResourceModel, resp)
}

// ImportState this function is used to import an existing resource's state into Terraform.
//
// It extracts the resource's identifier from the import request and sets
// it in Terraform state using resource.ImportStatePassthroughID.
func (r *IntegrationBitbucketResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
