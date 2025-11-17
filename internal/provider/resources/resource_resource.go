package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/entitleio/terraform-provider-entitle/docs"
	"github.com/entitleio/terraform-provider-entitle/internal/client"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/utils"
	"github.com/entitleio/terraform-provider-entitle/internal/validators"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ResourceResource{}
var _ resource.ResourceWithImportState = &ResourceResource{}

func NewResourceResource() resource.Resource {
	return &ResourceResource{}
}

// ResourceResource defines the resource implementation.
type ResourceResource struct {
	client *client.ClientWithResponses
}

// ResourceResourceModel describes the resource data model.
type ResourceResourceModel struct {
	ID                      types.String                        `tfsdk:"id"`
	Name                    types.String                        `tfsdk:"name"`
	AllowedDurations        types.Set                           `tfsdk:"allowed_durations"`
	Maintainers             []*utils.MaintainerModel            `tfsdk:"maintainers"`
	Tags                    types.Set                           `tfsdk:"tags"`
	UserDefinedTags         types.Set                           `tfsdk:"user_defined_tags"`
	UserDefinedDescription  types.String                        `tfsdk:"user_defined_description"`
	Workflow                *utils.IdNameModel                  `tfsdk:"workflow"`
	Integration             utils.IdNameModel                   `tfsdk:"integration"`
	PrerequisitePermissions []utils.PrerequisitePermissionModel `tfsdk:"prerequisite_permissions"`
	Requestable             types.Bool                          `tfsdk:"requestable"`
	Owner                   *utils.IdEmailModel                 `tfsdk:"owner"`
}

func (r *ResourceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resource"
}

func (r *ResourceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: docs.ResourceResourceMarkdownDescription,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Entitle Resource identifier in uuid format",
				Description:         "Entitle Resource identifier in uuid format",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The display name for the resource. Length between 2 and 50.",
				Description:         "The display name for the resource. Length between 2 and 50.",
				Validators: []validator.String{
					validators.NewName(2, 50),
				},
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"allowed_durations": schema.SetAttribute{
				ElementType: types.NumberType,
				Optional:    true,
				Description: "As the admin, you can set different durations for the resource, " +
					"compared to the workflow linked to it.",
				MarkdownDescription: "As the admin, you can set different durations for the resource, " +
					"compared to the workflow linked to it.",
				Validators: []validator.Set{
					validators.NewSetMinLength(1),
				},
			},
			"maintainers": schema.ListNestedAttribute{
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"type": schema.StringAttribute{
							Optional:            true,
							Description:         "\"user\" or \"group\" (default: \"user\")",
							MarkdownDescription: "\"user\" or \"group\" (default: \"user\")",
							Computed:            true,
							Default:             stringdefault.StaticString("user"),
						},
						"entity": schema.SingleNestedAttribute{
							Attributes: map[string]schema.Attribute{
								"id": schema.StringAttribute{
									Required:            true,
									Description:         "Maintainer's unique identifier",
									MarkdownDescription: "Maintainer's unique identifier",
								},
								"email": schema.StringAttribute{
									Computed:            true,
									Description:         "Maintainer's email",
									MarkdownDescription: "Maintainer's email",
								},
							},
							Optional:            true,
							Description:         "Maintainer's entity",
							MarkdownDescription: "Maintainer's entity",
						},
					},
				},
				Optional: true,
				Description: "Maintainer of the resource, second tier owner of that resource you can " +
					"have multiple resource Maintainer also can be IDP group. In the case of the bundle the Maintainer of each Resource.",
				MarkdownDescription: "Maintainer of the resource, second tier owner of that resource you can " +
					"have multiple resource Maintainer also can be IDP group. In the case of the bundle the Maintainer of each Resource.",
			},
			"tags": schema.SetAttribute{
				ElementType: types.StringType,
				Computed:    true,
				Description: "Any meta-data searchable tags should be added here, " +
					"like “accounting”, “ATL_Marketing” or “Production_Line_14”.",
				MarkdownDescription: "Any meta-data searchable tags should be added here, " +
					"like “accounting”, “ATL_Marketing” or “Production_Line_14”.",
			},
			"user_defined_tags": schema.SetAttribute{
				ElementType: types.StringType,
				Optional:    true,
				Description: "Any meta-data searchable tags should be added here, " +
					"like “accounting”, “ATL_Marketing” or “Production_Line_14”.",
				MarkdownDescription: "Any meta-data searchable tags should be added here, " +
					"like “accounting”, “ATL_Marketing” or “Production_Line_14”.",
				Validators: []validator.Set{
					validators.NewSetMinLength(1),
				},
			},
			"user_defined_description": schema.StringAttribute{
				Optional: true,
				Validators: []validator.String{
					validators.NewName(2, 2048),
				},
			},
			"workflow": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Optional:            true,
						Description:         "the workflow's id",
						MarkdownDescription: "the workflow's id",
					},
					"name": schema.StringAttribute{
						Computed:            true,
						Description:         "the workflow's name",
						MarkdownDescription: "the workflow's name",
					},
				},
				Optional:            true,
				Description:         "The default approval workflow for entitlements for the resource",
				MarkdownDescription: "The default approval workflow for entitlements for the resource",
			},
			"integration": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Required:            true,
						Description:         "the integration's id",
						MarkdownDescription: "the integration's id",
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.RequiresReplace(),
						},
					},
					"name": schema.StringAttribute{
						Computed:            true,
						Description:         "the integration's name",
						MarkdownDescription: "the integration's name",
					},
				},
				Required:            true,
				Description:         "Integration the resource belongs to",
				MarkdownDescription: "Integration the resource belongs to",
			},
			"requestable": schema.BoolAttribute{
				Required:            true,
				Description:         "Indicates if the resource is requestable",
				MarkdownDescription: "Indicates if the resource is requestable",
			},
			"owner": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "the owner's id",
						MarkdownDescription: "the owner's id",
					},
					"email": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						Description:         "the owner's email (lowercase) is used when id was not provided",
						MarkdownDescription: "the owner's email (lowercase) is used when id was not provided",
						//Validators:          []validator.String{
						//validators.LowerCaseNameValidator{},
						//},
					},
				},
				Optional: true,
				Description: "Define the owner of the resource, which will be used for administrative " +
					"purposes and approval workflows.",
				MarkdownDescription: "Define the owner of the resource, which will be used for administrative " +
					"purposes and approval workflows.",
			},
			"prerequisite_permissions": schema.ListNestedAttribute{
				Optional:            true,
				Description:         "Users granted any role from this resource through a request will automatically receive the permissions to the roles selected below.",
				MarkdownDescription: "Users granted any role from this resource through a request will automatically receive the permissions to the roles selected below.",
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
		},
	}
}

