// Package roles provides the implementation of the Entitle Agent Token resource for Terraform.
// It defines the resource type, its schema, and the CRUD operations for managing Agent Tokens.
package roles

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
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
var _ resource.Resource = &RoleResource{}
var _ resource.ResourceWithImportState = &RoleResource{}

// NewRoleResource creates a new instance of the RoleResource.
func NewRoleResource() resource.Resource {
	return &RoleResource{}
}

// RoleResource defines the resource implementation.
type RoleResource struct {
	client *client.ClientWithResponses
}

// RoleResourceModel describes the resource data model.
type RoleResourceModel struct {
	ID                      types.String                        `tfsdk:"id"`
	Name                    types.String                        `tfsdk:"name"`
	Resource                utils.IdNameModel                   `tfsdk:"resource" json:"resource"`
	AllowedDurations        types.Set                           `tfsdk:"allowed_durations"`
	Workflow                *utils.IdNameModel                  `tfsdk:"workflow"`
	PrerequisitePermissions []utils.PrerequisitePermissionModel `tfsdk:"prerequisite_permissions"`
	VirtualizedRole         *utils.IdNameModel                  `tfsdk:"virtualized_role"`
	Requestable             types.Bool                          `tfsdk:"requestable"`
}

// Metadata sets the metadata for the resource.
func (r *RoleResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

// Schema sets the schema for the resource.
func (r *RoleResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Defines the schema for an Entitle Role resource.",
		Description:         "Defines the schema for an Entitle Role resource.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Entitle Role identifier in UUID format",
				Description:         "Entitle Role identifier in UUID format",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The display name for Entitle Role.",
				Description:         "The display name for Entitle Role.",
				Validators: []validator.String{
					validators.NewName(2, 50),
				},
			},
			"resource": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Required:            true,
						Description:         "The unique ID of the resource assigned to the role.",
						MarkdownDescription: "The unique ID of the resource assigned to the role.",
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
				ElementType: types.NumberType,
				Optional:    true,
				Description: "As the admin, you can set different durations for the role, " +
					"compared to the workflow linked to it.",
				MarkdownDescription: "As the admin, you can set different durations for the role, " +
					"compared to the workflow linked to it.",
			},
			"workflow": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					// Attribute: workflow id
					"id": schema.StringAttribute{
						Required:            true,
						Description:         "The unique ID of the workflow assigned to the role.",
						MarkdownDescription: "The unique ID of the workflow assigned to the role.",
					},
					// Attribute: workflow name
					"name": schema.StringAttribute{
						Computed:            true,
						Description:         "The name of the assigned workflow.",
						MarkdownDescription: "The name of the assigned workflow.",
					},
				},
				Optional:            true,
				Description:         "In this field, you can assign an existing workflow to the new role.",
				MarkdownDescription: "In this field, you can assign an existing workflow to the new role.",
			},
			"prerequisite_permissions": schema.ListNestedAttribute{
				Optional:            true,
				Description:         "Users granted any role from this role through a request will automatically receive the permissions to the roles selected below.",
				MarkdownDescription: "Users granted any role from this role through a request will automatically receive the permissions to the roles selected below.",
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"default": schema.BoolAttribute{
							Optional:            true,
							Computed:            true,
							Default:             booldefault.StaticBool(false),
							Description:         "Indicates whether this prerequisite permission should be automatically granted as a default permission. When set to true, users will receive this permission by default when accessing the associated resource (default: false).",
							MarkdownDescription: "Indicates whether this prerequisite permission should be automatically granted as a default permission. When set to true, users will receive this permission by default when accessing the associated resource (default: false).",
						},
						"role": schema.SingleNestedAttribute{
							Required: true,
							Attributes: map[string]schema.Attribute{
								"id": schema.StringAttribute{
									Required:            true,
									Description:         "The identifier of the role to be granted.",
									MarkdownDescription: "The identifier of the role to be granted.",
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
						Required:            true,
						Description:         "The unique ID of the virtualized role assigned to the role.",
						MarkdownDescription: "The unique ID of the virtualized role assigned to the role.",
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
				Required:            true,
				MarkdownDescription: "Indicates if the role is requestable (default: true)",
				Description:         "Indicates if the role is requestable (default: true)",
			},
		},
	}
}

