package bundles

import (
	"context"
	"fmt"
	"math/big"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"

	"github.com/entitleio/terraform-provider-entitle/internal/provider/utils"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/entitleio/terraform-provider-entitle/internal/client"
	"github.com/entitleio/terraform-provider-entitle/internal/validators"
)

// Ensure that the provider-defined types fully satisfy the framework interfaces.
var _ datasource.DataSource = &BundleDataSource{}

// BundleDataSource defines the data source implementation for the Terraform provider.
type BundleDataSource struct {
	client *client.ClientWithResponses
}

// NewBundleDataSource creates a new instance of the BundleDataSource.
func NewBundleDataSource() datasource.DataSource {
	return &BundleDataSource{}
}

// BundleDataSourceModel defines the data model for FullBundleResultResponseSchema.
type BundleDataSourceModel struct {
	// ID identifier for the bundle resource in UUID format.
	ID types.String `tfsdk:"id" json:"id"`

	// Name the name of the bundle resource.
	Name types.String `tfsdk:"name" json:"name"`

	// Description the description of the bundle resource.
	Description types.String `tfsdk:"description" json:"description"`

	// Category the category of the resource.
	Category types.String `tfsdk:"category" json:"category"`

	// AllowedDurations the allowed durations for the resource
	AllowedDurations types.List `tfsdk:"allowed_durations" json:"allowedDurations"`

	// Workflow the id and name of the workflows associated with the resource
	Workflow *utils.IdNameModel `tfsdk:"workflow" json:"workflow"`

	// Tags list of tags associated with the resource
	Tags types.List `tfsdk:"tags" json:"tags"`

	// Roles list of roles associated with the resource
	Roles []*utils.Role `tfsdk:"roles" json:"roles"`
}

// Metadata sets the data source's metadata, such as its type name.
func (d *BundleDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bundle"
}

// Schema defines the expected structure of the data source.
func (d *BundleDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Entitle bundle is a set of entitlements that can be requested, approved, " +
			"or revoked by users in a single action, and set in a policy by the admin. Each entitlement can " +
			"provide the user with access to a resource, which can be as fine-grained as a MongoDB table " +
			"for example, usually by the use of a “Role”. Thus, one can think of a bundle " +
			"as a cross-application super role. [Read more about bundles](https://docs.beyondtrust.com/entitle/docs/bundles).",
		Description: "Entitle bundle is a set of entitlements that can be requested, approved, " +
			"or revoked by users in a single action, and set in a policy by the admin. Each entitlement can " +
			"provide the user with access to a resource, which can be as fine-grained as a MongoDB table " +
			"for example, usually by the use of a “Role”. Thus, one can think of a bundle " +
			"as a cross-application super role. [Read more about bundles](https://docs.beyondtrust.com/entitle/docs/bundles).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Entitle Bundle identifier in uuid format",
				Description:         "Entitle Bundle identifier in uuid format",
				Validators: []validator.String{
					validators.UUID{},
				},
			},
			"name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The bundle’s name. Users will ask for this name when requesting access.",
				Description:         "The bundle’s name. Users will ask for this name when requesting access.",
			},
			"description": schema.StringAttribute{
				Computed: true,
				MarkdownDescription: "The bundle’s extended description, for example, " +
					"“Permissions bundle for junior accountants” or “factory floor worker permissions bundle”.",
				Description: "The bundle’s extended description, for example, " +
					"“Permissions bundle for junior accountants” or “factory floor worker permissions bundle”.",
			},
			"category": schema.StringAttribute{
				Computed: true,
				MarkdownDescription: "You can select a category for the newly created bundle, or create a new one. " +
					"The category will usually describe a department, working group, etc. within your organization " +
					"like “Marketing”, “Operations” and so on.",
				Description: "You can select a category for the newly created bundle, or create a new one. " +
					"The category will usually describe a department, working group, etc. within your organization " +
					"like “Marketing”, “Operations” and so on.",
			},
			"allowed_durations": schema.ListAttribute{
				ElementType:         types.NumberType,
				Computed:            true,
				Description:         "You can override your organization’s default duration on each bundle",
				MarkdownDescription: "You can override your organization’s default duration on each bundle",
			},
			"tags": schema.ListAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "Any meta-data searchable tags should be added here, " +
					"like “accounting”, “ATL_Marketing” or “Production_Line_14”.",
				MarkdownDescription: "Any meta-data searchable tags should be added here, " +
					"like “accounting”, “ATL_Marketing” or “Production_Line_14”.",
			},
			"workflow": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Computed:            true,
						Description:         "Workflow's unique identifier.",
						MarkdownDescription: "Workflow's unique identifier.",
					},
					"name": schema.StringAttribute{
						Computed:            true,
						Description:         "workflow's name",
						MarkdownDescription: "workflow's name",
					},
				},
				Computed:            true,
				Description:         "In this field, you can assign an existing workflow to the new bundle.",
				MarkdownDescription: "In this field, you can assign an existing workflow to the new bundle.",
			},
			"roles": schema.ListNestedAttribute{
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Computed:            true,
							Description:         "Role's unique identifier.",
							MarkdownDescription: "Role's unique identifier.",
						},
						"name": schema.StringAttribute{
							Computed:            true,
							Description:         "Role's name.",
							MarkdownDescription: "Role's name.",
						},
						"resource": schema.SingleNestedAttribute{
							Attributes: map[string]schema.Attribute{
								"id": schema.StringAttribute{
									Computed:            true,
									Description:         "Resource's unique identifier.",
									MarkdownDescription: "Resource's unique identifier.",
								},
								"name": schema.StringAttribute{
									Computed:            true,
									Description:         "Resource's name.",
									MarkdownDescription: "Resource's name.",
								},
								"integration": schema.SingleNestedAttribute{
									Attributes: map[string]schema.Attribute{
										"id": schema.StringAttribute{
											Computed:            true,
											Description:         "Integration's unique identifier.",
											MarkdownDescription: "Integration's unique identifier.",
										},
										"name": schema.StringAttribute{
											Computed:            true,
											Description:         "Integration's name.",
											MarkdownDescription: "Integration's name.",
										},
										"application": schema.SingleNestedAttribute{
											Attributes: map[string]schema.Attribute{
												"name": schema.StringAttribute{
													Computed:            true,
													Description:         "Name of the application.",
													MarkdownDescription: "Name of the application.",
												},
											},
											Computed:            true,
											Description:         "Application of the integration.",
											MarkdownDescription: "Application of the integration.",
										},
									},
									Computed:            true,
									Description:         "Integration related to the resource.",
									MarkdownDescription: "Integration related to the resource.",
								},
							},
							Computed:            true,
							Description:         "The resource associated with the role.",
							MarkdownDescription: "The resource associated with the role.",
						},
					},
				},
				Computed:            true,
				Description:         "List of roles included in this bundle.",
				MarkdownDescription: "List of roles included in this bundle.",
			},
		},
	}
}

