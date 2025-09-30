package integrations

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
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

	"github.com/entitleio/terraform-provider-entitle/internal/client"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/utils"
	"github.com/entitleio/terraform-provider-entitle/internal/validators"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &IntegrationResource{}
var _ resource.ResourceWithImportState = &IntegrationResource{}

func NewIntegrationResource() resource.Resource {
	return &IntegrationResource{}
}

// IntegrationResource defines the resource implementation.
type IntegrationResource struct {
	client *client.ClientWithResponses
}

// IntegrationResourceModel describes the resource data model.
type IntegrationResourceModel struct {
	ID                                   types.String                        `tfsdk:"id"`
	Name                                 types.String                        `tfsdk:"name"`
	AllowedDurations                     types.Set                           `tfsdk:"allowed_durations"`
	AllowChangingAccountPermissions      types.Bool                          `tfsdk:"allow_changing_account_permissions"`
	AllowCreatingAccounts                types.Bool                          `tfsdk:"allow_creating_accounts"`
	Readonly                             types.Bool                          `tfsdk:"readonly"`
	AllowRequests                        types.Bool                          `tfsdk:"allow_requests"`
	AllowRequestsByDefault               types.Bool                          `tfsdk:"allow_requests_by_default"`
	Requestable                          types.Bool                          `tfsdk:"requestable"`
	RequestableByDefault                 types.Bool                          `tfsdk:"requestable_by_default"`
	AutoAssignRecommendedMaintainers     types.Bool                          `tfsdk:"auto_assign_recommended_maintainers"`
	AutoAssignRecommendedOwners          types.Bool                          `tfsdk:"auto_assign_recommended_owners"`
	NotifyAboutExternalPermissionChanges types.Bool                          `tfsdk:"notify_about_external_permission_changes"`
	Owner                                utils.IdEmailModel                  `tfsdk:"owner"`
	Application                          utils.NameModel                     `tfsdk:"application"`
	AgentToken                           *utils.NameModel                    `tfsdk:"agent_token"`
	Workflow                             utils.IdNameModel                   `tfsdk:"workflow"`
	Maintainers                          []*utils.MaintainerModel            `tfsdk:"maintainers"`
	ConnectionJson                       types.String                        `tfsdk:"connection_json"`
	PrerequisitePermissions              []utils.PrerequisitePermissionModel `tfsdk:"prerequisite_permissions"`
}

func (r *IntegrationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration"
}