// Configure configures the resource with the provided client.
func (r *RoleResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
// It reads the Terraform plan data, maps it to the RoleResourceModel,
// sends a request to the Entitle API to create the resource, and saves the
// resource's data into Terraform state.
func (r *RoleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// Create an instance of the RoleResourceModel to store the resource data.
	var plan RoleResourceModel

	// Read Terraform plan data into the model.
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var err error

	var res client.IdParamsSchema
	res.Id, err = uuid.Parse(plan.Resource.ID.String())
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Failed to parse the resource id (%s) to UUID, got error: %s", plan.Workflow.ID.String(), err),
		)
		return
	}

	var workflow *client.IdParamsSchema
	if plan.Workflow != nil {
		workflow = new(client.IdParamsSchema)
		workflow.Id, err = uuid.Parse(plan.Workflow.ID.String())
		if err != nil {
			resp.Diagnostics.AddError(
				"Client Error",
				fmt.Sprintf("Failed to parse the workflow id (%s) to UUID, got error: %s", plan.Workflow.ID.String(), err),
			)
			return
		}
	}

	var virtualizedRole *client.IdParamsSchema
	if plan.VirtualizedRole != nil {
		virtualizedRole = new(client.IdParamsSchema)
		virtualizedRole.Id, err = uuid.Parse(plan.VirtualizedRole.ID.String())
		if err != nil {
			resp.Diagnostics.AddError(
				"Client Error",
				fmt.Sprintf("Failed to parse the virtualized role id (%s) to UUID, got error: %s", plan.Workflow.ID.String(), err),
			)
			return
		}
	}

	var allowedDurations *[]client.EnumAllowedDurations
	if !plan.AllowedDurations.IsNull() && !plan.AllowedDurations.IsUnknown() {
		allowedDurationsSlice := make([]client.EnumAllowedDurations, 0)

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

			allowedDurationsSlice = append(allowedDurationsSlice, client.EnumAllowedDurations(valFloat32))
		}
		allowedDurations = &allowedDurationsSlice
	}

	var prerequisitePermissions *[][]client.IntegrationResourceRoleCreateBodySchema_PrerequisitePermissions_Item
	if len(plan.PrerequisitePermissions) > 0 {
		ppData := make([][]client.IntegrationResourceRoleCreateBodySchema_PrerequisitePermissions_Item, 0, len(plan.PrerequisitePermissions))
		for _, pp := range plan.PrerequisitePermissions {
			if pp.Role.ID.IsNull() || pp.Role.ID.IsUnknown() {
				continue
			}

			item := client.IntegrationResourceRoleCreateBodySchema_PrerequisitePermissions_Item{}
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
				return
			}

			ppData = append(ppData, []client.IntegrationResourceRoleCreateBodySchema_PrerequisitePermissions_Item{
				item,
			})
		}
		prerequisitePermissions = &ppData
	}

	// Send a request to the Entitle API to create the role.
	apiResp, err := r.client.RolesCreateWithResponse(ctx, client.RolesCreateJSONRequestBody{
		AllowedDurations:        allowedDurations,
		Name:                    plan.Name.ValueStringPointer(),
		PrerequisitePermissions: prerequisitePermissions,
		Requestable:             plan.Requestable.ValueBool(),
		Resource:                res,
		VirtualizedRole:         virtualizedRole,
		Workflow:                workflow,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to create the role, got error: %v", err),
		)
		return
	}

	err = utils.HTTPResponseToError(apiResp.StatusCode(), apiResp.Body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf(
				"Failed to create the role: %s",
				err.Error(),
			),
		)
		return
	}

	// Write logs using the tflog package.
	tflog.Trace(ctx, "created an Entitle role resource")

	plan, diags = IntegrationResourceRoleResultSchemaToRoleResourceModel(ctx, apiResp.JSON200.Result)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	// Save the data into Terraform state.
	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read retrieves an existing resource of type Entitle Role.
