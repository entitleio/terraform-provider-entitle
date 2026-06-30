// Package roles provides the implementation of the Entitle Role resource for Terraform.
// It defines the resource type, its schema, and the CRUD operations for managing Roles.
package roles

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/entitleio/terraform-provider-entitle/docs"
	"github.com/entitleio/terraform-provider-entitle/internal/client"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/utils"
	"github.com/entitleio/terraform-provider-entitle/internal/validators"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &RoleSyncedResource{}
var _ resource.ResourceWithImportState = &RoleSyncedResource{}

// NewRoleSyncedResource creates a new instance of the RoleSyncedResource.
func NewRoleSyncedResource() resource.Resource {
	return &RoleSyncedResource{}
}

// RoleSyncedResource defines the resource implementation.
type RoleSyncedResource struct {
	client *client.ClientWithResponses
}

// Metadata sets the metadata for the resource.
func (r *RoleSyncedResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role_synced"
}

// Schema sets the schema for the resource.
func (r *RoleSyncedResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: docs.RoleSyncedResourceMarkdownDescription,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Entitle Role identifier in UUID format",
				Description:         "Entitle Role identifier in UUID format",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"external_id": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "The external ID of the role as assigned by the upstream integration. Used together with resource.id to look up the existing synced resource.",
				MarkdownDescription: "The external ID of the role as assigned by the upstream integration. Used together with resource.id to look up the existing synced resource.",
				Validators: []validator.String{
					stringvalidator.LengthBetween(2, 255),
					stringvalidator.ExactlyOneOf(
						path.MatchRoot("external_id"),
						path.MatchRoot("name"),
					),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "The name of the role as assigned by the upstream integration. Used together with resource.id to look up the existing synced resource.",
				MarkdownDescription: "The name of the role as assigned by the upstream integration. Used together with resource.id to look up the existing synced resource.",
				Validators: []validator.String{
					stringvalidator.LengthBetween(2, 255),
					stringvalidator.ExactlyOneOf(
						path.MatchRoot("external_id"),
						path.MatchRoot("name"),
					),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
					stringplanmodifier.RequiresReplace(),
				},
			},
			"resource": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Required:            true,
						Description:         "The unique ID of the resource assigned to the role.",
						MarkdownDescription: "The unique ID of the resource assigned to the role.",
						Validators: []validator.String{
							validators.UUID{},
						},
					},
					"name": schema.StringAttribute{
						Computed:            true,
						Description:         "The name of the assigned resource.",
						MarkdownDescription: "The name of the assigned resource.",
					},
				},
				Required:            true,
				Description:         "In this field, you can assign an existing resource to the new role.",
				MarkdownDescription: "In this field, you can assign an existing resource to the new role.",
			},
			"allowed_durations": schema.SetAttribute{
				ElementType:         types.NumberType,
				Optional:            true,
				Computed:            true,
				Description:         "As the admin, you can set different durations for the role, compared to the workflow linked to it. \nAllowed values:\n  - 1800 - 30min\n  - 3600 - 1 hour\n  - 10800 - 3 hours\n  - 21600 - 6 hours\n  - 43200 - 12 hours\n  - 57600 - 16 hours\n  - 86400 - 24 hours\n  - 259200 - 3 days\n  - 604800 - 7 days\n  - 2628000  - ~30,4 days\n  - 7884000 - 91,25 days\n  - 15768000 - 182,5 days\n  - 31536000 - 365 days\n  - 63072000 - 730 days\n  - -1 - unlimited",
				MarkdownDescription: "As the admin, you can set different durations for the role, compared to the workflow linked to it. \nAllowed values:\n  - 1800 - 30min\n  - 3600 - 1 hour\n  - 10800 - 3 hours\n  - 21600 - 6 hours\n  - 43200 - 12 hours\n  - 57600 - 16 hours\n  - 86400 - 24 hours\n  - 259200 - 3 days\n  - 604800 - 7 days\n  - 2628000  - ~30,4 days\n  - 7884000 - 91,25 days\n  - 15768000 - 182,5 days\n  - 31536000 - 365 days\n  - 63072000 - 730 days\n  - -1 - unlimited",
				Validators: []validator.Set{
					setvalidator.SizeAtLeast(1),
				},
				PlanModifiers: []planmodifier.Set{
					setplanmodifier.UseStateForUnknown(),
				},
			},
			"workflow": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					// Attribute: workflow id
					"id": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "The unique ID of the workflow assigned to the role.",
						MarkdownDescription: "The unique ID of the workflow assigned to the role.",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					// Attribute: workflow name
					"name": schema.StringAttribute{
						Computed:            true,
						Description:         "The name of the assigned workflow.",
						MarkdownDescription: "The name of the assigned workflow.",
					},
				},
				Optional:            true,
				Computed:            true,
				Description:         "In this field, you can assign an existing workflow to the new role.",
				MarkdownDescription: "In this field, you can assign an existing workflow to the new role.",
				PlanModifiers: []planmodifier.Object{
					objectplanmodifier.UseStateForUnknown(),
				},
			},
			"prerequisite_permissions": schema.ListNestedAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Users granted any role from this role through a request will automatically receive the permissions to the roles selected below.",
				MarkdownDescription: "Users granted any role from this role through a request will automatically receive the permissions to the roles selected below.",
				PlanModifiers: []planmodifier.List{
					listplanmodifier.UseStateForUnknown(),
				},
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"default": schema.BoolAttribute{
							Optional:            true,
							Computed:            true,
							Default:             booldefault.StaticBool(false),
							Description:         "Indicates whether this prerequisite permission should be automatically granted as a default permission. When set to true, users will receive this permission by default when accessing the associated resource (default: false).",
							MarkdownDescription: "Indicates whether this prerequisite permission should be automatically granted as a default permission. When set to true, users will receive this permission by default when accessing the associated resource (default: false).",
							PlanModifiers: []planmodifier.Bool{
								boolplanmodifier.UseStateForUnknown(),
							},
						},
						"role": schema.SingleNestedAttribute{
							Optional: true,
							Computed: true,
							Attributes: map[string]schema.Attribute{
								"id": schema.StringAttribute{
									Optional:            true,
									Computed:            true,
									Description:         "The identifier of the role to be granted.",
									MarkdownDescription: "The identifier of the role to be granted.",
									PlanModifiers: []planmodifier.String{
										stringplanmodifier.UseStateForUnknown(),
									},
								},
								"name": schema.StringAttribute{
									Computed:            true,
									Description:         "The name of the role.",
									MarkdownDescription: "The name of the role.",
								},
								"resource": schema.SingleNestedAttribute{
									Attributes: map[string]schema.Attribute{
										"id": schema.StringAttribute{
											Computed:            true,
											Description:         "The unique identifier of the resource.",
											MarkdownDescription: "The unique identifier of the resource.",
										},
										"name": schema.StringAttribute{
											Computed:            true,
											Description:         "The display name of the resource.",
											MarkdownDescription: "The display name of the resource.",
										},
										"integration": schema.SingleNestedAttribute{
											Attributes: map[string]schema.Attribute{
												"id": schema.StringAttribute{
													Computed:            true,
													Description:         "The identifier of the integration.",
													MarkdownDescription: "The identifier of the integration.",
												},
												"name": schema.StringAttribute{
													Computed:            true,
													Description:         "The display name of the integration.",
													MarkdownDescription: "The display name of the integration.",
												},
												"application": schema.SingleNestedAttribute{
													Attributes: map[string]schema.Attribute{
														"name": schema.StringAttribute{
															Computed:            true,
															Description:         "The name of the connected application.",
															MarkdownDescription: "The name of the connected application.",
														},
													},
													Computed:            true,
													Description:         "The application that the integration is connected to.",
													MarkdownDescription: "The application that the integration is connected to.",
												},
											},
											Computed:            true,
											Description:         "The integration that the resource belongs to.",
											MarkdownDescription: "The integration that the resource belongs to.",
										},
									},
									Computed:            true,
									Description:         "The specific resource associated with the role.",
									MarkdownDescription: "The specific resource associated with the role.",
								},
							},
						},
					},
				},
			},
			"virtualized_role": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "The unique ID of the virtualized role assigned to the role.",
						MarkdownDescription: "The unique ID of the virtualized role assigned to the role.",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
					"name": schema.StringAttribute{
						Computed:            true,
						Description:         "The name of the assigned virtualized role.",
						MarkdownDescription: "The name of the assigned virtualized role.",
					},
				},
				Optional:            true,
				Description:         "In this field, you can assign an existing virtualized role to the new role.",
				MarkdownDescription: "In this field, you can assign an existing virtualized role to the new role.",
			},
			"requestable": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Indicates if the role is requestable.",
				Description:         "Indicates if the role is requestable.",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Configure configures the resource with the provided client.
