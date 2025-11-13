package bundles

import (
	"context"
	"fmt"
	"math/big"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/entitleio/terraform-provider-entitle/internal/client"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/utils"
	"github.com/entitleio/terraform-provider-entitle/internal/validators"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &BundleResource{}
var _ resource.ResourceWithImportState = &BundleResource{}

func NewBundleResource() resource.Resource {
	return &BundleResource{}
}

// BundleResource defines the resource implementation.
type BundleResource struct {
	client *client.ClientWithResponses
}

// BundleResourceModel describes the resource data model.
type BundleResourceModel struct {
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

	// Tags set of tags associated with the resource
	Tags types.Set `tfsdk:"tags" json:"tags"`

	// Roles list of roles associated with the resource
	Roles []*utils.Role `tfsdk:"roles" json:"roles"`
}

// Metadata is a function to set the TypeName for the Entitle bundle resource.
func (r *BundleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_bundle"
}

// Schema is a function to define the schema for the Entitle bundle resource.
func (r *BundleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
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
			// Attribute: id
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Entitle Bundle identifier in uuid format",
				Description:         "Entitle Bundle identifier in uuid format",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			// Attribute: name
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The name of the bundle. This is what users will reference when requesting access. Length must be between 2 and 50 characters.",
				Description:         "The name of the bundle. This is what users will reference when requesting access. Length must be between 2 and 50 characters.",
				Validators: []validator.String{
					validators.NewName(2, 50),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			// Attribute: description
			"description": schema.StringAttribute{
				Required: true,
				MarkdownDescription: "The bundle’s extended description, for example, " +
					"“Permissions bundle for junior accountants” or “factory floor worker permissions bundle”.",
				Description: "The bundle’s extended description, for example, " +
					"“Permissions bundle for junior accountants” or “factory floor worker permissions bundle”.",
			},
			// Attribute: category
			"category": schema.StringAttribute{
				Optional: true,
				MarkdownDescription: "You can select a category for the newly created bundle, or create a new one. " +
					"The category will usually describe a department, working group, etc. within your organization " +
					"like “Marketing”, “Operations” and so on.",
				Description: "You can select a category for the newly created bundle, or create a new one. " +
					"The category will usually describe a department, working group, etc. within your organization " +
					"like “Marketing”, “Operations” and so on.",
			},
			// Attribute: allowed_durations
			"allowed_durations": schema.ListAttribute{
				ElementType:         types.NumberType,
				Optional:            true,
				Description:         "You can override your organization’s default duration on each bundle",
				MarkdownDescription: "You can override your organization’s default duration on each bundle",
			},
			// Attribute: tags
			"tags": schema.SetAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "Any meta-data searchable tags should be added here, " +
					"like “accounting”, “ATL_Marketing” or “Production_Line_14”.",
				MarkdownDescription: "Any meta-data searchable tags should be added here, " +
					"like “accounting”, “ATL_Marketing” or “Production_Line_14”.",
			},
			// Attribute: workflow
			"workflow": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					// Attribute: workflow id
					"id": schema.StringAttribute{
						Required:            true,
						Description:         "The unique ID of the workflow assigned to the bundle.",
						MarkdownDescription: "The unique ID of the workflow assigned to the bundle.",
					},
					// Attribute: workflow name
					"name": schema.StringAttribute{
						Computed:            true,
						Description:         "The name of the assigned workflow.",
						MarkdownDescription: "The name of the assigned workflow.",
					},
				},
				Required:            true,
				Description:         "In this field, you can assign an existing workflow to the new bundle.",
				MarkdownDescription: "In this field, you can assign an existing workflow to the new bundle.",
			},
			// Attribute: roles
			"roles": schema.ListNestedAttribute{
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						// Attribute: role id
						"id": schema.StringAttribute{
							Optional:            true,
							Description:         "The unique ID of the role.",
							MarkdownDescription: "The unique ID of the role.",
						},
						// Attribute: role name
						"name": schema.StringAttribute{
							Computed:            true,
							Description:         "The name of the role.",
							MarkdownDescription: "The name of the role.",
						},
						// Attribute: resource
						"resource": schema.SingleNestedAttribute{
							Attributes: map[string]schema.Attribute{
								// Attribute: resource id
								"id": schema.StringAttribute{
									Computed:            true,
									Description:         "The unique identifier of the resource.",
									MarkdownDescription: "The unique identifier of the resource.",
								},
								// Attribute: resource name
								"name": schema.StringAttribute{
									Computed:            true,
									Description:         "The name of the resource.",
									MarkdownDescription: "The name of the resource.",
								},
								// Attribute: integration
								"integration": schema.SingleNestedAttribute{
									Attributes: map[string]schema.Attribute{
										// Attribute: integration id
										"id": schema.StringAttribute{
											Computed:            true,
											Description:         "The integration's ID.",
											MarkdownDescription: "The integration's ID.",
										},
										// Attribute: integration name
										"name": schema.StringAttribute{
											Computed:            true,
											Description:         "The integration's name.",
											MarkdownDescription: "The integration's name.",
										},
										// Attribute: application
										"application": schema.SingleNestedAttribute{
											Attributes: map[string]schema.Attribute{
												// Attribute: application name
												"name": schema.StringAttribute{
													Computed:            true,
													Description:         "The name of the application.",
													MarkdownDescription: "The name of the application.",
												},
											},
											Computed:            true,
											Description:         "The application linked to the integration.",
											MarkdownDescription: "The application linked to the integration.",
										},
									},
									Computed:            true,
									Description:         "The integration used to access the resource.",
									MarkdownDescription: "The integration used to access the resource.",
								},
							},
							Computed:            true,
							Description:         "The resource associated with the role.",
							MarkdownDescription: "The resource associated with the role.",
						},
					},
				},
				Required:            true,
				Description:         "List of roles included in the bundle.",
				MarkdownDescription: "List of roles included in the bundle.",
			},
		},
	}
}

