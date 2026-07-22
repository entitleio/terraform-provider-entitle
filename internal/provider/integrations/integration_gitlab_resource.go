package integrations

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/entitleio/terraform-provider-entitle/docs"
	"github.com/entitleio/terraform-provider-entitle/internal/client"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &IntegrationGitlabResource{}
var _ resource.ResourceWithImportState = &IntegrationGitlabResource{}

const GitlabDefaultDomain = "https://gitlab.com"

func NewIntegrationGitlabResource() resource.Resource {
	return &IntegrationGitlabResource{}
}

// IntegrationGitlabResource defines the resource implementation.
type IntegrationGitlabResource struct {
	client *client.ClientWithResponses
}

type GitlabConnectionModel struct {
	Domain       types.String `tfsdk:"domain"`
	PrivateToken types.String `tfsdk:"private_token"`
	SSLVerify    types.Bool   `tfsdk:"ssl_verify"`
	SSLCaCert    types.String `tfsdk:"ssl_ca_cert"`
}

// IntegrationGitlabResourceModel describes the resource data model.
type IntegrationGitlabResourceModel struct {
	BaseIntegrationResourceModel
	Connection *GitlabConnectionModel `tfsdk:"connection_data"`
}

func (r *IntegrationGitlabResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration_gitlab"
}

func (r *IntegrationGitlabResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: docs.IntegrationGitlabResourceMarkdownDescription,
		Attributes: func() map[string]schema.Attribute {
			m := GetBaseIntegrationResourceAttributes(applicationGitlab)

			m["connection_data"] = schema.SingleNestedAttribute{
				Description: "GitLab connection credentials and SSL settings.",
				Attributes: map[string]schema.Attribute{
					"domain": schema.StringAttribute{
						Description: "The GitLab instance URL. Defaults to \"https://gitlab.com\" for GitLab SaaS. " +
							"For self-hosted GitLab, provide your own domain (e.g. \"https://gitlab.example.com\").",
						Optional: true,
						Default:  stringdefault.StaticString(GitlabDefaultDomain),
						Computed: true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"private_token": schema.StringAttribute{
						Description: "A GitLab Personal Access Token with the `api` scope. " +
							"Create one in GitLab under Edit Profile → Access Tokens.",
						Required:  true,
						Sensitive: true,
					},
					"ssl_verify": schema.BoolAttribute{
						Description: "Whether to verify the GitLab server's SSL certificate. " +
							"Defaults to true. Set to false only when connecting to a self-hosted instance " +
							"without providing a custom CA certificate.",
						Optional: true,
						Default:  booldefault.StaticBool(true),
						Computed: true,
						PlanModifiers: []planmodifier.Bool{
							boolplanmodifier.UseStateForUnknown(),
						},
					},
					"ssl_ca_cert": schema.StringAttribute{
						Description: "Path to a custom CA certificate file in PEM format, used when connecting " +
							"to a self-hosted GitLab instance with a self-signed certificate " +
							"(e.g. \"/etc/ssl/certs/gitlab_ca.pem\"). The Entitle agent must have read access to this path. " +
							"Not required for public certificates or when ssl_verify is false.",
						Optional: true,
					},
				},
				Required: true,
			}

			return m
		}(),
	}
}

func (r *IntegrationGitlabResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	configureIntegrationResource(req.ProviderData, &r.client, &resp.Diagnostics)
}

// Create this function is responsible for creating a new resource of type Entitle Integration.
//
// Its reads the Terraform plan data provided in req.Plan and maps it to the IntegrationGitlabResourceModel.
// And sends a request to the Entitle API to create the resource using API requests.
// If the creation is successful, it saves the resource's data into Terraform state.
func (r *IntegrationGitlabResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan IntegrationGitlabResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	parsedConnectionJson := parseGitlabConnectionJson(plan.Connection)

	newBase, diags := CreateIntegration(ctx, r.client, plan.BaseIntegrationResourceModel, applicationGitlab, parsedConnectionJson)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, IntegrationGitlabResourceModel{
		BaseIntegrationResourceModel: newBase,
		Connection:                   plan.Connection,
	})...)
}

// Read this function is used to read an existing resource of type Entitle Integration.
//
// It retrieves the resource's data from the provider API requests.
// The retrieved data is then mapped to the IntegrationGitlabResourceModel,
// and the data is saved to Terraform state.
func (r *IntegrationGitlabResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data IntegrationGitlabResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	newBase, _, ok := ReadIntegration(ctx, r.client, data.BaseIntegrationResourceModel, resp)
	if !ok {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, IntegrationGitlabResourceModel{
		BaseIntegrationResourceModel: newBase,
		Connection:                   data.Connection,
	})...)
}

// Update this function handles updates to an existing resource of type Entitle Integration.
//
// It reads the updated Terraform plan data provided in req.Plan and maps it to the IntegrationGitlabResourceModel.
// And sends a request to the Entitle API to update the resource using API requests.
// If the update is successful, it saves the updated resource data into Terraform state.
func (r *IntegrationGitlabResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data IntegrationGitlabResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	parsedConnectionJson := parseGitlabConnectionJson(data.Connection)
	newBase := UpdateIntegration(ctx, r.client, data.BaseIntegrationResourceModel, applicationGitlab, parsedConnectionJson, resp)
	if newBase == nil {
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, IntegrationGitlabResourceModel{
		BaseIntegrationResourceModel: *newBase,
		Connection:                   data.Connection,
	})...)
}

func parseGitlabConnectionJson(m *GitlabConnectionModel) map[string]interface{} {
	var connectionModel GitlabConnectionModel
	if m != nil {
		connectionModel = *m
	}
	ssl := map[string]interface{}{
		"verify": connectionModel.SSLVerify.ValueBool(),
	}

	caCert := connectionModel.SSLCaCert.ValueStringPointer()
	if caCert != nil {
		ssl["ca_cert"] = caCert
	}

	jsonSchema := map[string]interface{}{
		"configurationSchemaName": "Configuration ",
		"domain":                  connectionModel.Domain.ValueString(),
		"private_token":           connectionModel.PrivateToken.ValueString(),
		"options": map[string]interface{}{
			"ssl": ssl,
		},
	}

	return jsonSchema
}

// Delete this function is responsible for deleting an existing resource of type Entitle GitLab Integration.
//
// It reads the resource's data from Terraform state, extracts the unique identifier,
// and sends a request to delete the resource using API requests.
// If the deletion is successful, it removes the resource from Terraform state.
func (r *IntegrationGitlabResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data IntegrationGitlabResourceModel

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
func (r *IntegrationGitlabResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
