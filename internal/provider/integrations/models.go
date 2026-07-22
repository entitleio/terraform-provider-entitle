package integrations

import (
	"maps"

	"github.com/entitleio/terraform-provider-entitle/internal/provider/utils"
	"github.com/entitleio/terraform-provider-entitle/internal/validators"
	"github.com/hashicorp/terraform-plugin-framework-validators/boolvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/setvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/objectplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/setplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// Constants for default values of integration settings.
const (
	defaultIntegrationAllowChangingAccountPermissions      = true
	defaultIntegrationAllowCreatingAccounts                = true
	defaultIntegrationReadonly                             = false
	defaultIntegrationAllowRequests                        = true
	defaultIntegrationAllowRequestsByDefault               = true
	defaultIntegrationAllowAsGrantMethod                   = false
	defaultIntegrationAllowAsGrantMethodByDefault          = false
	defaultIntegrationAutoAssignRecommendedMaintainers     = true
	defaultIntegrationAutoAssignRecommendedOwners          = true
	defaultIntegrationNotifyAboutExternalPermissionChanges = true
)

type applicationName string

const (
	applicationGitlab  applicationName = "gitlab"
	applicationVirtual applicationName = "virtual application"
)

func (a applicationName) String() string {
	return string(a)
}

func (i applicationName) canCreateActors() *bool {
	var v bool
	switch i {
	case applicationGitlab:
		v = false
	default:
		return nil
	}

	return &v
}

func (i applicationName) canEditPermissions() *bool {
	var v bool
	switch i {
	case applicationGitlab:
		v = true
	default:
		return nil
	}

	return &v
}

// BaseIntegrationResourceModel describes the base resource model.
type BaseIntegrationResourceModel struct {
	ID                                   types.String                        `tfsdk:"id"`
	Name                                 types.String                        `tfsdk:"name"`
	AllowedDurations                     types.Set                           `tfsdk:"allowed_durations"`
	AllowChangingAccountPermissions      types.Bool                          `tfsdk:"allow_changing_account_permissions"`
	AllowCreatingAccounts                types.Bool                          `tfsdk:"allow_creating_accounts"`
	Readonly                             types.Bool                          `tfsdk:"readonly"`
	Requestable                          types.Bool                          `tfsdk:"requestable"`
	RequestableByDefault                 types.Bool                          `tfsdk:"requestable_by_default"`
	AutoAssignRecommendedMaintainers     types.Bool                          `tfsdk:"auto_assign_recommended_maintainers"`
	AutoAssignRecommendedOwners          types.Bool                          `tfsdk:"auto_assign_recommended_owners"`
	NotifyAboutExternalPermissionChanges types.Bool                          `tfsdk:"notify_about_external_permission_changes"`
	Owner                                *utils.IdEmailModel                 `tfsdk:"owner"`
	AgentToken                           *utils.NameModel                    `tfsdk:"agent_token"`
	Workflow                             *utils.IdNameModel                  `tfsdk:"workflow"`
	Maintainers                          types.Set                           `tfsdk:"maintainers"`
	PrerequisitePermissions              []utils.PrerequisitePermissionModel `tfsdk:"prerequisite_permissions"`
}

var BaseIntegrationResourceAttributes = map[string]schema.Attribute{
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
			stringvalidator.LengthBetween(2, 50),
		},
	},
	"allowed_durations": schema.SetAttribute{
		Required:            true,
		ElementType:         types.NumberType,
		Description:         "As the admin, you can set different durations for the integration, compared to the workflow linked to it.  \nAllowed values:\n  - 1800 - 30min\n  - 3600 - 1 hour\n  - 10800 - 3 hours\n  - 21600 - 6 hours\n  - 43200 - 12 hours\n  - 57600 - 16 hours\n  - 86400 - 24 hours\n  - 259200 - 3 days\n  - 604800 - 7 days\n  - 2628000  - ~30,4 days\n  - 7884000 - 91,25 days\n  - 15768000 - 182,5 days\n  - 31536000 - 365 days\n  - 63072000 - 730 days\n  - -1 - unlimited",
		MarkdownDescription: "As the admin, you can set different durations for the integration, compared to the workflow linked to it.  \nAllowed values:\n  - 1800 - 30min\n  - 3600 - 1 hour\n  - 10800 - 3 hours\n  - 21600 - 6 hours\n  - 43200 - 12 hours\n  - 57600 - 16 hours\n  - 86400 - 24 hours\n  - 259200 - 3 days\n  - 604800 - 7 days\n  - 2628000  - ~30,4 days\n  - 7884000 - 91,25 days\n  - 15768000 - 182,5 days\n  - 31536000 - 365 days\n  - 63072000 - 730 days\n  - -1 - unlimited",
		Validators: []validator.Set{
			setvalidator.SizeAtLeast(1),
		},
	},
	"maintainers": schema.SetNestedAttribute{
		NestedObject: schema.NestedAttributeObject{
			Attributes: map[string]schema.Attribute{
				"type": schema.StringAttribute{
					Required:            true,
					Description:         "\"user\" or \"group\"",
					MarkdownDescription: "\"user\" or \"group\"",
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
		Computed: true,
		Validators: []validator.Set{
			setvalidator.SizeAtLeast(1),
		},
		PlanModifiers: []planmodifier.Set{
			setplanmodifier.UseStateForUnknown(),
		},
		Description: "Maintainer of the resource, second tier owner of that resource you can " +
			"have multiple resource Maintainer also can be IDP group. In the case of the bundle the Maintainer of each Resource.",
		MarkdownDescription: "Maintainer of the resource, second tier owner of that resource you can " +
			"have multiple resource Maintainer also can be IDP group. In the case of the bundle the Maintainer of each Resource.",
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
		MarkdownDescription: "Agent token configuration. Used for agent-based integrations where Entitle needs a token to authenticate.",
	},
	"readonly": schema.BoolAttribute{
		Optional: true,
		Computed: true,
		Description: "If turned on, any request opened by a user will not be automatically granted, " +
			"instead a ticket will be opened for manual resolution. (default: false)",
		MarkdownDescription: "If turned on, any request opened by a user will not be automatically granted, " +
			"instead a ticket will be opened for manual resolution. (default: false)",
		Default: booldefault.StaticBool(defaultIntegrationReadonly),
		PlanModifiers: []planmodifier.Bool{
			boolplanmodifier.UseStateForUnknown(),
		},
	},
	"requestable": schema.BoolAttribute{
		Optional: true,
		Computed: true,
		Description: "Controls whether a user can create requests for entitlements for resources " +
			"under the integration. (default: true)",
		MarkdownDescription: "Controls whether a user can create requests for entitlements for resources " +
			"under the integration. (default: true)",
		Default: booldefault.StaticBool(defaultIntegrationAllowRequests),
		PlanModifiers: []planmodifier.Bool{
			boolplanmodifier.UseStateForUnknown(),
		},
	},
	"requestable_by_default": schema.BoolAttribute{
		Optional: true,
		Computed: true,
		Description: "Controls whether resources that are added to the integration could be shown " +
			"to the user. (default: true)",
		MarkdownDescription: "Controls whether resources that are added to the integration could be shown " +
			"to the user. (default: true)",
		Default: booldefault.StaticBool(defaultIntegrationAllowRequestsByDefault),
		PlanModifiers: []planmodifier.Bool{
			boolplanmodifier.UseStateForUnknown(),
		},
	},
	"auto_assign_recommended_maintainers": schema.BoolAttribute{
		Optional:            true,
		Computed:            true,
		Description:         "When enabled, Entitle automatically assigns suggested maintainers to the integration based on usage patterns and access signals. (default: true)",
		MarkdownDescription: "When enabled, Entitle automatically assigns suggested maintainers to the integration based on usage patterns and access signals. (default: true)",
		Default:             booldefault.StaticBool(defaultIntegrationAutoAssignRecommendedMaintainers),
		PlanModifiers: []planmodifier.Bool{
			boolplanmodifier.UseStateForUnknown(),
		},
	},
	"auto_assign_recommended_owners": schema.BoolAttribute{
		Optional:            true,
		Computed:            true,
		Description:         "When enabled, Entitle automatically assigns suggested owners to the integration based on ownership signals, such as group ownership or historical access. (default: true)",
		MarkdownDescription: "When enabled, Entitle automatically assigns suggested owners to the integration based on ownership signals, such as group ownership or historical access. (default: true)",
		Default:             booldefault.StaticBool(defaultIntegrationAutoAssignRecommendedOwners),
		PlanModifiers: []planmodifier.Bool{
			boolplanmodifier.UseStateForUnknown(),
		},
	},
	"notify_about_external_permission_changes": schema.BoolAttribute{
		Optional:            true,
		Computed:            true,
		Description:         "When enabled, Entitle will notify owners if permissions are changed directly in the connected application, bypassing Entitle. (default: true)",
		MarkdownDescription: "When enabled, Entitle will notify owners if permissions are changed directly in the connected application, bypassing Entitle. (default: true)",
		Default:             booldefault.StaticBool(defaultIntegrationNotifyAboutExternalPermissionChanges),
		PlanModifiers: []planmodifier.Bool{
			boolplanmodifier.UseStateForUnknown(),
		},
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
					PlanModifiers: []planmodifier.Bool{
						boolplanmodifier.UseStateForUnknown(),
					},
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
							PlanModifiers: []planmodifier.String{
								stringplanmodifier.UseStateForUnknown(),
							},
						},
						"resource": schema.SingleNestedAttribute{
							Attributes: map[string]schema.Attribute{
								"id": schema.StringAttribute{
									Computed:            true,
									Description:         "The unique identifier of the resource.",
									MarkdownDescription: "The unique identifier of the resource.",
									PlanModifiers: []planmodifier.String{
										stringplanmodifier.UseStateForUnknown(),
									},
								},
								"name": schema.StringAttribute{
									Computed:            true,
									Description:         "The display name of the resource.",
									MarkdownDescription: "The display name of the resource.",
									PlanModifiers: []planmodifier.String{
										stringplanmodifier.UseStateForUnknown(),
									},
								},
								"integration": schema.SingleNestedAttribute{
									Attributes: map[string]schema.Attribute{
										"id": schema.StringAttribute{
											Computed:            true,
											Description:         "The identifier of the integration.",
											MarkdownDescription: "The identifier of the integration.",
											PlanModifiers: []planmodifier.String{
												stringplanmodifier.UseStateForUnknown(),
											},
										},
										"name": schema.StringAttribute{
											Computed:            true,
											Description:         "The display name of the integration.",
											MarkdownDescription: "The display name of the integration.",
											PlanModifiers: []planmodifier.String{
												stringplanmodifier.UseStateForUnknown(),
											},
										},
										"application": schema.SingleNestedAttribute{
											Attributes: map[string]schema.Attribute{
												"name": schema.StringAttribute{
													Computed:            true,
													Description:         "The name of the connected application.",
													MarkdownDescription: "The name of the connected application.",
													PlanModifiers: []planmodifier.String{
														stringplanmodifier.UseStateForUnknown(),
													},
												},
											},
											Computed:            true,
											Description:         "The application that the integration is connected to.",
											MarkdownDescription: "The application that the integration is connected to.",
											PlanModifiers: []planmodifier.Object{
												objectplanmodifier.UseStateForUnknown(),
											},
										},
									},
									Computed:            true,
									Description:         "The integration that the resource belongs to.",
									MarkdownDescription: "The integration that the resource belongs to.",
									PlanModifiers: []planmodifier.Object{
										objectplanmodifier.UseStateForUnknown(),
									},
								},
							},
							Computed:            true,
							Description:         "The specific resource associated with the role.",
							MarkdownDescription: "The specific resource associated with the role.",
							PlanModifiers: []planmodifier.Object{
								objectplanmodifier.UseStateForUnknown(),
							},
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
		Required: true,
		Description: "The default approval workflow for entitlements for the integration " +
			"(can be overwritten on resource/role level).",
		MarkdownDescription: "The default approval workflow for entitlements for the integration " +
			"(can be overwritten on resource/role level).",
	},
	"allow_changing_account_permissions": schema.BoolAttribute{
		Optional:            true,
		Computed:            true,
		Description:         "Controls whether Entitle can modify the permissions of accounts under this integration. If disabled, Entitle can only read permissions but cannot grant or revoke them. (default: true)",
		MarkdownDescription: "Controls whether Entitle can modify the permissions of accounts under this integration. If disabled, Entitle can only read permissions but cannot grant or revoke them. (default: true)",
		Default:             booldefault.StaticBool(defaultIntegrationAllowChangingAccountPermissions),
		PlanModifiers: []planmodifier.Bool{
			boolplanmodifier.UseStateForUnknown(),
			boolplanmodifier.RequiresReplace(),
		},
	},
	"allow_creating_accounts": schema.BoolAttribute{
		Optional:            true,
		Computed:            true,
		Description:         "Controls whether Entitle is allowed to create new user accounts in the connected application when access is requested. If disabled, users must already exist in the application before access can be granted. (default: true)",
		MarkdownDescription: "Controls whether Entitle is allowed to create new user accounts in the connected application when access is requested. If disabled, users must already exist in the application before access can be granted. (default: true)",
		Default:             booldefault.StaticBool(defaultIntegrationAllowCreatingAccounts),
		PlanModifiers: []planmodifier.Bool{
			boolplanmodifier.UseStateForUnknown(),
			boolplanmodifier.RequiresReplace(),
		},
	},
}

func GetBaseIntegrationResourceAttributes(appName applicationName) map[string]schema.Attribute {
	m := maps.Clone(BaseIntegrationResourceAttributes)

	if canCreateActors := appName.canCreateActors(); canCreateActors != nil {
		v := *canCreateActors
		allowCreatingAccounts, ok := m["allow_creating_accounts"].(schema.BoolAttribute)
		if !ok {
			panic("GetBaseIntegrationResourceAttributes: \"allow_creating_accounts\" is missing or not a schema.BoolAttribute")
		}

		allowCreatingAccounts.Validators = []validator.Bool{
			boolvalidator.Equals(v),
		}
		allowCreatingAccounts.Default = booldefault.StaticBool(v)

		m["allow_creating_accounts"] = allowCreatingAccounts
	}

	if canEditPermissions := appName.canEditPermissions(); canEditPermissions != nil {
		v := *canEditPermissions
		allowChangingAccountPermissions, ok := m["allow_changing_account_permissions"].(schema.BoolAttribute)
		if !ok {
			panic("GetBaseIntegrationResourceAttributes: \"allow_changing_account_permissions\" is missing or not a schema.BoolAttribute")
		}

		allowChangingAccountPermissions.Validators = []validator.Bool{
			boolvalidator.Equals(v),
		}
		allowChangingAccountPermissions.Default = booldefault.StaticBool(v)

		m["allow_changing_account_permissions"] = allowChangingAccountPermissions
	}

	return m
}