// Configure configures the data source with the provider's client.
func (d *BundleDataSource) Configure(
	ctx context.Context,
	req datasource.ConfigureRequest,
	resp *datasource.ConfigureResponse,
) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	cli, ok := req.ProviderData.(*client.ClientWithResponses)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = cli
}

// Read reads data from the external source and sets it in Terraform state.
// It retrieves the configuration data from Terraform, sends a request to the Entitle API to get the bundle data,
// converts the API response to the data source model, and saves it into Terraform state.
func (d *BundleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data BundleDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Parse the resource ID from the data source model
	uid, err := uuid.Parse(data.ID.String())
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("failed to parse the resource id (%s) to UUID, got error: %s", data.ID.String(), err),
		)
		return
	}

	// Log the start of the bundle GET operation with the resource ID
	tflog.Debug(ctx, "run bundle GET by id", map[string]interface{}{
		"uid": uid.String(),
	})

	// Send a request to the Entitle API to get the bundle data by ID
	bundleResp, err := d.client.BundlesShowWithResponse(ctx, uid)
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to get the bundle by the id (%s), got error: %s", uid.String(), err),
		)
		return
	}

	err = utils.HTTPResponseToError(bundleResp.HTTPResponse.StatusCode, bundleResp.Body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf(
				"Failed to get the Bundle by the id (%s), status code: %d, %s",
				uid.String(),
				bundleResp.HTTPResponse.StatusCode,
				err.Error(),
			),
		)
		return
	}

	// Log the start of converting the bundle response to the data source model
	tflog.Debug(ctx, "start convert bundle response to the data source model")

	// Convert the API response to the data source model
	data, diags := convertFullBundleResultResponseSchemaToBundleDataSourceModel(ctx, &bundleResp.JSON200.Result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save the converted data into Terraform state
	diagsResult := resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diagsResult...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Log a trace indicating the successful saving of the entitle bundle data source
	tflog.Trace(ctx, "saved entitle bundle data source successfully!")
}

// convertFullBundleResultResponseSchemaToBundleDataSourceModel converts the API response to the data source model.
// It takes the API response schema and converts it into the expected data source model,
// handling validations and conversions as necessary.
func convertFullBundleResultResponseSchemaToBundleDataSourceModel(
	ctx context.Context,
	data *client.FullBundleResultResponseSchema,
) (BundleDataSourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	// Check if the provided data is nil
	if data == nil {
		diags.AddError(
			"No data",
			"failed the given schema data is nil",
		)

		return BundleDataSourceModel{}, diags
	}

	// Extract and convert allowed durations from the API response
	allowedDurationsValues := make([]attr.Value, len(data.AllowedDurations))
	if data.AllowedDurations != nil {
		for i, duration := range data.AllowedDurations {
			allowedDurationsValues[i] = types.NumberValue(big.NewFloat(float64(duration)))
		}
	}

	allowedDurations, errs := types.ListValue(types.NumberType, allowedDurationsValues)
	diags.Append(errs...)
	if diags.HasError() {
		return BundleDataSourceModel{}, diags
	}

	// Extract and convert tags from the API response
	tags, diagsTags := utils.GetStringList(data.Tags)
	diags.Append(diagsTags...)
	if diags.HasError() {
		return BundleDataSourceModel{}, diags
	}

	// Extract and convert roles from the API response
	roles, diagsRoles := getRoles(ctx, data.Roles)
	diags.Append(diagsRoles...)
	if diags.HasError() {
		return BundleDataSourceModel{}, diags
	}

	// Create the BundleDataSourceModel with the converted data
	return BundleDataSourceModel{
		ID:               utils.TrimmedStringValue(data.Id.String()),
		Name:             utils.TrimmedStringValue(data.Name),
		Description:      utils.TrimmedStringValue(utils.StringValue(data.Description)),
		Category:         utils.TrimmedStringValue(utils.StringValue(data.Category)),
		AllowedDurations: allowedDurations,
		Tags:             tags,
		Workflow:         getWorkflow(data.Workflow),
		Roles:            roles,
	}, diags
}