func (r *IntegrationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "A specific instance or integration with an \"Application\". Integration includes the " +
			"configuration needed to connect Entitle including credentials, as well as all the users permissions information. [Read more about integrations](https://docs.beyondtrust.com/entitle/docs/integrations-resources-roles).",
		Description: "A specific instance or integration with an \"Application\". Integration includes the " +
			"configuration needed to connect Entitle including credentials, as well as all the users permissions information. [Read more about integrations](https://docs.beyondtrust.com/entitle/docs/integrations-resources-roles).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Entitle Integration identifier in uuid format",
				Description:         "Entitle Integration identifier in uuid format",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "The display name for the integration. Length between 2 and 50.",
				Description:         "The display name for the integration. Length between 2 and 50.",
				Validators: []validator.String{
					validators.NewName(2, 50),
				},
			},
			"allowed_durations": schema.SetAttribute{
				Required:    true,
				ElementType: types.NumberType,
				Description: "As the admin, you can set different durations for the integration, " +
					"compared to the workflow linked to it.",
				MarkdownDescription: "As the admin, you can set different durations for the integration, " +
					"compared to the workflow linked to it.",
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
						"user": schema.SingleNestedAttribute{
							Attributes: map[string]schema.Attribute{
								"id": schema.ListAttribute{
									ElementType:         types.StringType,
									Required:            true,
									Description:         "user's id",
									MarkdownDescription: "user's id",
								},
								"email": schema.ListAttribute{
									ElementType:         types.StringType,
									Computed:            true,
									Description:         "user's email",
									MarkdownDescription: "user's email",
								},
							},
							Optional:            true,
							Description:         "user",
							MarkdownDescription: "user",
						},
						"group": schema.SingleNestedAttribute{
							Attributes: map[string]schema.Attribute{
								"id": schema.ListAttribute{
									ElementType:         types.StringType,
									Required:            true,
									Description:         "group's id",
									MarkdownDescription: "group's id",
								},
								"email": schema.ListAttribute{
									ElementType:         types.StringType,
									Computed:            true,
									Description:         "group's email",
									MarkdownDescription: "group's email",
								},
							},
							Optional:            true,
							Description:         "group",
							MarkdownDescription: "group",
						},
					},
				},
				Required: true,
				Description: "Maintainer of the integration, second tier owner of that integration you can " +
					"have multiple integration Maintainer also can be IDP group. In the case of the bundle the Maintainer of each Integration.",
				MarkdownDescription: "Maintainer of the integration, second tier owner of that integration you can " +
					"have multiple integration Maintainer also can be IDP group. In the case of the bundle the Maintainer of each Integration.",
			},
			"application": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						Required:            true,
						Description:         "The application's name",
						MarkdownDescription: "The application's name",
						Validators: []validator.String{
							validators.NewName(2, 50),
							validators.Lowercase{},
						},
					},
				},
				Required: true,
				Description: "The application the integration connects to must be chosen from the list " +
					"of supported applications.",
				MarkdownDescription: "The application the integration connects to must be chosen from the list " +
					"of supported applications.",
			},
			"agent_token": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						Required:            true,
						Description:         "agent token's name",
						MarkdownDescription: "agent token's name",
					},
				},
				Optional:            true,
				Description:         "Agent token configuration. Used for agent-based integrations where Entitle needs a token to authenticate.",
				MarkdownDescription: "Agent token configuration. Used for agent-based integrations where Entitle needs a token to authenticate.n",
			},
			"allow_changing_account_permissions": schema.BoolAttribute{
				Required:            false,
				Optional:            true,
				Computed:            true,
				Description:         "Controls whether Entitle can modify the permissions of accounts under this integration. If disabled, Entitle can only read permissions but cannot grant or revoke them. (default: true)",
				MarkdownDescription: "Controls whether Entitle can modify the permissions of accounts under this integration. If disabled, Entitle can only read permissions but cannot grant or revoke them. (default: true)",
				Default:             booldefault.StaticBool(defaultIntegrationAllowChangingAccountPermissions),
			},
			"allow_creating_accounts": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "Controls whether Entitle is allowed to create new user accounts in the connected application when access is requested. If disabled, users must already exist in the application before access can be granted. (default: true)",
				MarkdownDescription: "Controls whether Entitle is allowed to create new user accounts in the connected application when access is requested. If disabled, users must already exist in the application before access can be granted. (default: true)",
				Default:             booldefault.StaticBool(defaultIntegrationAllowCreatingAccounts),
			},
			"readonly": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Description: "If turned on, any request opened by a user will not be automatically granted, " +
					"instead a ticket will be opened for manual resolution. (default: false)",
				MarkdownDescription: "If turned on, any request opened by a user will not be automatically granted, " +
					"instead a ticket will be opened for manual resolution. (default: false)",
				Default: booldefault.StaticBool(defaultIntegrationReadonly),
			},
			"allow_requests": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Description: "Controls whether a user can create requests for entitlements for resources " +
					"under the integration. (default: true)",
				MarkdownDescription: "Controls whether a user can create requests for entitlements for resources " +
					"under the integration. (default: true)",
				Default: booldefault.StaticBool(defaultIntegrationAllowRequests),
			},
			"allow_requests_by_default": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Description: "Controls whether resources that are added to the integration could be shown " +
					"to the user. (default: true)",
				MarkdownDescription: "Controls whether resources that are added to the integration could be shown " +
					"to the user. (default: true)",
				Default: booldefault.StaticBool(defaultIntegrationAllowRequestsByDefault),
			},
			"requestable": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Description: "Controls whether a user can create requests for entitlements for resources " +
					"under the integration. (default: true)",
				MarkdownDescription: "Controls whether a user can create requests for entitlements for resources " +
					"under the integration. (default: true)",
				Default: booldefault.StaticBool(defaultIntegrationAllowRequests),
			},
			"requestable_by_default": schema.BoolAttribute{
				Optional: true,
				Computed: true,
				Description: "Controls whether resources that are added to the integration could be shown " +
					"to the user. (default: true)",
				MarkdownDescription: "Controls whether resources that are added to the integration could be shown " +
					"to the user. (default: true)",
				Default: booldefault.StaticBool(defaultIntegrationAllowRequestsByDefault),
			},
			"auto_assign_recommended_maintainers": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "When enabled, Entitle automatically assigns suggested maintainers to the integration based on usage patterns and access signals. (default: true)",
				MarkdownDescription: "When enabled, Entitle automatically assigns suggested maintainers to the integration based on usage patterns and access signals. (default: true)",
				Default:             booldefault.StaticBool(defaultIntegrationAutoAssignRecommendedMaintainers),
			},
			"auto_assign_recommended_owners": schema.BoolAttribute{
				Required:            false,
				Optional:            true,
				Computed:            true,
				Description:         "When enabled, Entitle automatically assigns suggested owners to the integration based on ownership signals, such as group ownership or historical access. (default: true)",
				MarkdownDescription: "When enabled, Entitle automatically assigns suggested owners to the integration based on ownership signals, such as group ownership or historical access. (default: true)",
				Default:             booldefault.StaticBool(defaultIntegrationAutoAssignRecommendedOwners),
			},
			"notify_about_external_permission_changes": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				Description:         "When enabled, Entitle will notify owners if permissions are changed directly in the connected application, bypassing Entitle. (default: true)",
				MarkdownDescription: "When enabled, Entitle will notify owners if permissions are changed directly in the connected application, bypassing Entitle. (default: true)",
				Default:             booldefault.StaticBool(defaultIntegrationNotifyAboutExternalPermissionChanges),
			},
			"connection_json": schema.StringAttribute{
				Required:            true,
				Description:         "go to https://app.entitle.io/integrations and provide the latest schema.",
				MarkdownDescription: "go to https://app.entitle.io/integrations and provide the latest schema.",
			},
			"prerequisite_permissions": schema.ListNestedAttribute{
				Optional:            true,
				Description:         "Users granted any role from this integration through a request will automatically receive the permissions to the roles selected below.",
				MarkdownDescription: "Users granted any role from this integration through a request will automatically receive the permissions to the roles selected below.",
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
			"owner": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Required:            true,
						Description:         "the owner's id",
						MarkdownDescription: "the owner's id",
					},
					"email": schema.StringAttribute{
						Computed:            true,
						Description:         "the owner's email",
						MarkdownDescription: "the owner's email",
					},
				},
				Required: true,
				Description: "Define the owner of the integration, which will be used for administrative " +
					"purposes and approval workflows.",
				MarkdownDescription: "Define the owner of the integration, which will be used for administrative " +
					"purposes and approval workflows.",
			},
			"workflow": schema.SingleNestedAttribute{
				Attributes: map[string]schema.Attribute{
					"id": schema.StringAttribute{
						Required:            true,
						Description:         "the workflow's id",
						MarkdownDescription: "the workflow's id",
						Validators: []validator.String{
							validators.UUID{},
						},
					},
					"name": schema.StringAttribute{
						Computed:            true,
						Description:         "the workflow's name",
						MarkdownDescription: "the workflow's name",
					},
				},
				Required: true,
				Description: "The default approval workflow for entitlements for the integration " +
					"(can be overwritten on resource/role level).",
				MarkdownDescription: "The default approval workflow for entitlements for the integration " +
					"(can be overwritten on resource/role level).",
			},
		},
	}
}