func (r *ResourceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Create this function is responsible for creating a new resource of type Entitle Resource.
//
// Its reads the Terraform plan data provided in req.Plan and maps it to the ResourceResourceModel.
// And sends a request to the Entitle API to create the resource using API requests.
// If the creation is successful, it saves the resource's data into Terraform state.
func (r *ResourceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var err error
	var plan ResourceResourceModel

	// Read Terraform plan data into the model
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := plan.Name.ValueString()

	allowedDurations, diags := utils.ConvertTerraformSetToAllowedDurations(ctx, plan.AllowedDurations)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	var workflow client.IdParamsSchema
	if plan.Workflow != nil {
		if !plan.Workflow.ID.IsNull() && !plan.Workflow.ID.IsUnknown() {
			workflow.Id, err = uuid.Parse(plan.Workflow.ID.String())
			if err != nil {
				resp.Diagnostics.AddError(
					"Client Error",
					fmt.Sprintf("Failed to parse given workflow id to UUID, got error: %v", err),
				)
				return
			}
		}
	}

	var integration client.IdParamsSchema
	integration.Id, err = uuid.Parse(plan.Integration.ID.String())
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Failed to parse given integration id to UUID, got error: %v", err),
		)
		return
	}

	var owner client.UserEntitySchema
	if plan.Owner != nil {
		if v := plan.Owner.Id.ValueString(); v != "" {
			owner.Id = utils.TrimPrefixSuffix(v)
		} else if v := plan.Owner.Email.ValueString(); v != "" {
			owner.Id = strings.ToLower(utils.TrimPrefixSuffix(v))
		} else {
			resp.Diagnostics.AddError(
				"Config Error",
				"Missing the owner's identifier for entitle resource",
			)
			return
		}
	}

	maintainers := make([]client.IntegrationResourcesCreateBodySchema_Maintainers_Item, 0)
	for _, maintainer := range plan.Maintainers {
		if maintainer.Type.IsNull() || maintainer.Type.IsUnknown() {
			continue
		}

		if maintainer.Entity.IsNull() {
			resp.Diagnostics.AddError(
				"Client Error",
				"Missing data for entity maintainer",
			)
			return
		}

		idAttr := maintainer.Entity.Attributes()["id"]
		strVal, ok := idAttr.(basetypes.StringValue)
		if !ok {
			resp.Diagnostics.AddError(
				"Client Error",
				"Missing data for entity maintainer id",
			)
			return
		}
		entityID := strVal.ValueString()

		switch maintainer.Type.ValueString() {
		case utils.MaintainerTypeUser:
			maintainerUser := client.UserMaintainerSchema{
				Type: client.EnumMaintainerTypeUserUser,
				User: client.UserEntitySchema{
					Id: entityID,
				},
			}

			item := client.IntegrationResourcesCreateBodySchema_Maintainers_Item{}
			err := item.MergeUserMaintainerSchema(maintainerUser)
			if err != nil {
				resp.Diagnostics.AddError(
					"Client Error",
					fmt.Sprintf("Failed to merge user maintainer data, error: %v", err),
				)
			}

			maintainers = append(maintainers, item)
		case utils.MaintainerTypeGroup:
			maintainerGroup := client.GroupMaintainerSchema{
				Type: client.EnumMaintainerTypeGroupGroup,
				Group: client.GroupEntitySchema{
					Id: entityID,
				},
			}

			item := client.IntegrationResourcesCreateBodySchema_Maintainers_Item{}
			err := item.MergeGroupMaintainerSchema(maintainerGroup)
			if err != nil {
				resp.Diagnostics.AddError(
					"Client Error",
					"Failed to merge group maintainer",
				)
				return
			}

			maintainers = append(maintainers, item)
		default:
			resp.Diagnostics.AddError(
				"Client Error",
				"Invalid maintainer type only support user and group",
			)
			return
		}
	}

	var prerequisitePermissions *[][]client.IntegrationResourcesCreateBodySchema_PrerequisitePermissions_Item
	if len(plan.PrerequisitePermissions) > 0 {
		ppData := make([][]client.IntegrationResourcesCreateBodySchema_PrerequisitePermissions_Item, 0, len(plan.PrerequisitePermissions))
		for _, pp := range plan.PrerequisitePermissions {
			if pp.Role.ID.IsNull() || pp.Role.ID.IsUnknown() {
				continue
			}

			item := client.IntegrationResourcesCreateBodySchema_PrerequisitePermissions_Item{}
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

			ppData = append(ppData, []client.IntegrationResourcesCreateBodySchema_PrerequisitePermissions_Item{
				item,
			})
		}
		prerequisitePermissions = &ppData
	}

	var userDefinedTags []string
	udtDiags := plan.UserDefinedTags.ElementsAs(ctx, &userDefinedTags, true)
	if udtDiags.HasError() {
		resp.Diagnostics.Append(udtDiags...)
		return
	}

	body := client.IntegrationResourcesCreateBodySchema{
		Integration:             integration,
		Maintainers:             &maintainers,
		Name:                    name,
		Owner:                   &owner,
		PrerequisitePermissions: prerequisitePermissions,
		Requestable:             plan.Requestable.ValueBool(),
		UserDefinedDescription:  plan.UserDefinedDescription.ValueStringPointer(),
		UserDefinedTags:         &userDefinedTags,
		Workflow:                &workflow,
	}

	if len(allowedDurations) > 0 {
		body.AllowedDurations = &allowedDurations
	}

	resourceResp, err := r.client.ResourcesCreateWithResponse(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiConnection.Error(),
			fmt.Sprintf("Unable to create the resource, got error: %v", err),
		)
		return
	}

	err = utils.HTTPResponseToError(resourceResp.HTTPResponse.StatusCode, resourceResp.Body)
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiResponse.Error(),
			fmt.Sprintf(
				"Failed to create the resource, %s",
				err.Error(),
			),
		)

		return
	}

	plan, diags = convertFullResourceResultResponseSchemaToModel(
		ctx,
		&resourceResp.JSON200.Result,
	)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "created a entitle resource resource")

	// Save data into Terraform state
	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read this function is used to read an existing resource of type Entitle Resource.