func (r *RoleSyncedResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.ClientWithResponses)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = c
}

// Create handles the creation of a new resource of type Entitle Role.
//
// It reads the Terraform plan data, maps it to the RoleSyncedResourceModel,
// sends a request to the Entitle API to create the resource, and saves the
// resource's data into Terraform state.
func (r *RoleSyncedResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Use a plan-read struct that absorbs unknown values for computed fields,
	// since on first apply there is no prior state to resolve them from.
	var createPlan roleSyncedCreatePlan

	// Read Terraform plan data into the model.
	diags := req.Plan.Get(ctx, &createPlan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var err error

	name := createPlan.Name.ValueStringPointer()
	externalID := createPlan.ExternalID.ValueStringPointer()
	resourceID := createPlan.Resource.ID.ValueString()
	roleID, err := r.getRoleID(ctx, uuid.MustParse(resourceID), externalID, name)
	if err != nil {
		resp.Diagnostics.AddError("Role not found", fmt.Sprintf(
			"Failed to get the Role by the name (%s) or external id (%s) and resource (%s), %s",
			createPlan.Name.ValueString(), createPlan.ExternalID.ValueString(), resourceID,
			err.Error(),
		))

		return
	}

	apiResp, err := r.client.RolesShowWithResponse(ctx, *roleID)
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiConnection.Error(),
			fmt.Sprintf("Unable to get the role. If you want to create new then use entitle_role. Got error: %v", err),
		)
		return
	}

	err = utils.HTTPResponseToError(apiResp.StatusCode(), apiResp.Body)
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiResponse.Error(),
			fmt.Sprintf(
				"Failed to get the role: %s",
				err.Error(),
			),
		)
		return
	}

	if !utils.IsApplicationWithSyncedResources(apiResp.JSON200.Result.Resource.Integration.Application.Name) {
		resp.Diagnostics.AddError(
			utils.ErrApiResponse.Error(),
			"Got resource created manually, use entitle_role resource instead.",
		)
		return
	}

	state, diags := IntegrationResourceRoleResultSchemaToRoleResourceModel(ctx, apiResp.JSON200.Result)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	// Save the data into Terraform state.
	diags = resp.State.Set(ctx, &state)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	r.compareAndUpdate(ctx, createPlan, apiResp.JSON200.Result, resp)
	if resp.Diagnostics.HasError() {
		return
	}
}