//
// It retrieves the resource's data from the provider API requests,
// maps it to the RoleResourceModel, and saves the data to Terraform state.
func (r *RoleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Create an instance of the RoleResourceModel to store the resource data.
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
			"Client Error",
			fmt.Sprintf("Unable to get the role by the id (%s), got error: %s", uid.String(), err),
		)
		return
	}

	err = utils.HTTPResponseToError(apiResp.StatusCode(), apiResp.Body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf(
				"Failed to get the role by the id (%s), %s",
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

// Update handles updates to an existing resource of type Entitle Role.
//
// It reads the updated Terraform plan data, sends a request to the Entitle API
// to update the resource, and saves the updated resource data into Terraform state.
func (r *RoleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// Create an instance of the RoleResourceModel to store the resource data.
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
					fmt.Sprintf("Failed to merge preqrequisite permission data, error: %v", err),
				)
			}

			ppData = append(ppData, []client.IntegrationResourceRolesUpdateBodySchema_PrerequisitePermissions_Item{
				item,
			})
		}
		prerequisitePermissions = &ppData
	}

	var workflow client.IdParamsSchema
	if data.Workflow != nil {
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
		AllowedDurations:        &allowedDurations,
		PrerequisitePermissions: prerequisitePermissions,
		Requestable:             data.Requestable.ValueBoolPointer(),
		Workflow:                &workflow,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to update role by the id (%s), got error: %s", uid.String(), err),
		)
		return
	}

	err = utils.HTTPResponseToError(apiResp.StatusCode(), apiResp.Body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
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
func (r *RoleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Create an instance of the RoleResourceModel to store the resource data.
	var data RoleResourceModel

	// Read Terraform prior state data into the model.
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	// Check for errors in reading Terraform state data.
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

	// Send a request to the Entitle API to delete the role.
	httpResp, err := r.client.RoleDeleteWithResponse(ctx, uid)
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to delete role, id: (%s), got error: %v", data.ID.String(), err),
		)
		return
	}

	err = utils.HTTPResponseToError(httpResp.HTTPResponse.StatusCode, httpResp.Body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to delete role, id: (%s), %s",
				data.ID.String(), err.Error(),
			),
		)
		return
	}

	err = utils.HTTPResponseToError(httpResp.HTTPResponse.StatusCode, httpResp.Body)
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to delete role, id: (%s), %s", data.ID.String(), err.Error()),
		)
		return
	}
}

// ImportState is used to import an existing resource's state into Terraform.
//
// It extracts the resource's identifier from the import request and sets
// it in Terraform state using resource.ImportStatePassthroughID.
func (r *RoleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

func IntegrationResourceRoleResultSchemaToRoleResourceModel(ctx context.Context, data client.IntegrationResourceRoleResultSchema) (RoleResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	var workflow *utils.IdNameModel
	if data.Workflow != nil {
		workflow = &utils.IdNameModel{
			ID:   utils.TrimmedStringValue(data.Workflow.Id.String()),
			Name: utils.TrimmedStringValue(data.Workflow.Name),
		}
	}

	var prerequisitePermissions []utils.PrerequisitePermissionModel
	if data.PrerequisitePermissions != nil {
		for _, item := range *data.PrerequisitePermissions {
			v, err := item.AsPrerequisiteRolePermissionResponseSchema()
			if err != nil {
				diags.AddError(
					"No data",
					fmt.Sprintf("Failed to unmarshal the prerequisite permissions data, err: %s", err.Error()),
				)
				return RoleResourceModel{}, diags
			}

			roleModel, diagsGetRoles := utils.GetRole(ctx, v.Role.Id.String(), v.Role.Name, v.Role.Resource)
			if diagsGetRoles.HasError() {
				diags.Append(diagsGetRoles...)
				return RoleResourceModel{}, diags
			}

			prerequisitePermissions = append(prerequisitePermissions,
				utils.PrerequisitePermissionModel{
					Default: types.BoolValue(v.Default),
					Role:    roleModel,
				},
			)
		}
	}

	// Extract and convert allowed durations from the API response
	allowedDurationsValues, advDiags := utils.GetNumberSetFromAllowedDurations(data.AllowedDurations)
	if advDiags.HasError() {
		diags.Append(advDiags...)
		return RoleResourceModel{}, diags
	}

	var virtualizedRole *utils.IdNameModel
	if data.VirtualizedRole != nil {
		virtualizedRole = &utils.IdNameModel{
			ID:   utils.TrimmedStringValue(data.VirtualizedRole.Id.String()),
			Name: utils.TrimmedStringValue(data.VirtualizedRole.Name),
		}
	}

	return RoleResourceModel{
		ID:   utils.TrimmedStringValue(data.Id.String()),
		Name: utils.TrimmedStringValue(data.Name),
		Resource: utils.IdNameModel{
			ID:   utils.TrimmedStringValue(data.Resource.Id.String()),
			Name: utils.TrimmedStringValue(data.Resource.Name),
		},
		AllowedDurations:        allowedDurationsValues,
		Workflow:                workflow,
		PrerequisitePermissions: prerequisitePermissions,
		VirtualizedRole:         virtualizedRole,
		Requestable:             types.BoolValue(data.Requestable),
	}, diags
}
