package provider

import (
	"context"
	_ "embed"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/entitleio/terraform-provider-entitle/docs"
	"github.com/entitleio/terraform-provider-entitle/internal/client"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/accessRequestForwards"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/accessReviewForwards"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/accounts"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/agentTokens"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/bundles"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/directoryGroups"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/integrations"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/permissions"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/policies"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/resources"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/roles"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/users"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/workflows"
)

const (
	defaultAPIServer = "https://api.entitle.io"
)

// Ensure EntitleProvider satisfies various provider interfaces.
var _ provider.Provider = &EntitleProvider{}

// EntitleProvider defines the provider implementation.
type EntitleProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// EntitleProviderModel describes the provider data model.
type EntitleProviderModel struct {
	Endpoint types.String `tfsdk:"endpoint"`
	APIKey   types.String `tfsdk:"api_key"`
}

// Metadata sets the provider metadata.
func (p *EntitleProvider) Metadata(
	ctx context.Context,
	req provider.MetadataRequest,
	resp *provider.MetadataResponse,
) {
	resp.TypeName = "entitle"
	resp.Version = p.version
}

// Schema defines the provider schema with its attributes.
func (p *EntitleProvider) Schema(
	ctx context.Context,
	req provider.SchemaRequest,
	resp *provider.SchemaResponse,
) {
	resp.Schema = schema.Schema{
		MarkdownDescription: docs.ProviderMarkdownDescription,
		Attributes: map[string]schema.Attribute{
			"api_key": schema.StringAttribute{
				MarkdownDescription: "API key for authentication with the Entitle API. Can also be set via the `ENTITLE_API_KEY` environment variable.",
				Description:         "API key for authentication with the Entitle API. Can also be set via the `ENTITLE_API_KEY` environment variable.",
				Sensitive:           true,
				Optional:            true,
			},
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "The Entitle API endpoint URL. Defaults to `https://api.entitle.io` (EU region). See [Regional Endpoints](#regional-endpoints).",
				Description:         "The Entitle API endpoint URL. Defaults to `https://api.entitle.io` (EU region). See [Regional Endpoints](#regional-endpoints).",
				Optional:            true,
			},
		},
	}
}

// Configure configures the provider based on the provided configuration.
func (p *EntitleProvider) Configure(
	ctx context.Context,
	req provider.ConfigureRequest,
	resp *provider.ConfigureResponse,
) {
	var config EntitleProviderModel

	// Retrieve configuration values from the request.
	diags := req.Config.Get(ctx, &config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate and set API key.
	token := os.Getenv("ENTITLE_API_KEY")
	if !config.APIKey.IsUnknown() {
		token = config.APIKey.String()
	}

	if token == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_key"),
			"Missing Entitle API Key",
			"The provider cannot create the Entitle API client as there is a missing or empty value for the "+
				"Entitle API token. "+
				"Set the token value in the configuration or use the ENTITLE_API_KEY environment variable. "+
				"If either is already set, ensure the value is not empty.",
		)
	}

	// Check for errors before proceeding.
	if resp.Diagnostics.HasError() {
		return
	}

	// Retrieve server value from environment variables or configuration.
	server := os.Getenv("ENTITLE_API_ENDPOINT")
	if !config.Endpoint.IsNull() {
		server = config.Endpoint.ValueString()
	}

	if server == "" {
		config.Endpoint = types.StringValue(defaultAPIServer)
		server = defaultAPIServer
	}

	// Set log fields and create Entitle client.
	ctx = tflog.SetField(ctx, "entitle_endpoint", server)
	ctx = tflog.SetField(ctx, "entitle_token", token)
	ctx = tflog.MaskFieldValuesWithFieldKeys(ctx, "entitle_token")

	tflog.Debug(ctx, "Creating entitle client...")

	c, err := client.NewClientWithResponses(
		server,
		client.WithRequestEditorFn(
			client.SetBearerToken(token),
		),
	)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Entitle API Client",
			"An unexpected error occurred when creating the Entitle API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"Entitle Client Error: "+err.Error(),
		)
		return
	}

	// Set client configuration for data sources and resources.
	resp.DataSourceData = c
	resp.ResourceData = c

	tflog.Info(ctx, "Configured Entitle client", map[string]any{"success": true})
}

// Resources returns the list of provider resources.
func (p *EntitleProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		accessRequestForwards.NewAccessRequestForwardResource,
		accessReviewForwards.NewAccessReviewForwardResource,
		agentTokens.NewAgentTokenResource,
		bundles.NewBundleResource,
		integrations.NewIntegrationResource,
		permissions.NewPermissionResource,
		policies.NewPolicyResource,
		resources.NewResourceResource,
		roles.NewRoleResource,
		workflows.NewWorkflowResource,
	}
}

// DataSources returns the list of provider data sources.
func (p *EntitleProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		accounts.NewAccountsDataSource,
		agentTokens.NewAgentTokenDataSource,
		accessRequestForwards.NewAccessRequestForwardDataSource,
		accessReviewForwards.NewAccessReviewForwardDataSource,
		bundles.NewBundleDataSource,
		directoryGroups.NewDirectoryGroupsDataSource,
		integrations.NewIntegrationDataSource,
		permissions.NewPermissionsDataSource,
		policies.NewPolicyDataSource,
		resources.NewResourcesDataSource,
		resources.NewResourceDataSource,
		roles.NewRoleDataSource,
		roles.NewRolesDataSource,
		users.NewUserDataSource,
		users.NewUsersDataSource,
		workflows.NewWorkflowDataSource,
	}
}

// New returns a new instance of the EntitleProvider.
// This function is a factory for creating a new provider instance.
// It takes the version as a parameter and returns a function that, when called, creates a new EntitleProvider.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &EntitleProvider{
			version: version,
		}
	}
}