type roleSyncedCreatePlan struct {
	Name                    types.String       `tfsdk:"name"`
	ExternalID              types.String       `tfsdk:"external_id"`
	Resource                *utils.IdNameModel `tfsdk:"resource"`
	ID                      types.String       `tfsdk:"id"`
	AllowedDurations        types.Set          `tfsdk:"allowed_durations"`
	Workflow                types.Object       `tfsdk:"workflow"`
	PrerequisitePermissions types.List         `tfsdk:"prerequisite_permissions"`
	VirtualizedRole         types.Object       `tfsdk:"virtualized_role"`
	Requestable             types.Bool         `tfsdk:"requestable"`
}

func (r *RoleSyncedResource) compareAndUpdate(ctx context.Context, plan roleSyncedCreatePlan, result client.IntegrationResourceRoleResultSchema, resp *resource.CreateResponse) {
	var updateRequest client.RolesUpdateJSONRequestBody
	diff := false

	// Requestable
	if !plan.Requestable.IsNull() && !plan.Requestable.IsUnknown() {
		if plan.Requestable.ValueBool() != result.Requestable {
			diff = true
			updateRequest.Requestable = plan.Requestable.ValueBoolPointer()
		}
	}

	// AllowedDurations
	if !plan.AllowedDurations.IsNull() && !plan.AllowedDurations.IsUnknown() {
		planDurations, diags := utils.GetEnumAllowedDurationsSliceFromNumberSet(ctx, plan.AllowedDurations)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
		if !utils.AllowedDurationsEqual(planDurations, result.AllowedDurations) {
			diff = true
			updateRequest.AllowedDurations = &planDurations
		}
	}

	// Workflow
	if !plan.Workflow.IsNull() && !plan.Workflow.IsUnknown() {
		if idAttr, ok := plan.Workflow.Attributes()["id"].(types.String); ok && !idAttr.IsNull() && !idAttr.IsUnknown() {
			planWorkflowID := idAttr.ValueString()
			if result.Workflow == nil || result.Workflow.Id.String() != planWorkflowID {
				diff = true
				wfID, err := uuid.Parse(planWorkflowID)
				if err != nil {
					resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Failed to parse workflow id to UUID: %v", err))
					return
				}
				updateRequest.Workflow = &client.IdParamsSchema{Id: wfID}
			}
		}
	}

	// PrerequisitePermissions — include if set in plan
	if !plan.PrerequisitePermissions.IsNull() && !plan.PrerequisitePermissions.IsUnknown() && len(plan.PrerequisitePermissions.Elements()) > 0 {
		var ppModels []utils.PrerequisitePermissionModel
		if diags := plan.PrerequisitePermissions.ElementsAs(ctx, &ppModels, false); diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
		ppData := make([][]client.IntegrationResourceRolesUpdateBodySchema_PrerequisitePermissions_Item, 0, len(ppModels))
		for _, pp := range ppModels {
			if pp.Role == nil || pp.Role.ID.IsNull() || pp.Role.ID.IsUnknown() {
				continue
			}
			item := client.IntegrationResourceRolesUpdateBodySchema_PrerequisitePermissions_Item{}
			mergeErr := item.MergePrerequisitePermissionCreateBodySchema(client.PrerequisitePermissionCreateBodySchema{
				Default: pp.Default.ValueBool(),
				Role: map[string]interface{}{
					"id": pp.Role.ID.ValueString(),
				},
			})
			if mergeErr != nil {
				resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Failed to merge prerequisite permission data, error: %v", mergeErr))
				return
			}
			ppData = append(ppData, []client.IntegrationResourceRolesUpdateBodySchema_PrerequisitePermissions_Item{item})
		}
		if len(ppData) > 0 {
			diff = true
			updateRequest.PrerequisitePermissions = &ppData
		}
	}

	if !diff {
		return
	}

	apiResp, err := r.client.RolesUpdateWithResponse(ctx, result.Id, updateRequest)
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiConnection.Error(),
			fmt.Sprintf("Unable to update the role. Got error: %v", err),
		)
		return
	}

	err = utils.HTTPResponseToError(apiResp.StatusCode(), apiResp.Body)
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiResponse.Error(),
			fmt.Sprintf("Failed to update the role: %s", err.Error()),
		)
		return
	}

	data, diags := IntegrationResourceRoleResultSchemaToRoleResourceModel(ctx, apiResp.JSON200.Result)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	// Save the updated data into Terraform state.
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read retrieves an existing resource of type Entitle Role.
//
// It retrieves the resource's data from the provider API requests,
// maps it to the RoleSyncedResourceModel, and saves the data to Terraform state.
func (r *RoleSyncedResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Create an instance of the RoleSyncedResourceModel to store the resource data.
	var data RoleResourceModel

	// Read Terraform prior state data into the model.
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Parse the resource ID into a UUID for API request.
	uid, err := uuid.Parse(data.ID.String())
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Failed to parse the resource id (%s) to UUID, got error: %s", data.ID.String(), err),
		)
		return
	}

	// Retrieve the role details from the Entitle API.
	apiResp, err := r.client.RolesShowWithResponse(ctx, uid)
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiConnection.Error(),
			fmt.Sprintf("Unable to get the role by the id (%s), got error: %s", uid.String(), err),
		)
		return
	}

	err = utils.HTTPResponseToError(apiResp.StatusCode(), apiResp.Body)
	if err != nil {
		if errors.Is(err, utils.ErrNotFound) {
			tflog.Debug(ctx, "Resource no longer exists, removing from state")

			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			utils.ErrApiResponse.Error(),
			fmt.Sprintf(
				"Failed to get the role by the id (%s), %s",
				uid.String(),
				err.Error(),
			),
		)
		return
	}

	if !utils.IsApplicationWithSyncedResources(apiResp.JSON200.Result.Resource.Integration.Application.Name) {
		resp.Diagnostics.AddError(
			utils.ErrApiResponse.Error(),
			"Got resource created manually, use entitle_role resource instead.",
		)
		return
	}

	data, diags = IntegrationResourceRoleResultSchemaToRoleResourceModel(ctx, apiResp.JSON200.Result)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	// Save the updated data into Terraform state.
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Update handles updates to an existing resource of type Entitle Role.
//
// It reads the updated Terraform plan data, sends a request to the Entitle API
// to update the resource, and saves the updated resource data into Terraform state.
func (r *RoleSyncedResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Create an instance of the RoleSyncedResourceModel to store the resource data.
	var data RoleResourceModel

	// Read Terraform plan data into the model.
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Parse the unique identifier from the resource data.
	uid, err := uuid.Parse(data.ID.String())
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Failed to parse the given id to UUID format, got error: %v", err),
		)
		return
	}

	var allowedDurations *[]client.EnumAllowedDurations
	if !data.AllowedDurations.IsNull() && !data.AllowedDurations.IsUnknown() {
		aDurations, diags := utils.GetEnumAllowedDurationsSliceFromNumberSet(ctx, data.AllowedDurations)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}

		allowedDurations = &aDurations
	}

	var prerequisitePermissions *[][]client.IntegrationResourceRolesUpdateBodySchema_PrerequisitePermissions_Item
	if len(data.PrerequisitePermissions) > 0 {
		ppData := make([][]client.IntegrationResourceRolesUpdateBodySchema_PrerequisitePermissions_Item, 0, len(data.PrerequisitePermissions))
		for _, pp := range data.PrerequisitePermissions {
			if pp.Role.ID.IsNull() || pp.Role.ID.IsUnknown() {
				continue
			}

			item := client.IntegrationResourceRolesUpdateBodySchema_PrerequisitePermissions_Item{}
			err := item.MergePrerequisitePermissionCreateBodySchema(client.PrerequisitePermissionCreateBodySchema{
				Default: pp.Default.ValueBool(),
				Role: map[string]interface{}{
					"id": pp.Role.ID.ValueString(),
				},
			})
			if err != nil {
				resp.Diagnostics.AddError(
					"Client Error",
					fmt.Sprintf("Failed to merge prerequisite permission data, error: %v", err),
				)
			}

			ppData = append(ppData, []client.IntegrationResourceRolesUpdateBodySchema_PrerequisitePermissions_Item{
				item,
			})
		}
		prerequisitePermissions = &ppData
	}

	var workflow *client.IdParamsSchema
	if data.Workflow != nil && !data.Workflow.ID.IsNull() && !data.Workflow.ID.IsUnknown() {
		workflow = new(client.IdParamsSchema)
		workflow.Id, err = uuid.Parse(data.Workflow.ID.String())
		if err != nil {
			resp.Diagnostics.AddError(
				"Client Error",
				fmt.Sprintf("Failed to parse the workflow id (%s) to UUID, got error: %s", data.Workflow.ID.String(), err),
			)
			return
		}
	}

	// Send a request to the Entitle API to update the role.
	apiResp, err := r.client.RolesUpdateWithResponse(ctx, uid, client.RolesUpdateJSONRequestBody{
		AllowedDurations:        allowedDurations,
		PrerequisitePermissions: prerequisitePermissions,
		Requestable:             data.Requestable.ValueBoolPointer(),
		Workflow:                workflow,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiConnection.Error(),
			fmt.Sprintf("Unable to update role by the id (%s), got error: %s", uid.String(), err),
		)
		return
	}

	err = utils.HTTPResponseToError(apiResp.StatusCode(), apiResp.Body)
	if err != nil {
		if errors.Is(err, utils.ErrNotFound) {
			tflog.Debug(ctx, "Resource no longer exists, removing from state")

			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError(
			utils.ErrApiResponse.Error(),
			fmt.Sprintf(
				"Failed to update the role by the id (%s),%s",
				uid.String(),
				err.Error(),
			),
		)
		return
	}

	data, diags = IntegrationResourceRoleResultSchemaToRoleResourceModel(ctx, apiResp.JSON200.Result)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	// Save the updated data into Terraform state.
	diags = resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Delete is responsible for deleting an existing resource of type Entitle Role.
//
// It reads the resource's data from Terraform state, extracts the unique identifier,
// and sends a request to delete the resource using API requests. If the deletion
// is successful, it removes the resource from Terraform state.
func (r *RoleSyncedResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Create an instance of the RoleSyncedResourceModel to store the resource data.
	var data RoleResourceModel

	// Read Terraform prior state data into the model.
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	// Check for errors in reading Terraform state data.
	if resp.Diagnostics.HasError() {
		return
	}

	// No action needed
}

// ImportState is used to import an existing resource's state into Terraform.
//
// It extracts the resource's identifier from the import request and sets
// it in Terraform state using resource.ImportStatePassthroughID.
func (r *RoleSyncedResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// getRoleID paginates through IntegrationResourceListItemResponseSchema to find a role by exact name or external id
// within the given integrationId.
func (r *RoleSyncedResource) getRoleID(ctx context.Context, resourceID uuid.UUID, externalID, name *string) (*uuid.UUID, error) {
	fetch := func(ctx context.Context, page int) ([]client.IntegrationResourceRoleListItemResponseSchema, int, error) {
		params := client.RolesIndexParams{
			PerPage:    utils.IntPointer(100),
			Page:       utils.IntPointer(page),
			Search:     name,
			ResourceId: &resourceID,
			ExternalId: externalID,
		}

		resp, err := r.client.RolesIndexWithResponse(ctx, &params)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to list roles: %w", err)
		}

		if resp.HTTPResponse.StatusCode >= http.StatusBadRequest {
			return nil, 0, fmt.Errorf("API returned status %d while listing roles (page %d)",
				resp.HTTPResponse.StatusCode, page)
		}

		if resp.JSON200 == nil || resp.JSON200.Result == nil {
			return nil, 0, fmt.Errorf("received invalid role response structure (page %d)", page)
		}

		items := resp.JSON200.Result
		total := int(resp.JSON200.Pagination.TotalPages)
		return items, total, nil
	}

	if externalID != nil && *externalID != "" {
		return utils.FindIDByExternalID(ctx, *externalID, fetch)
	}
	if name != nil && *name != "" {
		return utils.FindIDByName(ctx, *name, fetch)
	}

	return nil, fmt.Errorf("name or externalId must be set")
}