//
// It retrieves the resource's data from the provider API requests.
// The retrieved data is then mapped to the ResourceResourceModel,
// and the data is saved to Terraform state.
func (r *ResourceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ResourceResourceModel

	// Read Terraform prior state data into the model
	diags := req.State.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	uid, err := uuid.Parse(data.ID.String())
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Failed to parse the resource id (%s) to UUID, got error: %s", data.ID.String(), err),
		)
		return
	}

	resourceResp, err := r.client.ResourcesShowWithResponse(ctx, uid)
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiConnection.Error(),
			fmt.Sprintf("Unable to get the resource by the id (%s), got error: %s", uid.String(), err),
		)
		return
	}

	err = utils.HTTPResponseToError(resourceResp.HTTPResponse.StatusCode, resourceResp.Body)
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiResponse.Error(),
			fmt.Sprintf(
				"Failed to get the resource by the id (%s), %s",
				uid.String(),
				err.Error(),
			),
		)

		return
	}

	data, diags = convertFullResourceResultResponseSchemaToModel(
		ctx,
		&resourceResp.JSON200.Result,
	)
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

// Update this function handles updates to an existing resource of type Entitle Resource.
//
// It reads the updated Terraform plan data provided in req.Plan and maps it to the ResourceResourceModel.
// And sends a request to the Entitle API to update the resource using API requests.
// If the update is successful, it saves the updated resource data into Terraform state.
func (r *ResourceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ResourceResourceModel

	// Read Terraform plan data into the model
	diags := req.Plan.Get(ctx, &data)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	uid, err := uuid.Parse(data.ID.String())
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Failed to parse the given id to UUID format, got error: %v", err),
		)
		return
	}

	if data.Name.IsNull() || data.Name.IsUnknown() {
		resp.Diagnostics.AddError(
			"Client Error",
			"Missing the name variable for entitle resource",
		)
		return
	}

	allowedDurations, diags := utils.ConvertTerraformSetToAllowedDurations(ctx, data.AllowedDurations)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	var workflow client.IdParamsSchema
	if data.Workflow != nil {
		if !data.Workflow.ID.IsNull() && !data.Workflow.ID.IsUnknown() {
			workflow.Id, err = uuid.Parse(data.Workflow.ID.String())
			if err != nil {
				resp.Diagnostics.AddError(
					"Client Error",
					fmt.Sprintf("Failed to parse given workflow id to UUID, got error: %v", err),
				)
				return
			}
		}
	}

	var owner client.UserEntitySchema
	if data.Owner != nil {
		if data.Owner.Id.ValueString() != "" {
			owner.Id = utils.TrimPrefixSuffix(data.Owner.Id.String())
		} else if data.Owner.Email.ValueString() != "" {
			owner.Id = strings.ToLower(utils.TrimPrefixSuffix(data.Owner.Email.String()))
		} else {
			resp.Diagnostics.AddError(
				"Config Error",
				"Missing the owner's identifier for entitle resource",
			)
			return
		}
	}

	var maintainers []client.IntegrationResourcesUpdateBodySchema_Maintainers_Item
	if len(data.Maintainers) > 0 {
		maintainers = make([]client.IntegrationResourcesUpdateBodySchema_Maintainers_Item, 0, len(data.Maintainers))
		for _, maintainer := range data.Maintainers {
			if maintainer.Type.IsNull() || maintainer.Type.IsUnknown() {
				continue
			}

			idAttr := maintainer.Entity.Attributes()["id"]
			strVal, ok := idAttr.(basetypes.StringValue)
			if !ok {
				resp.Diagnostics.AddError(
					"Client Error",
					"Missing data for entity maintainer id",
				)
				return
			}
			entityID := strVal.ValueString()

			if maintainer.Entity.IsNull() {
				resp.Diagnostics.AddError(
					"Client Error",
					"Missing data for entity maintainer",
				)
				return
			}

			switch maintainer.Type.ValueString() {
			case utils.MaintainerTypeUser:

				maintainerUser := client.UserMaintainerSchema{
					Type: client.EnumMaintainerTypeUserUser,
					User: client.UserEntitySchema{
						Id: entityID,
					},
				}

				item := client.IntegrationResourcesUpdateBodySchema_Maintainers_Item{}
				err := item.MergeUserMaintainerSchema(maintainerUser)
				if err != nil {
					resp.Diagnostics.AddError(
						"Client Error",
						fmt.Sprintf("Failed to merge user maintainer data, error: %v", err),
					)
				}

				maintainers = append(maintainers, item)
			case utils.MaintainerTypeGroup:
				maintainerGroup := client.GroupMaintainerSchema{
					Type: client.EnumMaintainerTypeGroupGroup,
					Group: client.GroupEntitySchema{
						Id: entityID,
					},
				}

				item := client.IntegrationResourcesUpdateBodySchema_Maintainers_Item{}
				err = item.MergeGroupMaintainerSchema(maintainerGroup)
				if err != nil {
					resp.Diagnostics.AddError(
						"Client Error",
						fmt.Sprintf("Failed to merge group maintainer data, error: %v", err),
					)
				}

				maintainers = append(maintainers, item)
			default:
				resp.Diagnostics.AddError(
					"Client Error",
					"Failed invalid maintainer type only support user and group",
				)
				return
			}
		}
	}

	var prerequisitePermissions *[][]client.IntegrationResourcesUpdateBodySchema_PrerequisitePermissions_Item
	if len(data.PrerequisitePermissions) > 0 {
		ppData := make([][]client.IntegrationResourcesUpdateBodySchema_PrerequisitePermissions_Item, 0, len(data.PrerequisitePermissions))
		for _, pp := range data.PrerequisitePermissions {
			if pp.Role.ID.IsNull() || pp.Role.ID.IsUnknown() {
				continue
			}

			item := client.IntegrationResourcesUpdateBodySchema_PrerequisitePermissions_Item{}
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

			ppData = append(ppData, []client.IntegrationResourcesUpdateBodySchema_PrerequisitePermissions_Item{
				item,
			})
		}
		prerequisitePermissions = &ppData
	}

	var userDefinedTags []string
	if !data.UserDefinedTags.IsUnknown() {
		for _, tag := range data.UserDefinedTags.Elements() {
			tagValue, ok := tag.(basetypes.StringValue)
			if !ok {
				continue
			}

			userDefinedTags = append(userDefinedTags, tagValue.ValueString())
		}
	}

	request := client.ResourcesUpdateJSONRequestBody{
		PrerequisitePermissions: prerequisitePermissions,
		Requestable:             data.Requestable.ValueBoolPointer(),
		UserDefinedDescription:  data.UserDefinedDescription.ValueStringPointer(),
	}

	if len(maintainers) > 0 {
		request.Maintainers = &maintainers
	}

	if owner.Id != "" {
		request.Owner = &owner
	}

	if len(allowedDurations) > 0 {
		request.AllowedDurations = &allowedDurations
	}

	if workflow.Id.String() != "" {
		request.Workflow = &workflow
	}

	if len(userDefinedTags) > 0 {
		request.UserDefinedTags = &userDefinedTags
	}

	resourceResp, err := r.client.ResourcesUpdateWithResponse(ctx, uid, request)

	if err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiConnection.Error(),
			fmt.Sprintf("Unable to update the resource by the id (%s), got error: %s", uid.String(), err),
		)
		return
	}

	err = utils.HTTPResponseToError(resourceResp.HTTPResponse.StatusCode, resourceResp.Body)
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiResponse.Error(),
			fmt.Sprintf(
				"Failed to update the resource by the id (%s), %s",
				uid.String(),
				err.Error(),
			),
		)

		return
	}

	data, diags = convertFullResourceResultResponseSchemaToModel(
		ctx,
		&resourceResp.JSON200.Result,
	)
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