func (r *IntegrationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Create this function is responsible for creating a new resource of type Entitle Integration.
//
// Its reads the Terraform plan data provided in req.Plan and maps it to the IntegrationResourceModel.
// And sends a request to the Entitle API to create the resource using API requests.
// If the creation is successful, it saves the resource's data into Terraform state.
func (r *IntegrationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var err error
	var plan IntegrationResourceModel

	// Read Terraform plan data into the model
	diags := req.Plan.Get(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	var allowedDurations *[]client.EnumAllowedDurations
	aDurations, diags := utils.GetEnumAllowedDurationsSliceFromNumberSet(ctx, plan.AllowedDurations)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	if aDurations != nil {
		allowedDurations = &aDurations
	}

	var workflow client.IdParamsSchema
	workflow.Id = uuid.MustParse(plan.Workflow.ID.ValueString())

	var agentToken *client.NameSchema
	if plan.AgentToken != nil {
		agentToken = &client.NameSchema{
			Name: plan.AgentToken.Name.ValueString(),
		}
	}

	owner := client.UserEntitySchema{
		Id: utils.TrimPrefixSuffix(plan.Owner.Id.ValueString()),
	}

	application := client.NameSchema{
		Name: utils.TrimPrefixSuffix(plan.Application.Name.ValueString()),
	}

	var connectionJson *map[string]interface{}
	if plan.ConnectionJson.IsNull() || plan.ConnectionJson.IsUnknown() || plan.ConnectionJson.ValueString() == "" {
		resp.Diagnostics.AddError(
			"Client Error",
			"missing the connection_json variable for entitle integration",
		)
		return
	}

	var data map[string]interface{}

	// Use json.Unmarshal to convert the JSON string into the map
	if err := json.Unmarshal(
		[]byte(plan.ConnectionJson.ValueString()),
		&data,
	); err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf(
				"failed to parse given connection json to json, %s, error: %v",
				plan.ConnectionJson.ValueString(),
				err,
			),
		)
		return
	}

	connectionJson = &data

	maintainers := make([]client.IntegrationCreateBodySchema_Maintainers_Item, 0)
	for _, maintainer := range plan.Maintainers {
		if maintainer.Type.IsNull() || maintainer.Type.IsUnknown() {
			continue
		}

		if maintainer.Entity.IsNull() {
			resp.Diagnostics.AddError(
				"Client Error",
				"failed missing data for entity maintainer",
			)
			return
		}

		var target client.EntityResponseSchema
		diagsAs := maintainer.Entity.As(ctx, &target, basetypes.ObjectAsOptions{
			UnhandledUnknownAsEmpty: true,
		})
		if diagsAs.HasError() {
			diags.Append(diagsAs...)
		}

		switch maintainer.Type.String() {
		case utils.MaintainerTypeUser:
			maintainerUser := client.UserMaintainerSchema{
				Type: client.EnumMaintainerTypeUserUser,
				User: client.UserEntitySchema{
					Id: target.Id.String(),
				},
			}

			item := client.IntegrationCreateBodySchema_Maintainers_Item{}
			err := item.MergeUserMaintainerSchema(maintainerUser)
			if err != nil {
				resp.Diagnostics.AddError(
					"Client Error",
					fmt.Sprintf("failed to merge user maintainer data, error: %v", err),
				)
			}

			maintainers = append(maintainers, item)
		case utils.MaintainerTypeGroup:
			maintainerGroup := client.GroupMaintainerSchema{
				Type: client.EnumMaintainerTypeGroupGroup,
				Group: client.GroupEntitySchema{
					Id: target.Id.String(),
				},
			}

			item := client.IntegrationCreateBodySchema_Maintainers_Item{}
			err := item.MergeGroupMaintainerSchema(maintainerGroup)
			if err != nil {
				resp.Diagnostics.AddError(
					"Client Error",
					"failed to merge group maintainer",
				)
				return
			}

			maintainers = append(maintainers, item)
		default:
			resp.Diagnostics.AddError(
				"Client Error",
				"failed invalid maintainer type only support user and group",
			)
			return
		}
	}

	var prerequisitePermissions *[][]client.IntegrationCreateBodySchema_PrerequisitePermissions_Item
	if len(plan.PrerequisitePermissions) > 0 {
		ppData := make([][]client.IntegrationCreateBodySchema_PrerequisitePermissions_Item, 0, len(plan.PrerequisitePermissions))
		for _, pp := range plan.PrerequisitePermissions {
			if pp.Role.ID.IsNull() || pp.Role.ID.IsUnknown() {
				continue
			}

			item := client.IntegrationCreateBodySchema_PrerequisitePermissions_Item{}
			err := item.MergePrerequisitePermissionCreateBodySchema(client.PrerequisitePermissionCreateBodySchema{
				Default: pp.Default.ValueBool(),
				Role: map[string]interface{}{
					"id": pp.Role.ID.ValueString(),
				},
			})
			if err != nil {
				resp.Diagnostics.AddError(
					"Client Error",
					fmt.Sprintf("failed to merge prerequisite permission data, error: %v", err),
				)
				return
			}

			ppData = append(ppData, []client.IntegrationCreateBodySchema_PrerequisitePermissions_Item{
				item,
			})
		}
		prerequisitePermissions = &ppData
	}

	body := client.IntegrationCreateBodySchema{
		AgentToken:                           agentToken,
		AllowChangingAccountPermissions:      plan.AllowChangingAccountPermissions.ValueBool(),
		AllowCreatingAccounts:                plan.AllowCreatingAccounts.ValueBool(),
		AllowRequests:                        plan.AllowRequests.ValueBoolPointer(),
		AllowRequestsByDefault:               plan.AllowRequestsByDefault.ValueBoolPointer(),
		Requestable:                          plan.Requestable.ValueBoolPointer(),
		RequestableByDefault:                 plan.RequestableByDefault.ValueBoolPointer(),
		AllowedDurations:                     allowedDurations,
		Application:                          application,
		AutoAssignRecommendedMaintainers:     plan.AutoAssignRecommendedMaintainers.ValueBool(),
		AutoAssignRecommendedOwners:          plan.AutoAssignRecommendedOwners.ValueBool(),
		ConnectionJson:                       connectionJson,
		Maintainers:                          &maintainers,
		Name:                                 plan.Name.ValueString(),
		NotifyAboutExternalPermissionChanges: plan.NotifyAboutExternalPermissionChanges.ValueBool(),
		Owner:                                owner,
		Readonly:                             plan.Readonly.ValueBool(),
		Workflow:                             workflow,
		PrerequisitePermissions:              prerequisitePermissions,
	}

	integrationResp, err := r.client.IntegrationsCreateWithResponse(ctx, body)
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiConnection.Error(),
			fmt.Sprintf("Unable to create the integration, got error: %v", err),
		)
		return
	}

	err = utils.HTTPResponseToError(integrationResp.HTTPResponse.StatusCode, integrationResp.Body)
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiResponse.Error(),
			fmt.Sprintf(
				"Failed to create the Integration, %s, status code: %d, %s",
				string(integrationResp.Body),
				integrationResp.HTTPResponse.StatusCode,
				err.Error(),
			))
		return
	}

	agentTokenName := ""
	if agentToken != nil {
		agentTokenName = agentToken.Name
	}

	plan, diags = convertFullIntegrationResultResponseSchemaToModel(
		ctx,
		&integrationResp.JSON200.Result,
		plan.ConnectionJson.ValueString(),
		agentTokenName,
	)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Write logs using the tflog package
	// Documentation: https://terraform.io/plugin/log
	tflog.Trace(ctx, "created a entitle integration resource")

	// Save data into Terraform state
	diags = resp.State.Set(ctx, &plan)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
}

