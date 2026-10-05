package integrations

import (
	"context"
	"testing"

	"github.com/entitleio/terraform-provider-entitle/internal/provider/utils"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// --- unit tests for the pure remap helpers -----------------------------------------------
// These cover the actual risk in the migration (mapping the old shape onto the new one)
// without needing any tfsdk.State/schema marshaling.

func TestUpgradeOwnerIDV0(t *testing.T) {
	if got := upgradeOwnerIDV0(nil); !got.IsNull() {
		t.Errorf("expected null for nil owner, got %v", got)
	}

	owner := &utils.IdEmailModel{Id: types.StringValue("owner-1"), Email: types.StringValue("owner@example.com")}
	if got := upgradeOwnerIDV0(owner); got.ValueString() != "owner-1" {
		t.Errorf("expected owner-1, got %v", got)
	}
}

func TestUpgradeWorkflowIDV0(t *testing.T) {
	if got := upgradeWorkflowIDV0(nil); !got.IsNull() {
		t.Errorf("expected null for nil workflow, got %v", got)
	}

	workflow := &utils.IdNameModel{ID: types.StringValue("workflow-1"), Name: types.StringValue("wf-name")}
	if got := upgradeWorkflowIDV0(workflow); got.ValueString() != "workflow-1" {
		t.Errorf("expected workflow-1, got %v", got)
	}
}

func TestUpgradeAgentTokenV0(t *testing.T) {
	if got := upgradeAgentTokenV0(nil); !got.IsNull() {
		t.Errorf("expected null for nil agent token, got %v", got)
	}

	agentToken := &utils.NameModel{Name: types.StringValue("agent-token-name")}
	if got := upgradeAgentTokenV0(agentToken); got.ValueString() != "agent-token-name" {
		t.Errorf("expected agent-token-name, got %v", got)
	}
}

func TestUpgradeMaintainersV0(t *testing.T) {
	old := []maintainerModelV0{
		{
			Type:   types.StringValue("user"),
			Entity: &maintainerEntityModelV0{ID: types.StringValue("user-1"), Email: types.StringValue("user1@example.com")},
		},
		{
			Type:   types.StringValue("group"),
			Entity: &maintainerEntityModelV0{ID: types.StringValue("group-1"), Email: types.StringNull()},
		},
		{
			// entity missing entirely - should map to a null id, not panic.
			Type:   types.StringValue("user"),
			Entity: nil,
		},
	}

	got := upgradeMaintainersV0(old)
	if len(got) != 3 {
		t.Fatalf("expected 3 maintainers, got %d", len(got))
	}

	if got[0].Type.ValueString() != "user" || got[0].ID.ValueString() != "user-1" {
		t.Errorf("unexpected maintainer[0]: %+v", got[0])
	}
	if got[1].Type.ValueString() != "group" || got[1].ID.ValueString() != "group-1" {
		t.Errorf("unexpected maintainer[1]: %+v", got[1])
	}
	if got[2].Type.ValueString() != "user" || !got[2].ID.IsNull() {
		t.Errorf("expected maintainer[2] to have a null id, got %+v", got[2])
	}
}

func TestUpgradePrerequisitePermissionsV0(t *testing.T) {
	resourceAttrType, ok := utils.Role{}.AttributeTypes()["resource"].(types.ObjectType)
	if !ok {
		t.Fatal("utils.Role{}.AttributeTypes()[\"resource\"] is not a types.ObjectType")
	}
	roleResourceNull := types.ObjectNull(resourceAttrType.AttrTypes)

	old := []utils.PrerequisitePermissionModel{
		{
			Default: types.BoolValue(true),
			Role: &utils.Role{
				ID:       types.StringValue("role-1"),
				Name:     types.StringValue("Role One"),
				Resource: roleResourceNull,
			},
		},
		{
			// role missing entirely - should map to a null role_id, not panic.
			Default: types.BoolValue(false),
			Role:    nil,
		},
	}

	got := upgradePrerequisitePermissionsV0(old)
	if len(got) != 2 {
		t.Fatalf("expected 2 prerequisite permissions, got %d", len(got))
	}

	if !got[0].Default.ValueBool() || got[0].RoleID.ValueString() != "role-1" {
		t.Errorf("unexpected prerequisite permission[0]: %+v", got[0])
	}
	if got[1].Default.ValueBool() || !got[1].RoleID.IsNull() {
		t.Errorf("unexpected prerequisite permission[1]: %+v", got[1])
	}
}

// --- full round-trip test through the actual StateUpgrader --------------------------------
// Exercises the tfsdk.State plumbing (Get old shape / Set new shape) end to end, which the
// pure-function tests above intentionally skip. This is the part most likely to break from a
// tfsdk tag or schema-shape mismatch, so it's worth the extra setup.

func TestUpgradeIntegrationResourceStateV0toV1(t *testing.T) {
	ctx := context.Background()

	resourceAttrType, ok := utils.Role{}.AttributeTypes()["resource"].(types.ObjectType)
	if !ok {
		t.Fatal("utils.Role{}.AttributeTypes()[\"resource\"] is not a types.ObjectType")
	}
	roleResourceNull := types.ObjectNull(resourceAttrType.AttrTypes)

	maintainerObjectTypeV0 := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"type": types.StringType,
			"entity": types.ObjectType{
				AttrTypes: map[string]attr.Type{
					"id":    types.StringType,
					"email": types.StringType,
				},
			},
		},
	}

	oldMaintainers, diags := types.SetValueFrom(ctx, maintainerObjectTypeV0, []maintainerModelV0{
		{
			Type:   types.StringValue("user"),
			Entity: &maintainerEntityModelV0{ID: types.StringValue("user-1"), Email: types.StringValue("user1@example.com")},
		},
		{
			Type:   types.StringValue("group"),
			Entity: &maintainerEntityModelV0{ID: types.StringValue("group-1"), Email: types.StringNull()},
		},
	})
	if diags.HasError() {
		t.Fatalf("failed building old maintainers set: %v", diags)
	}

	oldAllowedDurations, diags := types.SetValueFrom(ctx, types.NumberType, []float64{3600, 86400})
	if diags.HasError() {
		t.Fatalf("failed building old allowed_durations set: %v", diags)
	}

	oldModel := integrationResourceModelV0{
		ID:                                   types.StringValue("11111111-1111-1111-1111-111111111111"),
		Name:                                 types.StringValue("test-integration"),
		AllowedDurations:                     oldAllowedDurations,
		AllowChangingAccountPermissions:      types.BoolValue(true),
		AllowCreatingAccounts:                types.BoolValue(true),
		Readonly:                             types.BoolValue(false),
		Requestable:                          types.BoolValue(true),
		RequestableByDefault:                 types.BoolValue(true),
		AutoAssignRecommendedMaintainers:     types.BoolValue(true),
		AutoAssignRecommendedOwners:          types.BoolValue(true),
		NotifyAboutExternalPermissionChanges: types.BoolValue(true),
		Owner:                                &utils.IdEmailModel{Id: types.StringValue("owner-1"), Email: types.StringValue("owner@example.com")},
		AgentToken:                           &utils.NameModel{Name: types.StringValue("agent-token-name")},
		Workflow:                             &utils.IdNameModel{ID: types.StringValue("workflow-1"), Name: types.StringValue("wf-name")},
		Maintainers:                          oldMaintainers,
		PrerequisitePermissions: []utils.PrerequisitePermissionModel{
			{
				Default: types.BoolValue(true),
				Role: &utils.Role{
					ID:       types.StringValue("role-1"),
					Name:     types.StringValue("Role One"),
					Resource: roleResourceNull,
				},
			},
		},
		ConnectionJson: types.StringValue("{}"),
		Application:    &utils.NameModel{Name: types.StringValue("slack")},
	}

	oldState := tfsdk.State{Schema: priorIntegrationSchemaV0}
	if diags := oldState.Set(ctx, &oldModel); diags.HasError() {
		t.Fatalf("failed to build old state fixture: %v", diags)
	}

	// Use the resource's real, current Schema() so the new state is decoded against
	// whatever entitle_integration actually looks like today - not a hand-copied version
	// that could silently drift from the production schema.
	r := &IntegrationResource{}
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("failed to build current schema: %v", schemaResp.Diagnostics)
	}

	upgradeReq := resource.UpgradeStateRequest{State: &oldState}
	upgradeResp := &resource.UpgradeStateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	upgraders := r.UpgradeState(ctx)
	upgrader, ok := upgraders[0]
	if !ok {
		t.Fatal("no StateUpgrader registered for prior schema version 0")
	}

	upgrader.StateUpgrader(ctx, upgradeReq, upgradeResp)
	if upgradeResp.Diagnostics.HasError() {
		t.Fatalf("StateUpgrader returned errors: %v", upgradeResp.Diagnostics)
	}

	var newModel IntegrationResourceModel
	if diags := upgradeResp.State.Get(ctx, &newModel); diags.HasError() {
		t.Fatalf("failed to read back upgraded state: %v", diags)
	}

	if newModel.OwnerID.ValueString() != "owner-1" {
		t.Errorf("expected owner_id owner-1, got %q", newModel.OwnerID.ValueString())
	}
	if newModel.WorkflowID.ValueString() != "workflow-1" {
		t.Errorf("expected workflow_id workflow-1, got %q", newModel.WorkflowID.ValueString())
	}
	if newModel.AgentToken.ValueString() != "agent-token-name" {
		t.Errorf("expected agent_token agent-token-name, got %q", newModel.AgentToken.ValueString())
	}
	if newModel.ConnectionJson.ValueString() != "{}" {
		t.Errorf("expected connection_json to survive untouched, got %q", newModel.ConnectionJson.ValueString())
	}
	if newModel.Application == nil || newModel.Application.Name.ValueString() != "slack" {
		t.Errorf("expected application.name slack to survive untouched, got %+v", newModel.Application)
	}

	var newMaintainers []IntegrationMaintainerModel
	if diags := newModel.Maintainers.ElementsAs(ctx, &newMaintainers, false); diags.HasError() {
		t.Fatalf("failed to read back maintainers: %v", diags)
	}
	if len(newMaintainers) != 2 {
		t.Fatalf("expected 2 maintainers, got %d", len(newMaintainers))
	}
	byType := map[string]string{}
	for _, m := range newMaintainers {
		byType[m.Type.ValueString()] = m.ID.ValueString()
	}
	if byType["user"] != "user-1" || byType["group"] != "group-1" {
		t.Errorf("unexpected maintainers after upgrade: %+v", newMaintainers)
	}

	var newPrereqs []utils.ResourcePrerequisitePermissionModel
	if diags := newModel.PrerequisitePermissions.ElementsAs(ctx, &newPrereqs, false); diags.HasError() {
		t.Fatalf("failed to read back prerequisite_permissions: %v", diags)
	}
	if len(newPrereqs) != 1 {
		t.Fatalf("expected 1 prerequisite permission, got %d", len(newPrereqs))
	}
	if !newPrereqs[0].Default.ValueBool() || newPrereqs[0].RoleID.ValueString() != "role-1" {
		t.Errorf("unexpected prerequisite permission after upgrade: %+v", newPrereqs[0])
	}
}
