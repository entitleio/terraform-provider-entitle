package provider

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strconv"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/entitleio/terraform-provider-entitle/docs"
	"github.com/entitleio/terraform-provider-entitle/internal/client"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/accessRequestForwards"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/accessReviewForwards"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/accounts"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/agentTokens"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/applications"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/bundles"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/directoryGroups"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/integrations"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/permissions"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/policies"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/resources"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/roles"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/users"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/utils"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/workflows"
)

const (
	defaultAPIServer = "https://api.entitle.io"

	envBulkChunkSize   = "ENTITLE_BULK_CHUNK_SIZE"
	envBulkParallelism = "ENTITLE_BULK_PARALLELISM"
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
	Endpoint types.String       `tfsdk:"endpoint"`
	APIKey   types.String       `tfsdk:"api_key"`
	Bulk     *BulkProviderModel `tfsdk:"bulk"`
}

// BulkProviderModel describes the provider-wide limits for bulk operations.
type BulkProviderModel struct {
	ChunkSize   types.Int64 `tfsdk:"chunk_size"`
	Parallelism types.Int64 `tfsdk:"parallelism"`
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
			"endpoint": schema.StringAttribute{
				MarkdownDescription: "Entitle API server address. Allowed values:\n\n" +
					"  - https://api.entitle.io (default, Europe)\n" +
					"  - https://api.ca.entitle.io (Canada)\n" +
					"  - https://api.us.entitle.io (United States)\n",
				Description: "Entitle API server address. Allowed values: https://api.entitle.io (default, Europe), https://api.ca.entitle.io (Canada), https://api.us.entitle.io (United States)",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(
						"https://api.entitle.io",
						"https://api.ca.entitle.io",
						"https://api.us.entitle.io",
					),
				},
			},
			"api_key": schema.StringAttribute{
				MarkdownDescription: "API key for authentication with the Entitle API. Can also be set via the `ENTITLE_API_KEY` environment variable.",
				Description:         "API key for authentication with the Entitle API. Can also be set via the `ENTITLE_API_KEY` environment variable.",
				Sensitive:           true,
				Optional:            true,
			},
		},
		Blocks: map[string]schema.Block{
			"bulk": schema.SingleNestedBlock{
				MarkdownDescription: "Provider-wide limits applied to bulk operations.",
				Description:         "Provider-wide limits applied to bulk operations.",
				Attributes: map[string]schema.Attribute{
					"chunk_size": schema.Int64Attribute{
						MarkdownDescription: fmt.Sprintf("Maximum number of items sent in a single bulk API request. "+
							"Defaults to `%d`. Can also be set via the `%s` environment variable.", utils.DefaultBulkChunkSize, envBulkChunkSize),
						Description: fmt.Sprintf("Maximum number of items sent in a single bulk API request. "+
							"Defaults to %d. Can also be set via the %s environment variable.", utils.DefaultBulkChunkSize, envBulkChunkSize),
						Optional:   true,
						Validators: []validator.Int64{int64validator.AtLeast(1)},
					},
					"parallelism": schema.Int64Attribute{
						MarkdownDescription: fmt.Sprintf("Maximum number of bulk API requests executed concurrently. "+
							"Defaults to `%d`. Can also be set via the `%s` environment variable.", utils.DefaultBulkParallelism, envBulkParallelism),
						Description: fmt.Sprintf("Maximum number of bulk API requests executed concurrently. "+
							"Defaults to %d. Can also be set via the %s environment variable.", utils.DefaultBulkParallelism, envBulkParallelism),
						Optional:   true,
						Validators: []validator.Int64{int64validator.AtLeast(1)},
					},
				},
			},
		},
	}
}

// resolveBulkConfig builds the effective bulk configuration. Precedence: provider
// configuration, then environment variables, then defaults.
func resolveBulkConfig(model *BulkProviderModel, getenv func(string) string) (utils.BulkConfig, diag.Diagnostics) {
	var diags diag.Diagnostics
	cfg := utils.DefaultBulkConfig()

	resolve := func(attr string, envName string, value types.Int64, target *int64) {
		if !value.IsNull() && !value.IsUnknown() {
			*target = value.ValueInt64()
			return
		}

		raw := getenv(envName)
		if raw == "" {
			return
		}

		v, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || v < 1 {
			diags.AddAttributeError(
				path.Root("bulk").AtName(attr),
				"Invalid Bulk Configuration",
				fmt.Sprintf("The %s environment variable must be a positive integer, got: %q.", envName, raw),
			)
			return
		}

		*target = v
	}

	var chunkSize, parallelism types.Int64
	if model != nil {
		chunkSize = model.ChunkSize
		parallelism = model.Parallelism
	} else {
		chunkSize = types.Int64Null()
		parallelism = types.Int64Null()
	}

	resolve("chunk_size", envBulkChunkSize, chunkSize, &cfg.ChunkSize)
	resolve("parallelism", envBulkParallelism, parallelism, &cfg.Parallelism)

	return cfg, diags
}

// Configure configures the provider based on the provided configuration.
func (p *EntitleProvider) Configure(
	ctx context.Context,
	req provider.ConfigureRequest,
	resp *provider.ConfigureResponse,
) {
	config := new(EntitleProviderModel)

	// Retrieve configuration values from the request.
	diags := req.Config.Get(ctx, config)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Validate and set API key.
	token := os.Getenv("ENTITLE_API_KEY")
	if !config.APIKey.IsNull() && !config.APIKey.IsUnknown() {
		token = config.APIKey.ValueString()
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

	bulkConfig, bulkDiags := resolveBulkConfig(config.Bulk, os.Getenv)
	resp.Diagnostics.Append(bulkDiags...)

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

	// Build an HTTP client with a per-request timeout
	httpClient := &http.Client{
		Timeout: client.DefaultRequestTimeout,
	}

	// Wrap the HTTP client with retry logic.
	retryClient := client.NewRetryDoer(httpClient)

	c, err := client.NewClientWithResponses(
		server,
		client.WithHTTPClient(retryClient),
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
	providerData := &utils.ProviderData{
		Client: c,
		Bulk:   bulkConfig,
	}
	resp.DataSourceData = providerData
	resp.ResourceData = providerData

	tflog.Info(ctx, "Configured Entitle client", map[string]any{
		"success":          true,
		"bulk_chunk_size":  bulkConfig.ChunkSize,
		"bulk_parallelism": bulkConfig.Parallelism,
	})
}

// Resources returns the list of provider resources.
func (p *EntitleProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		accessRequestForwards.NewAccessRequestForwardResource,
		accessReviewForwards.NewAccessReviewForwardResource,
		agentTokens.NewAgentTokenResource,
		bundles.NewBundleResource,
		integrations.NewIntegrationResource,
		integrations.NewIntegrationGitlabResource,
		permissions.NewPermissionResource,
		policies.NewPolicyResource,
		resources.NewResourceResource,
		resources.NewResourceSyncedResource,
		resources.NewResourceTagsResource,
		roles.NewRoleResource,
		roles.NewRoleSyncedResource,
		workflows.NewWorkflowResource,
	}
}

// DataSources returns the list of provider data sources.
func (p *EntitleProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		accounts.NewAccountsDataSource,
		accessRequestForwards.NewAccessRequestForwardDataSource,
		accessReviewForwards.NewAccessReviewForwardDataSource,
		agentTokens.NewAgentTokenDataSource,
		applications.NewApplicationsDataSource,
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