// Read this function is used to read an existing resource of type Entitle Integration.
//
// It retrieves the resource's data from the provider API requests.
// The retrieved data is then mapped to the IntegrationResourceModel,
// and the data is saved to Terraform state.
func (r *IntegrationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data IntegrationResourceModel

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
			fmt.Sprintf("failed to parse the resource id (%s) to UUID, got error: %s", data.ID.String(), err),
		)
		return
	}

	integrationResp, err := r.client.IntegrationsShowWithResponse(ctx, uid)
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiConnection.Error(),
			fmt.Sprintf("Unable to get the integration by the id (%s), got error: %s", uid.String(), err),
		)
		return
	}

	err = utils.HTTPResponseToError(integrationResp.HTTPResponse.StatusCode, integrationResp.Body)
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiResponse.Error(),
			fmt.Sprintf(
				"Failed to get the Integration by the id (%s), status code: %d, %s",
				uid.String(),
				integrationResp.HTTPResponse.StatusCode,
				err.Error(),
			),
		)
		return
	}

	agentTokenName := ""
	if data.AgentToken != nil {
		agentTokenName = data.AgentToken.Name.ValueString()
	}

	data, diags = convertFullIntegrationResultResponseSchemaToModel(
		ctx,
		&integrationResp.JSON200.Result,
		data.ConnectionJson.ValueString(),
		agentTokenName,
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

// Update this function handles updates to an existing resource of type Entitle Integration.
//
// It reads the updated Terraform plan data provided in req.Plan and maps it to the IntegrationResourceModel.
// And sends a request to the Entitle API to update the resource using API requests.
// If the update is successful, it saves the updated resource data into Terraform state.
func (r *IntegrationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data IntegrationResourceModel

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
			fmt.Sprintf("falied to parse the given id to UUID format, got error: %v", err),
		)
		return
	}

	name := data.Name.ValueString()

	var allowedDurations *[]client.EnumAllowedDurations
	aDurations, diags := utils.GetEnumAllowedDurationsSliceFromNumberSet(ctx, data.AllowedDurations)
	if diags.HasError() {
		resp.Diagnostics.Append(diags...)
		return
	}

	if aDurations != nil {
		allowedDurations = &aDurations
	}

	var workflow client.IdParamsSchema
	if data.Workflow.ID.ValueString() != "" {
		workflow.Id, err = uuid.Parse(data.Workflow.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Client Error",
				fmt.Sprintf("failed to parse given workflow id to UUID, got error: %v", err),
			)
			return
		}
	}

	var connectionJson *map[string]interface{}
	if data.ConnectionJson.ValueString() != "" {
		var result map[string]interface{}

		// Use json.Unmarshal to convert the JSON string into the map
		if err := json.Unmarshal(
			[]byte(data.ConnectionJson.ValueString()),
			&result,
		); err != nil {
			resp.Diagnostics.AddError(
				"Client Error",
				fmt.Sprintf(
					"failed to parse given connection json to json, %s, error: %v",
					utils.TrimPrefixSuffix(data.ConnectionJson.String()),
					err,
				),
			)
			return
		}

		connectionJson = &result
	}

	var maintainers []client.IntegrationsUpdateBodySchema_Maintainers_Item
	if len(data.Maintainers) > 0 {
		maintainers = make([]client.IntegrationsUpdateBodySchema_Maintainers_Item, 0)
		for _, maintainer := range data.Maintainers {
			if maintainer.Type.IsNull() || maintainer.Type.IsUnknown() {
				continue
			}

			if maintainer.Entity.IsNull() {
				resp.Diagnostics.AddError(
					"Client Error",
					"failed missing data for entity maintainer",
				)
				return
			}

			var target client.EntityResponseSchema
			diagsAs := maintainer.Entity.As(ctx, &target, basetypes.ObjectAsOptions{
				UnhandledUnknownAsEmpty: true,
			})
			if diagsAs.HasError() {
				diags.Append(diagsAs...)
			}

			switch maintainer.Type.String() {
			case utils.MaintainerTypeUser:

				maintainerUser := client.UserMaintainerSchema{
					Type: client.EnumMaintainerTypeUserUser,
					User: client.UserEntitySchema{
						Id: target.Id.String(),
					},
				}

				item := client.IntegrationsUpdateBodySchema_Maintainers_Item{}
				err := item.MergeUserMaintainerSchema(maintainerUser)
				if err != nil {
					resp.Diagnostics.AddError(
						"Client Error",
						fmt.Sprintf("failed to merge user maintainer data, error: %v", err),
					)
				}

				maintainers = append(maintainers, item)
			case utils.MaintainerTypeGroup:
				maintainerGroup := client.GroupMaintainerSchema{
					Type: client.EnumMaintainerTypeGroupGroup,
					Group: client.GroupEntitySchema{
						Id: target.Id.String(),
					},
				}

				item := client.IntegrationsUpdateBodySchema_Maintainers_Item{}
				err = item.MergeGroupMaintainerSchema(maintainerGroup)
				if err != nil {
					resp.Diagnostics.AddError(
						"Client Error",
						fmt.Sprintf("failed to merge group maintainer data, error: %v", err),
					)
				}

				maintainers = append(maintainers, item)
			default:
				resp.Diagnostics.AddError(
					"Client Error",
					"failed invalid maintainer type only support user and group",
				)
				return
			}
		}
	}

	var prerequisitePermissions *[][]client.IntegrationsUpdateBodySchema_PrerequisitePermissions_Item
	if len(data.PrerequisitePermissions) > 0 {
		ppData := make([][]client.IntegrationsUpdateBodySchema_PrerequisitePermissions_Item, 0, len(data.PrerequisitePermissions))
		for _, pp := range data.PrerequisitePermissions {
			if pp.Role.ID.IsNull() || pp.Role.ID.IsUnknown() {
				continue
			}

			item := client.IntegrationsUpdateBodySchema_PrerequisitePermissions_Item{}
			err := item.MergePrerequisitePermissionCreateBodySchema(client.PrerequisitePermissionCreateBodySchema{
				Default: pp.Default.ValueBool(),
				Role: map[string]interface{}{
					"id": pp.Role.ID.ValueString(),
				},
			})
			if err != nil {
				resp.Diagnostics.AddError(
					"Client Error",
					fmt.Sprintf("failed to merge preqrequisite permission data, error: %v", err),
				)
			}

			ppData = append(ppData, []client.IntegrationsUpdateBodySchema_PrerequisitePermissions_Item{
				item,
			})
		}
		prerequisitePermissions = &ppData
	}

	integrationResp, err := r.client.IntegrationsUpdateWithResponse(ctx, uid, client.IntegrationsUpdateBodySchema{
		AllowRequests:                        utils.BoolPointer(data.AllowRequests.ValueBool()),
		AllowRequestsByDefault:               utils.BoolPointer(data.AllowRequestsByDefault.ValueBool()),
		AllowedDurations:                     allowedDurations,
		AutoAssignRecommendedMaintainers:     utils.BoolPointer(data.AutoAssignRecommendedMaintainers.ValueBool()),
		AutoAssignRecommendedOwners:          utils.BoolPointer(data.AutoAssignRecommendedOwners.ValueBool()),
		ConnectionJson:                       connectionJson,
		Maintainers:                          &maintainers,
		Name:                                 utils.StringPointer(name),
		NotifyAboutExternalPermissionChanges: utils.BoolPointer(data.NotifyAboutExternalPermissionChanges.ValueBool()),
		Owner:                                &client.UserEntitySchema{Id: data.Owner.Id.ValueString()},
		Workflow:                             &workflow,
		PrerequisitePermissions:              prerequisitePermissions,
	})
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiConnection.Error(),
			fmt.Sprintf("Unable to update the integration by the id (%s), got error: %s", uid.String(), err),
		)
		return
	}

	err = utils.HTTPResponseToError(integrationResp.HTTPResponse.StatusCode, integrationResp.Body)
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiResponse.Error(),
			fmt.Sprintf(
				"Failed to update the Integration by the id (%s), status code: %d, %s",
				uid.String(),
				integrationResp.HTTPResponse.StatusCode,
				err.Error(),
			),
		)
		return
	}

	agentTokenName := ""
	if data.AgentToken != nil {
		agentTokenName = data.AgentToken.Name.ValueString()
	}

	data, diags = convertFullIntegrationResultResponseSchemaToModel(
		ctx,
		&integrationResp.JSON200.Result,
		data.ConnectionJson.ValueString(),
		agentTokenName,
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
func (r *IntegrationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data IntegrationResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	parsedUUID, err := uuid.Parse(data.ID.String())
	if err != nil {
		return
	}

	httpResp, err := r.client.IntegrationsDestroyWithResponse(ctx, parsedUUID)
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiConnection.Error(),
			fmt.Sprintf("Unable to delete integrations, id: (%s), got error: %v", data.ID.String(), err),
		)
		return
	}

	err = utils.HTTPResponseToError(httpResp.HTTPResponse.StatusCode, httpResp.Body, utils.WithIgnoreNotFound())
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiResponse.Error(),
			fmt.Sprintf(
				"Failed to delete the Integration by the id (%s), status code: %d, %s",
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
func (r *IntegrationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// convertFullIntegrationResultResponseSchemaToModel is a utility function used to convert the API response data
// (of type client.IntegrationResultSchema) to a Terraform resource model (of type IntegrationResourceModel).
//
// It extracts and transforms data from the API response into a format that can be stored in Terraform state.
// It returns the converted model and any diagnostic information if there are errors during the conversion.
func convertFullIntegrationResultResponseSchemaToModel(
	ctx context.Context,
	data *client.IntegrationResultSchema,
	connectionJson, agentTokenName string,
) (IntegrationResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	// Check if the API response data is nil
	if data == nil {
		diags.AddError(
			"No data",
			"Failed: the given schema data is nil",
		)

		return IntegrationResourceModel{}, diags
	}

	// Extract and convert allowed durations from the API response
	allowedDurationsValues, advDiags := utils.GetNumberSetFromAllowedDurations(data.AllowedDurations)
	if advDiags.HasError() {
		diags.Append(advDiags...)
		return IntegrationResourceModel{}, diags
	}

	marshalJSON, err := data.Owner.Email.MarshalJSON()
	if err != nil {
		return IntegrationResourceModel{}, nil
	}

	maintainers := make([]*utils.MaintainerModel, 0, len(data.Maintainers))
	for _, item := range data.Maintainers {
		var body utils.MaintainerCommonResponseSchema

		dataBytes, err := item.MarshalJSON()
		if err != nil {
			diags.AddError(
				"No data",
				"failed to marshal the maintainer data",
			)

			return IntegrationResourceModel{}, diags
		}

		err = json.Unmarshal(dataBytes, &body)
		if err != nil {
			diags.AddError(
				"No data",
				"failed to unmarshal the maintainer data",
			)

			return IntegrationResourceModel{}, diags
		}

		switch strings.ToLower(body.Type) {
		case utils.MaintainerTypeUser:
			responseSchema, err := item.AsMaintainerUserResponseSchema()
			if err != nil {
				diags.AddError(
					"No data",
					fmt.Sprintf("failed to convert response schema to user response schema, error: %v", err),
				)

				return IntegrationResourceModel{}, diags
			}

			bytes, err := responseSchema.User.Email.MarshalJSON()
			if err != nil {
				diags.AddError(
					"No data",
					fmt.Sprintf("failed to get maintainer user email bytes, error: %v", err),
				)

				return IntegrationResourceModel{}, diags
			}

			u := &utils.IdEmailModel{
				Id:    utils.TrimmedStringValue(responseSchema.User.Id.String()),
				Email: utils.TrimmedStringValue(string(bytes)),
			}

			uObject, diagsValues := u.AsObjectValue(ctx)
			if diagsValues.HasError() {
				diags.Append(diagsValues...)
				return IntegrationResourceModel{}, diags
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
					fmt.Sprintf("failed to convert response schema to group response schema, error: %v", err),
				)

				return IntegrationResourceModel{}, diags
			}

			bytes, err := responseSchema.Group.Email.MarshalJSON()
			if err != nil {
				diags.AddError(
					"No data",
					fmt.Sprintf("failed to get maintainer group email bytes, error: %v", err),
				)

				return IntegrationResourceModel{}, diags
			}

			g := &utils.IdEmailModel{
				Id:    utils.TrimmedStringValue(responseSchema.Group.Id.String()),
				Email: utils.TrimmedStringValue(string(bytes)),
			}

			gObject, diagsValues := g.AsObjectValue(ctx)
			if diagsValues.HasError() {
				diags.Append(diagsValues...)
				return IntegrationResourceModel{}, diags
			}

			maintainerGroup := &utils.MaintainerModel{
				Type:   utils.TrimmedStringValue(body.Type),
				Entity: gObject,
			}

			maintainers = append(maintainers, maintainerGroup)
		default:
			diags.AddError("failed invalid type for maintainer", body.Type)
			return IntegrationResourceModel{}, diags
		}
	}

	var agentToken *utils.NameModel
	if len(agentTokenName) != 0 {
		agentToken = &utils.NameModel{
			Name: utils.TrimmedStringValue(agentTokenName),
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
						fmt.Sprintf("failed to unmarshal the prerequisite permissions data, err: %s", err.Error()),
					)
					return IntegrationResourceModel{}, diags
				}

				roleModel, diagsGetRoles := utils.GetRole(ctx, v.Role.Id.String(), v.Role.Name, v.Role.Resource)
				if diagsGetRoles.HasError() {
					diags.Append(diagsGetRoles...)
					return IntegrationResourceModel{}, diags
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

	// Create the Terraform resource model using the extracted data
	return IntegrationResourceModel{
		ID:                                   utils.TrimmedStringValue(data.Id.String()),
		Name:                                 utils.TrimmedStringValue(data.Name),
		AllowedDurations:                     allowedDurationsValues,
		AllowChangingAccountPermissions:      types.BoolValue(data.AllowChangingAccountPermissions),
		AllowCreatingAccounts:                types.BoolValue(data.AllowCreatingAccounts),
		Readonly:                             types.BoolValue(data.Readonly),
		AllowRequests:                        types.BoolValue(data.Requestable),
		AllowRequestsByDefault:               types.BoolValue(data.RequestableByDefault),
		Requestable:                          types.BoolValue(data.Requestable),
		RequestableByDefault:                 types.BoolValue(data.RequestableByDefault),
		AutoAssignRecommendedMaintainers:     types.BoolValue(data.AutoAssignRecommendedMaintainers),
		AutoAssignRecommendedOwners:          types.BoolValue(data.AutoAssignRecommendedOwners),
		NotifyAboutExternalPermissionChanges: types.BoolValue(data.NotifyAboutExternalPermissionChanges),
		Owner: utils.IdEmailModel{
			Id:    utils.TrimmedStringValue(data.Owner.Id.String()),
			Email: utils.TrimmedStringValue(string(marshalJSON)),
		},
		Application: utils.NameModel{
			Name: utils.TrimmedStringValue(strings.ToLower(data.Application.Name)),
		},
		AgentToken: agentToken,
		Workflow: utils.IdNameModel{
			ID:   utils.TrimmedStringValue(data.Workflow.Id.String()),
			Name: utils.TrimmedStringValue(data.Workflow.Name),
		},
		Maintainers:             maintainers,
		ConnectionJson:          utils.TrimmedStringValue(connectionJson),
		PrerequisitePermissions: prerequisitePermissions,
	}, diags
}
