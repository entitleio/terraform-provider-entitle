package integrations

import (
	"context"

	"github.com/entitleio/terraform-provider-entitle/internal/provider/utils"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// This file supports migrating entitle_integration state written by provider versions
// prior to this schema change (owner/workflow/agent_token as nested objects, maintainers
// as {type, entity{id, email}}, prerequisite_permissions as a list of deeply nested role
// objects) onto the current flattened schema (owner_id/workflow_id/agent_token as plain
// strings, maintainers as {type, id}, prerequisite_permissions as a set of {default,
// role_id}). Without this, `terraform plan` fails outright for every existing
// entitle_integration resource the moment a user upgrades the provider.
//
// entitle_integration_gitlab and entitle_integration_bitbucket need no equivalent: they
// were never released with the old schema, so there is no prior state to migrate.

// --- v0 (pre-flatten) shapes, needed only to decode existing state ---------------------

// integrationResourceModelV0 mirrors exactly what is on disk in every
// entitle_integration.tfstate written before this change.
type integrationResourceModelV0 struct {
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
	ConnectionJson                       types.String                        `tfsdk:"connection_json"`
	Application                          *utils.NameModel                    `tfsdk:"application"`
}

// maintainerEntityModelV0 / maintainerModelV0 mirror the old maintainers element shape:
// {type, entity{id, email}}.
type maintainerEntityModelV0 struct {
	ID    types.String `tfsdk:"id"`
	Email types.String `tfsdk:"email"`
}

type maintainerModelV0 struct {
	Type   types.String             `tfsdk:"type"`
	Entity *maintainerEntityModelV0 `tfsdk:"entity"`
}

// priorIntegrationSchemaV0 is the attribute layout entitle_integration shipped with through
// provider v3.x (implicit schema version 0). The framework needs this to know how to decode
// existing state files during UpgradeState; only attribute shape (Required/Optional/Computed
// plus nested Attributes) matters for decoding, so validators/plan modifiers/defaults from
// the original schema are intentionally omitted here.
var priorIntegrationSchemaV0 = schema.Schema{
	Version: 0,
	Attributes: map[string]schema.Attribute{
		"id":   schema.StringAttribute{Computed: true},
		"name": schema.StringAttribute{Required: true},
		"allowed_durations": schema.SetAttribute{
			Required:    true,
			ElementType: types.NumberType,
		},
		"maintainers": schema.SetNestedAttribute{
			Optional: true,
			Computed: true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"type": schema.StringAttribute{Required: true},
					"entity": schema.SingleNestedAttribute{
						Optional: true,
						Attributes: map[string]schema.Attribute{
							"id":    schema.StringAttribute{Required: true},
							"email": schema.StringAttribute{Computed: true},
						},
					},
				},
			},
		},
		"agent_token": schema.SingleNestedAttribute{
			Optional: true,
			Attributes: map[string]schema.Attribute{
				"name": schema.StringAttribute{Required: true},
			},
		},
		"readonly":                                 schema.BoolAttribute{Optional: true, Computed: true},
		"requestable":                              schema.BoolAttribute{Optional: true, Computed: true},
		"requestable_by_default":                   schema.BoolAttribute{Optional: true, Computed: true},
		"auto_assign_recommended_maintainers":      schema.BoolAttribute{Optional: true, Computed: true},
		"auto_assign_recommended_owners":           schema.BoolAttribute{Optional: true, Computed: true},
		"notify_about_external_permission_changes": schema.BoolAttribute{Optional: true, Computed: true},
		"allow_changing_account_permissions":       schema.BoolAttribute{Optional: true, Computed: true},
		"allow_creating_accounts":                  schema.BoolAttribute{Optional: true, Computed: true},
		"prerequisite_permissions": schema.ListNestedAttribute{
			Optional: true,
			NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"default": schema.BoolAttribute{Optional: true, Computed: true},
					"role": schema.SingleNestedAttribute{
						Required: true,
						Attributes: map[string]schema.Attribute{
							"id":   schema.StringAttribute{Required: true},
							"name": schema.StringAttribute{Computed: true},
							"resource": schema.SingleNestedAttribute{
								Computed: true,
								Attributes: map[string]schema.Attribute{
									"id":   schema.StringAttribute{Computed: true},
									"name": schema.StringAttribute{Computed: true},
									"integration": schema.SingleNestedAttribute{
										Computed: true,
										Attributes: map[string]schema.Attribute{
											"id":   schema.StringAttribute{Computed: true},
											"name": schema.StringAttribute{Computed: true},
											"application": schema.SingleNestedAttribute{
												Computed: true,
												Attributes: map[string]schema.Attribute{
													"name": schema.StringAttribute{Computed: true},
												},
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		"owner": schema.SingleNestedAttribute{
			Required: true,
			Attributes: map[string]schema.Attribute{
				"id":    schema.StringAttribute{Required: true},
				"email": schema.StringAttribute{Computed: true},
			},
		},
		"workflow": schema.SingleNestedAttribute{
			Required: true,
			Attributes: map[string]schema.Attribute{
				"id":   schema.StringAttribute{Required: true},
				"name": schema.StringAttribute{Computed: true},
			},
		},
		"connection_json": schema.StringAttribute{Required: true},
		"application": schema.SingleNestedAttribute{
			Required: true,
			Attributes: map[string]schema.Attribute{
				"name": schema.StringAttribute{Required: true},
			},
		},
	},
}

// --- pure remap helpers -----------------------------------------------------------------
// Deliberately kept free of tfsdk.State/schema plumbing so they can be unit tested directly
// with plain Go values, no framework marshaling required.

func upgradeOwnerIDV0(owner *utils.IdEmailModel) types.String {
	if owner == nil {
		return types.StringNull()
	}
	return owner.Id
}

func upgradeWorkflowIDV0(workflow *utils.IdNameModel) types.String {
	if workflow == nil {
		return types.StringNull()
	}
	return workflow.ID
}

func upgradeAgentTokenV0(agentToken *utils.NameModel) types.String {
	if agentToken == nil {
		return types.StringNull()
	}
	return agentToken.Name
}

// upgradeMaintainersV0 converts old {type, entity{id, email}} maintainer models into the
// new flattened {type, id} shape, dropping the email (which the new schema never had).
func upgradeMaintainersV0(old []maintainerModelV0) []IntegrationMaintainerModel {
	result := make([]IntegrationMaintainerModel, 0, len(old))
	for _, m := range old {
		id := types.StringNull()
		if m.Entity != nil {
			id = m.Entity.ID
		}
		result = append(result, IntegrationMaintainerModel{
			Type: m.Type,
			ID:   id,
		})
	}
	return result
}

// upgradePrerequisitePermissionsV0 converts the old list of {default, role{id, name,
// resource{...}}} into the new set of {default, role_id}, dropping everything under role
// except its id.
func upgradePrerequisitePermissionsV0(old []utils.PrerequisitePermissionModel) []utils.ResourcePrerequisitePermissionModel {
	result := make([]utils.ResourcePrerequisitePermissionModel, 0, len(old))
	for _, pp := range old {
		roleID := types.StringNull()
		if pp.Role != nil {
			roleID = pp.Role.ID
		}
		result = append(result, utils.ResourcePrerequisitePermissionModel{
			Default: pp.Default,
			RoleID:  roleID,
		})
	}
	return result
}

// --- UpgradeState wiring ------------------------------------------------------------------

// UpgradeState implements resource.ResourceWithUpgradeState for IntegrationResource. It
// registers the v0 -> v1 (current) migration described above.
func (r *IntegrationResource) UpgradeState(ctx context.Context) map[int64]resource.StateUpgrader {
	return map[int64]resource.StateUpgrader{
		0: {
			PriorSchema:   &priorIntegrationSchemaV0,
			StateUpgrader: upgradeIntegrationResourceStateV0toV1,
		},
	}
}

func upgradeIntegrationResourceStateV0toV1(ctx context.Context, req resource.UpgradeStateRequest, resp *resource.UpgradeStateResponse) {
	var old integrationResourceModelV0
	resp.Diagnostics.Append(req.State.Get(ctx, &old)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var maintainersSet types.Set
	if old.Maintainers.IsNull() || old.Maintainers.IsUnknown() {
		maintainersSet = types.SetNull(maintainerElementTypeV1)
	} else {
		var oldMaintainers []maintainerModelV0
		resp.Diagnostics.Append(old.Maintainers.ElementsAs(ctx, &oldMaintainers, false)...)
		if resp.Diagnostics.HasError() {
			return
		}

		newMaintainers := upgradeMaintainersV0(oldMaintainers)

		var setDiags diag.Diagnostics
		if len(newMaintainers) == 0 {
			maintainersSet = types.SetNull(maintainerElementTypeV1)
		} else {
			maintainersSet, setDiags = types.SetValueFrom(ctx, maintainerElementTypeV1, newMaintainers)
			resp.Diagnostics.Append(setDiags...)
			if resp.Diagnostics.HasError() {
				return
			}
		}
	}

	newPrereqs := upgradePrerequisitePermissionsV0(old.PrerequisitePermissions)
	var prereqSet types.Set
	if len(newPrereqs) == 0 {
		prereqSet = types.SetNull(prerequisitePermissionElementTypeV1)
	} else {
		var ppDiags diag.Diagnostics
		prereqSet, ppDiags = types.SetValueFrom(ctx, prerequisitePermissionElementTypeV1, newPrereqs)
		resp.Diagnostics.Append(ppDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	newState := IntegrationResourceModel{
		BaseIntegrationResourceModel: BaseIntegrationResourceModel{
			ID:                                   old.ID,
			Name:                                 old.Name,
			AllowedDurations:                     old.AllowedDurations,
			AllowChangingAccountPermissions:      old.AllowChangingAccountPermissions,
			AllowCreatingAccounts:                old.AllowCreatingAccounts,
			Readonly:                             old.Readonly,
			Requestable:                          old.Requestable,
			RequestableByDefault:                 old.RequestableByDefault,
			AutoAssignRecommendedMaintainers:     old.AutoAssignRecommendedMaintainers,
			AutoAssignRecommendedOwners:          old.AutoAssignRecommendedOwners,
			NotifyAboutExternalPermissionChanges: old.NotifyAboutExternalPermissionChanges,
			OwnerID:                              upgradeOwnerIDV0(old.Owner),
			AgentToken:                           upgradeAgentTokenV0(old.AgentToken),
			WorkflowID:                           upgradeWorkflowIDV0(old.Workflow),
			Maintainers:                          maintainersSet,
			PrerequisitePermissions:              prereqSet,
		},
		ConnectionJson: old.ConnectionJson,
		Application:    old.Application,
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, newState)...)
}

// maintainerElementTypeV1 / prerequisitePermissionElementTypeV1 mirror the element types
// built in ConvertBaseIntegrationResultToBaseModel (base_integration.go) for the current
// (v1) maintainers/prerequisite_permissions set shapes.
var maintainerElementTypeV1 = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"type": types.StringType,
		"id":   types.StringType,
	},
}

var prerequisitePermissionElementTypeV1 = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"default": types.BoolType,
		"role_id": types.StringType,
	},
}