// Configure is a function to set the client configuration for the BundleResource.
func (r *BundleResource) Configure(
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
			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = cli
}

// Create is responsible for creating a new resource of type Entitle Bundle.
//
// It reads the Terraform plan data provided in req.Plan and maps it to the BundleResourceModel.
// Then, it sends a request to the Entitle API to create the resource using API requests.
// If the creation is successful, it saves the resource's data into Terraform state.
func (r *BundleResource) Create(
	ctx context.Context,
	req resource.CreateRequest,
	resp *resource.CreateResponse,
) {
	var err error
	var plan BundleResourceModel

	// Read Terraform plan data into the model
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Process AllowedDurations
	allowedDurations := make([]client.EnumAllowedDurations, 0)
	if !plan.AllowedDurations.IsNull() && !plan.AllowedDurations.IsUnknown() {
		for _, item := range plan.AllowedDurations.Elements() {
			val, ok := item.(types.Number)
			if !ok {
				continue
			}

			val, diags := val.ToNumberValue(ctx)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}

			valFloat32, _ := val.ValueBigFloat().Float32()
			allowedDurations = append(allowedDurations, client.EnumAllowedDurations(valFloat32))
		}
	}

	// Process Tags
	tags := make([]string, 0)
	if !plan.Tags.IsNull() && !plan.Tags.IsUnknown() {
		for _, element := range plan.Tags.Elements() {
			if element.IsNull() || element.IsUnknown() {
				continue
			}

			tags = append(tags, utils.TrimPrefixSuffix(element.String()))
		}
	}

	// Process Roles
	roles := make([]client.IdParamsSchema, 0)
	if plan.Roles != nil {
		for _, role := range plan.Roles {
			if role.ID.IsNull() || role.ID.IsUnknown() {
				continue
			}

			parsedUUID, err := uuid.Parse(role.ID.ValueString())
			if err != nil {
				resp.Diagnostics.AddError(
					"Client Error",
					fmt.Sprintf("failed to parse the role id (%s) to UUID, got error: %s", role.ID.String(), err),
				)
				return
			}

			roles = append(roles, client.IdParamsSchema{
				Id: parsedUUID,
			})
		}
	}

	// Process Workflow
	var workflow client.IdParamsSchema
	if plan.Workflow != nil {
		workflow.Id, err = uuid.Parse(plan.Workflow.ID.String())
		if err != nil {
			resp.Diagnostics.AddError(
				"Client Error",
				fmt.Sprintf("failed to parse the workflow id (%s) to UUID, got error: %s", plan.Workflow.ID.String(), err),
			)
			return
		}
	} else {
		resp.Diagnostics.AddError(
			"Client Error",
			"failed to create bundle resource required workflow variable",
		)
		return
	}

	// Call Entitle API to create the bundle resource
	bundleResp, err := r.client.BundlesCreateWithResponse(ctx, client.PublicBundleCreateBodySchema{
		AllowedDurations: &allowedDurations,
		Category:         plan.Category.ValueStringPointer(),
		Description:      plan.Description.ValueStringPointer(),
		Name:             plan.Name.ValueString(),
		Roles:            roles,
		Tags:             &tags,
		Workflow:         workflow,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiConnection.Error(),
			fmt.Sprintf("Unable to create the bundle, got error: %v", err),
		)
		return
	}

	err = utils.HTTPResponseToError(bundleResp.HTTPResponse.StatusCode, bundleResp.Body, utils.WithIgnoreNotFound())
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiResponse.Error(),
			fmt.Sprintf(
				"Failed to create a Bundle status code: %d, %s",
				bundleResp.HTTPResponse.StatusCode,
				err.Error(),
			),
		)
		return
	}

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "created an Entitle bundle resource")

	// Update the plan with the new resource ID
	plan.ID = utils.TrimmedStringValue(bundleResp.JSON200.Result.Id.String())

	// Convert API response data to the model
	plan, diags = convertFullBundleResultResponseSchemaToModel(ctx, roles, &bundleResp.JSON200.Result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save data into Terraform state
	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read is used to read an existing resource of type Entitle Bundle.
//
// It retrieves the resource's data from the provider API requests.
// The retrieved data is then mapped to the BundleResourceModel,
// and the data is saved to Terraform state.
func (r *BundleResource) Read(
	ctx context.Context,
	req resource.ReadRequest,
	resp *resource.ReadResponse,
) {
	var data BundleResourceModel

	// Read Terraform prior state data into the model
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Parse the resource ID from the model
	uid, err := uuid.Parse(data.ID.String())
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("failed to parse the resource id (%s) to UUID, got error: %s", data.ID.String(), err),
		)
		return
	}

	// Call Entitle API to get the bundle resource by ID
	bundleResp, err := r.client.BundlesShowWithResponse(ctx, uid)
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiConnection.Error(),
			fmt.Sprintf("Unable to get the bundle by the id (%s), got error: %s", uid.String(), err),
		)
		return
	}

	err = utils.HTTPResponseToError(bundleResp.HTTPResponse.StatusCode, bundleResp.Body)
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiResponse.Error(),
			fmt.Sprintf(
				"Failed to get the Bundle by the id (%s), status code: %d, %s",
				uid.String(),
				bundleResp.HTTPResponse.StatusCode,
				err.Error(),
			),
		)
		return
	}

	// Convert API response data to the model
	data, diags = convertFullBundleResultResponseSchemaToModel(ctx, nil, &bundleResp.JSON200.Result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update handles updates to an existing resource of type Entitle Bundle.
//
// It reads the updated Terraform plan data provided in req.Plan and maps it to the BundleResourceModel.
// Then, it sends a request to the Entitle API to update the resource using API requests.
// If the update is successful, it saves the updated resource data into Terraform state.
func (r *BundleResource) Update(
	ctx context.Context,
	req resource.UpdateRequest,
	resp *resource.UpdateResponse,
) {
	var data BundleResourceModel

	// Read Terraform plan data into the model
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Parse the resource ID from the model
	uid, err := uuid.Parse(data.ID.String())
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("failed to parse the resource id (%s) to UUID, got error: %s", data.ID.String(), err),
		)
		return
	}

	// Process AllowedDurations
	allowedDurations := make([]client.EnumAllowedDurations, 0)
	if !data.AllowedDurations.IsNull() && !data.AllowedDurations.IsUnknown() {
		for _, item := range data.AllowedDurations.Elements() {
			val, ok := item.(types.Number)
			if !ok {
				continue
			}

			val, diags := val.ToNumberValue(ctx)
			resp.Diagnostics.Append(diags...)
			if resp.Diagnostics.HasError() {
				return
			}

			valFloat32, _ := val.ValueBigFloat().Float32()
			allowedDurations = append(allowedDurations, client.EnumAllowedDurations(valFloat32))
		}
	}

	// Process Tags
	tags := make([]string, 0)
	if !data.Tags.IsNull() && !data.Tags.IsUnknown() {
		for _, element := range data.Tags.Elements() {
			if element.IsNull() && element.IsUnknown() {
				continue
			}

			tags = append(tags, utils.TrimPrefixSuffix(element.String()))
		}
	}

	// Process Roles
	var roles *[]client.IdParamsSchema
	if data.Roles != nil {
		rolesTemp := make([]client.IdParamsSchema, 0)
		for _, role := range data.Roles {
			if role.ID.IsNull() || role.ID.IsUnknown() {
				continue
			}

			parsedUUID, err := uuid.Parse(role.ID.ValueString())
			if err != nil {
				resp.Diagnostics.AddError(
					"Client Error",
					fmt.Sprintf("failed to parse the role id (%s) to UUID, got error: %s", role.ID.String(), err),
				)
				return
			}

			rolesTemp = append(rolesTemp, client.IdParamsSchema{
				Id: parsedUUID,
			})
		}

		roles = &rolesTemp
	}

	// Process Workflow
	var workflow *client.IdParamsSchema
	if data.Workflow != nil {
		workflow = &client.IdParamsSchema{}
		workflow.Id, err = uuid.Parse(data.Workflow.ID.String())
		if err != nil {
			resp.Diagnostics.AddError(
				"Client Error",
				fmt.Sprintf("failed to parse the workflow id (%s) to UUID, got error: %s", data.Workflow.ID.String(), err),
			)
			return
		}
	}

	// Call Entitle API to update the bundle resource
	bundleResp, err := r.client.BundlesUpdateWithResponse(ctx, uid, client.BundleUpdatedBodySchema{
		AllowedDurations: &allowedDurations,
		Category:         data.Category.ValueStringPointer(),
		Description:      data.Description.ValueStringPointer(),
		Name:             data.Name.ValueStringPointer(),
		Roles:            roles,
		Tags:             utils.StringSlicePointer(tags),
		Workflow:         workflow,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiConnection.Error(),
			fmt.Sprintf("Unable to update the bundle by the id (%s), got error: %s", uid.String(), err),
		)
		return
	}

	err = utils.HTTPResponseToError(bundleResp.HTTPResponse.StatusCode, bundleResp.Body)
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiResponse.Error(),
			fmt.Sprintf(
				"Failed to update the Bundle by the id (%s), status code: %d, %s",
				uid.String(),
				bundleResp.HTTPResponse.StatusCode,
				err.Error(),
			),
		)
		return
	}

	// Convert API response data to the model
	data, diags = convertFullBundleResultResponseSchemaToModel(ctx, utils.IdParamsSchemaSliceValue(roles), &bundleResp.JSON200.Result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Save updated data into Terraform state
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete is responsible for deleting an existing resource of type Entitle Bundle.
//
// It reads the resource's data from Terraform state, extracts the unique identifier,
// and sends a request to delete the resource using API requests.
// If the deletion is successful, it removes the resource from Terraform state.
func (r *BundleResource) Delete(
	ctx context.Context,
	req resource.DeleteRequest,
	resp *resource.DeleteResponse,
) {
	var data BundleResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// Parse the resource ID from the model
	parsedUUID, err := uuid.Parse(data.ID.String())
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to parse uuid of the bundle, id: (%s), got error: %v", data.ID.String(), err),
		)
		return
	}

	// Call Entitle API to delete the bundle resource
	httpResp, err := r.client.BundlesDestroyWithResponse(ctx, parsedUUID)
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiConnection.Error(),
			fmt.Sprintf("Unable to delete bundle, id: (%s), got error: %v", data.ID.String(), err),
		)
		return
	}

	err = utils.HTTPResponseToError(httpResp.HTTPResponse.StatusCode, httpResp.Body, utils.WithIgnoreNotFound())
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiResponse.Error(),
			fmt.Sprintf(
				"Failed to delete the Bundle by the id (%s), status code: %d, %s",
				data.ID.String(),
				httpResp.HTTPResponse.StatusCode,
				err.Error(),
			),
		)
		return
	}
}