// Delete this function is responsible for deleting an existing resource of type
//
// It reads the resource's data from Terraform state, extracts the unique identifier,
// and sends a request to delete the resource using API requests.
// If the deletion is successful, it removes the resource from Terraform state.
func (r *ResourceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ResourceResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	parsedUUID, err := uuid.Parse(data.ID.String())
	if err != nil {
		return
	}

	httpResp, err := r.client.ResourcesDeleteWithResponse(ctx, parsedUUID)
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiConnection.Error(),
			fmt.Sprintf("Unable to delete resource, id: (%s), got error: %v", data.ID.String(), err),
		)
		return
	}

	err = utils.HTTPResponseToError(httpResp.HTTPResponse.StatusCode, httpResp.Body, utils.WithIgnoreNotFound())
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiResponse.Error(),
			fmt.Sprintf(
				"Failed to delete the resource by the id (%s), %s",
				data.ID.String(),
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
func (r *ResourceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// convertFullResourceResultResponseSchemaToModel is a utility function used to convert the API response data
// (of type client.IntegrationResourceResultSchema) to a Terraform resource model (of type ResourceResourceModel).
//
// It extracts and transforms data from the API response into a format that can be stored in Terraform state.
// It returns the converted model and any diagnostic information if there are errors during the conversion.
func convertFullResourceResultResponseSchemaToModel(
	ctx context.Context,
	data *client.IntegrationResourceResultSchema,
) (ResourceResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	// Check if the API response data is nil
	if data == nil {
		diags.AddError(
			"No data",
			"Failed: the given schema data is nil",
		)

		return ResourceResourceModel{}, diags
	}

	// Extract and convert allowed durations from the API response
	allowedDurationsValues := make([]attr.Value, len(data.AllowedDurations))
	allowedDurations := types.SetNull(types.NumberType)

	if data.AllowedDurations != nil {
		for i, duration := range data.AllowedDurations {
			allowedDurationsValues[i] = types.NumberValue(big.NewFloat(float64(duration)))
		}

		var sDiag diag.Diagnostics

		allowedDurations, sDiag = types.SetValue(types.NumberType, allowedDurationsValues)
		if sDiag.HasError() {
			return ResourceResourceModel{}, sDiag
		}
	}

	if diags.HasError() {
		return ResourceResourceModel{}, diags
	}

	maintainers := make([]*utils.MaintainerModel, 0, len(data.Maintainers))
	for _, item := range data.Maintainers {
		var body utils.MaintainerCommonResponseSchema

		dataBytes, err := item.MarshalJSON()
		if err != nil {
			diags.AddError(
				"No data",
				fmt.Sprintf("Failed to marshal the maintainer data, %s",
					err.Error()),
			)

			return ResourceResourceModel{}, diags
		}

		err = json.Unmarshal(dataBytes, &body)
		if err != nil {
			diags.AddError(
				"No data",
				fmt.Sprintf("Failed to unmarshal the maintainer data, %s",
					err.Error()),
			)

			return ResourceResourceModel{}, diags
		}

		switch strings.ToLower(body.Type) {
		case utils.MaintainerTypeUser:
			responseSchema, err := item.AsMaintainerUserResponseSchema()
			if err != nil {
				diags.AddError(
					"No data",
					fmt.Sprintf(
						"Failed to convert response schema to user response schema, error: %v",
						err,
					),
				)

				return ResourceResourceModel{}, diags
			}

			u := &utils.IdEmailModel{
				Id:    utils.TrimmedStringValue(responseSchema.User.Id.String()),
				Email: utils.TrimmedStringValue(string(responseSchema.User.Email)),
			}

			uObject, diagsValues := u.AsObjectValue(ctx)
			if diagsValues.HasError() {
				diags.Append(diagsValues...)
				return ResourceResourceModel{}, diags
			}

			maintainerUser := &utils.MaintainerModel{
				Type:   utils.TrimmedStringValue(body.Type),
				Entity: uObject,
			}

			maintainers = append(maintainers, maintainerUser)
		case utils.MaintainerTypeGroup:
			responseSchema, err := item.AsMaintainerGroupResponseSchema()
			if err != nil {
				diags.AddError(
					"No data",
					fmt.Sprintf(
						"Failed to convert response schema to group response schema, error: %v",
						err,
					),
				)

				return ResourceResourceModel{}, diags
			}

			g := &utils.IdEmailModel{
				Id:    utils.TrimmedStringValue(responseSchema.Group.Id.String()),
				Email: utils.TrimmedStringValue(string(responseSchema.Group.Email)),
			}

			gObject, diagsValues := g.AsObjectValue(ctx)
			if diagsValues.HasError() {
				diags.Append(diagsValues...)
				return ResourceResourceModel{}, diags
			}

			maintainerGroup := &utils.MaintainerModel{
				Type:   utils.TrimmedStringValue(body.Type),
				Entity: gObject,
			}

			maintainers = append(maintainers, maintainerGroup)
		default:
			diags.AddError("Failed invalid type for maintainer", body.Type)
			return ResourceResourceModel{}, diags
		}
	}

	var prerequisitePermissions []utils.PrerequisitePermissionModel
	if data.PrerequisitePermissions != nil {
		for _, pp := range *data.PrerequisitePermissions {
			for _, item := range pp {
				v, err := item.AsPrerequisiteRolePermissionResponseSchema()
				if err != nil {
					diags.AddError(
						"No data",
						fmt.Sprintf(
							"Failed to unmarshal the prerequisite permissions data, %s",
							err.Error(),
						),
					)
					return ResourceResourceModel{}, diags
				}

				roleModel, diagsGetRoles := utils.GetRole(ctx, v.Role.Id.String(), v.Role.Name, v.Role.Resource)
				if diagsGetRoles.HasError() {
					diags.Append(diagsGetRoles...)
					return ResourceResourceModel{}, diags
				}

				prerequisitePermissions = append(prerequisitePermissions,
					utils.PrerequisitePermissionModel{
						Default: types.BoolValue(v.Default),
						Role:    roleModel,
					},
				)
			}
		}
	}

	userDefinedTags := types.SetNull(types.StringType)
	if data.UserDefinedTags != nil {
		tagVals := make([]attr.Value, len(*data.UserDefinedTags))
		for i, v := range *data.UserDefinedTags {
			tagVals[i] = types.StringValue(v)
		}

		// Create a typed set value
		setVal, tDiags := types.SetValue(types.StringType, tagVals)
		if tDiags.HasError() {
			diags.Append(tDiags...)
			return ResourceResourceModel{}, diags
		}

		userDefinedTags = setVal
	}

	tags := types.SetNull(types.StringType)
	if data.Tags != nil {
		tagVals := make([]attr.Value, len(*data.Tags))
		for i, v := range *data.Tags {
			tagVals[i] = types.StringValue(v)
		}

		// Create a typed set value
		setVal, tDiags := types.SetValue(types.StringType, tagVals)
		if tDiags.HasError() {
			diags.Append(tDiags...)
			return ResourceResourceModel{}, diags
		}

		tags = setVal
	}

	var owner *utils.IdEmailModel
	if data.Owner != nil {
		marshalJSON, err := data.Owner.Email.MarshalJSON()
		if err != nil {
			return ResourceResourceModel{}, nil
		}
		owner = &utils.IdEmailModel{
			Id:    utils.TrimmedStringValue(data.Owner.Id.String()),
			Email: utils.TrimmedStringValue(strings.ToLower(string(marshalJSON))),
		}
	}

	var workflow *utils.IdNameModel
	if data.Workflow.Id.String() != "" {
		workflow = &utils.IdNameModel{
			ID:   utils.TrimmedStringValue(data.Workflow.Id.String()),
			Name: utils.TrimmedStringValue(data.Workflow.Name),
		}
	}

	model := ResourceResourceModel{
		ID:                     utils.TrimmedStringValue(data.Id.String()),
		Name:                   utils.TrimmedStringValue(data.Name),
		AllowedDurations:       allowedDurations,
		Maintainers:            maintainers,
		Tags:                   tags,
		UserDefinedTags:        userDefinedTags,
		UserDefinedDescription: types.StringPointerValue(data.UserDefinedDescription),
		Workflow:               workflow,
		Integration: utils.IdNameModel{
			ID:   utils.TrimmedStringValue(data.Integration.Id.String()),
			Name: utils.TrimmedStringValue(data.Integration.Name),
		},
		Requestable:             types.BoolValue(data.Requestable),
		Owner:                   owner,
		PrerequisitePermissions: prerequisitePermissions,
	}

	// Create the Terraform resource model using the extracted data
	return model, diags
}
