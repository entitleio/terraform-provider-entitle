package provider

import (
	"context"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/entitleio/terraform-provider-entitle/internal/client"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/agentTokens"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/bundles"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/directoryGroups"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/integrations"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/policies"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/resources"
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
		MarkdownDescription: "The Entitle provider allows you to manage your [Entitle](https://www.entitle.io) resources and configurations through Terraform. It provides the ability to automate the management of integrations, workflows, and access policies within your Entitle environment.",
		Description:         "The Entitle provider allows you to manage your Entitle resources and configurations through Terraform.",
		Attributes: map[string]schema.Attribute{
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "entitle API server address, default: https://api.entitle.io",
				Description:         "entitle API server address, default: https://api.entitle.io",
				Optional:            true,
			},
			"api_key": schema.StringAttribute{
				MarkdownDescription: "entitle API bearer authorizations (http, Bearer)",
				Description:         "entitle API bearer authorizations (http, Bearer)",
				Sensitive:           true,
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
		integrations.NewIntegrationResource,
		bundles.NewBundleResource,
		policies.NewPolicyResource,
		workflows.NewWorkflowResource,
		agentTokens.NewAgentTokenResource,
	}
}

// DataSources returns the list of provider data sources.
func (p *EntitleProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		integrations.NewIntegrationDataSource,
		bundles.NewBundleDataSource,
		policies.NewPolicyDataSource,
		workflows.NewWorkflowDataSource,
		agentTokens.NewAgentTokenDataSource,
		resources.NewResourceDataSource,
		users.NewUserDataSource,
		directoryGroups.NewDirectoryGroupDataSource,
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