// ImportState this function is used to import an existing resource's state into Terraform.
//
// It extracts the resource's identifier from the import request and sets
// it in Terraform state using resource.ImportStatePassthroughID.
func (r *BundleResource) ImportState(
	ctx context.Context,
	req resource.ImportStateRequest,
	resp *resource.ImportStateResponse,
) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// convertFullBundleResultResponseSchemaToModel is a utility function used to convert the API response data
// (of type client.FullBundleResultResponseSchema) to a Terraform resource model (of type BundleResourceModel).
//
// It extracts and transforms data from the API response into a format that can be stored in Terraform state.
// It returns the converted model and any diagnostic information if there are errors during the conversion.
func convertFullBundleResultResponseSchemaToModel(
	ctx context.Context,
	plannedRoles []client.IdParamsSchema,
	data *client.FullBundleResultResponseSchema,
) (BundleResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	// Check if the API response data is nil
	if data == nil {
		diags.AddError(
			"No data",
			"Failed: the given schema data is nil",
		)

		return BundleResourceModel{}, diags
	}

	// Extract and convert allowed durations from the API response
	allowedDurationsValues := make([]attr.Value, len(data.AllowedDurations))
	for i, duration := range data.AllowedDurations {
		allowedDurationsValues[i] = types.NumberValue(big.NewFloat(float64(duration)))
	}

	allowedDurations, errs := types.ListValue(types.NumberType, allowedDurationsValues)
	diags.Append(errs...)
	if diags.HasError() {
		return BundleResourceModel{}, diags
	}

	// Extract tags from the API response
	tags, diagsTags := utils.GetStringSet(data.Tags)
	diags.Append(diagsTags...)
	if diags.HasError() {
		return BundleResourceModel{}, diags
	}

	var roles []*utils.Role
	var diagsRoles diag.Diagnostics
	if plannedRoles == nil {
		roles, diagsRoles = getRoles(ctx, data.Roles)
	} else {
		roles, diagsRoles = getRolesAsPlanned(ctx, plannedRoles, data.Roles)
	}

	diags.Append(diagsRoles...)
	if diags.HasError() {
		return BundleResourceModel{}, diags
	}

	category := utils.TrimmedStringValue(utils.StringValue(data.Category))
	if category.ValueString() == "" {
		category = types.StringNull()
	}

	// Create the Terraform resource model using the extracted data
	return BundleResourceModel{
		ID:               utils.TrimmedStringValue(data.Id.String()),
		Name:             utils.TrimmedStringValue(data.Name),
		Description:      utils.TrimmedStringValue(utils.StringValue(data.Description)),
		Category:         category,
		AllowedDurations: allowedDurations,
		Tags:             tags,
		Workflow:         getWorkflow(data.Workflow),
		Roles:            roles,
	}, diags
}
