//go:build acceptance

package workflows

import (
	"context"
	"math/big"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/entitleio/terraform-provider-entitle/internal/provider/utils"
)

// TestGetWorkflowsRules_NotifiedEntityIDs verifies that notified entity IDs
// are sent without surrounding quotes. This was a bug where .String() was used
// instead of .ValueString() on types.String fields, causing the API to receive
// IDs like `"\"uuid\""` and return "Could not find all users".
func TestGetWorkflowsRules_NotifiedEntityIDs(t *testing.T) {
	ctx := context.Background()

	const (
		userID     = "aaaaaaaa-1111-2222-3333-444444444444"
		groupID    = "bbbbbbbb-1111-2222-3333-444444444444"
		scheduleID = "cccccccc-1111-2222-3333-444444444444"
	)

	userObj, diags := types.ObjectValue(
		map[string]attr.Type{"id": types.StringType, "email": types.StringType},
		map[string]attr.Value{"id": types.StringValue(userID), "email": types.StringNull()},
	)
	if diags.HasError() {
		t.Fatalf("failed to create user object: %s", diags.Errors())
	}

	groupObj, diags := types.ObjectValue(
		map[string]attr.Type{"id": types.StringType, "name": types.StringType},
		map[string]attr.Value{"id": types.StringValue(groupID), "name": types.StringNull()},
	)
	if diags.HasError() {
		t.Fatalf("failed to create group object: %s", diags.Errors())
	}

	scheduleObj, diags := types.ObjectValue(
		map[string]attr.Type{"id": types.StringType, "name": types.StringType},
		map[string]attr.Value{"id": types.StringValue(scheduleID), "name": types.StringNull()},
	)
	if diags.HasError() {
		t.Fatalf("failed to create schedule object: %s", diags.Errors())
	}

	nullUser := types.ObjectNull(utils.IdEmailModel{}.AttributeTypes())
	nullGroup := types.ObjectNull(utils.IdNameModel{}.AttributeTypes())
	nullSchedule := types.ObjectNull(utils.IdNameModel{}.AttributeTypes())

	planRules := []*workflowRulesModel{
		{
			SortOrder:     types.NumberValue(big.NewFloat(0)),
			UnderDuration: types.NumberValue(big.NewFloat(3600)),
			AnySchedule:   types.BoolValue(true),
			ApprovalFlow: &workflowRulesApprovalFlowModel{
				Steps: []*workflowRulesApprovalFlowStepModel{
					{
						SortOrder: types.NumberValue(big.NewFloat(0)),
						Operator:  types.StringValue("and"),
						NotifiedEntities: []*workflowRulesApprovalFlowStepApprovalNotifiedModel{
							{
								Type:     types.StringValue("user"),
								User:     userObj,
								Group:    nullGroup,
								Schedule: nullSchedule,
								Webhook:  types.ObjectNull((&utils.IdNameModel{}).AttributeTypes()),
								Channel:  types.ObjectNull((&utils.IdentityOnlyModel{}).AttributeTypes()),
							},
							{
								Type:     types.StringValue("group"),
								User:     nullUser,
								Group:    groupObj,
								Schedule: nullSchedule,
								Webhook:  types.ObjectNull((&utils.IdNameModel{}).AttributeTypes()),
								Channel:  types.ObjectNull((&utils.IdentityOnlyModel{}).AttributeTypes()),
							},
							{
								Type:     types.StringValue("schedule"),
								User:     nullUser,
								Group:    nullGroup,
								Schedule: scheduleObj,
								Webhook:  types.ObjectNull((&utils.IdNameModel{}).AttributeTypes()),
								Channel:  types.ObjectNull((&utils.IdentityOnlyModel{}).AttributeTypes()),
							},
						},
						ApprovalEntities: []*workflowRulesApprovalFlowStepApprovalNotifiedModel{
							{
								Type:     types.StringValue("Automatic"),
								User:     nullUser,
								Group:    nullGroup,
								Schedule: nullSchedule,
								Webhook:  types.ObjectNull((&utils.IdNameModel{}).AttributeTypes()),
								Channel:  types.ObjectNull((&utils.IdentityOnlyModel{}).AttributeTypes()),
							},
						},
					},
				},
			},
		},
	}

	rules, diags := getWorkflowsRules(ctx, planRules)
	if diags.HasError() {
		t.Fatalf("getWorkflowsRules returned errors: %s", diags.Errors())
	}

	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}

	notified := rules[0].ApprovalFlow.Steps[0].NotifiedEntities
	if len(notified) != 3 {
		t.Fatalf("expected 3 notified entities, got %d", len(notified))
	}

	// Verify user notified entity ID has no surrounding quotes
	userEntity, err := notified[0].AsApprovalEntityUserSchema()
	if err != nil {
		t.Fatalf("failed to extract user entity: %v", err)
	}
	if userEntity.Entity.Id != userID {
		t.Errorf("notified user entity ID = %q, want %q", userEntity.Entity.Id, userID)
	}

	// Verify group notified entity ID has no surrounding quotes
	groupEntity, err := notified[1].AsApprovalEntityGroupSchema()
	if err != nil {
		t.Fatalf("failed to extract group entity: %v", err)
	}
	if groupEntity.Entity.Id != groupID {
		t.Errorf("notified group entity ID = %q, want %q", groupEntity.Entity.Id, groupID)
	}

	// Verify schedule notified entity ID has no surrounding quotes
	schedEntity, err := notified[2].AsApprovalEntityScheduleSchema()
	if err != nil {
		t.Fatalf("failed to extract schedule entity: %v", err)
	}
	if schedEntity.Entity.Id != scheduleID {
		t.Errorf("notified schedule entity ID = %q, want %q", schedEntity.Entity.Id, scheduleID)
	}
}

// makeGroupEntity creates a group approval entity with the given ID and name.
func makeGroupEntity(t *testing.T, id, name string) *workflowRulesApprovalFlowStepApprovalNotifiedModel {
	t.Helper()
	ctx := context.Background()

	v := utils.IdNameModel{
		ID:   types.StringValue(id),
		Name: types.StringValue(name),
	}
	vObj, diags := v.AsObjectValue(ctx)
	if diags.HasError() {
		t.Fatalf("failed to create group object: %v", diags.Errors())
	}

	return &workflowRulesApprovalFlowStepApprovalNotifiedModel{
		Type:     types.StringValue("directory_group"),
		Group:    vObj,
		User:     types.ObjectNull((&utils.IdEmailModel{}).AttributeTypes()),
		Schedule: types.ObjectNull((&utils.IdNameModel{}).AttributeTypes()),
		Webhook:  types.ObjectNull((&utils.IdNameModel{}).AttributeTypes()),
		Channel:  types.ObjectNull((&utils.IdentityOnlyModel{}).AttributeTypes()),
	}
}

// makeUserEntity creates a user approval entity with the given ID and email.
func makeUserEntity(t *testing.T, id, email string) *workflowRulesApprovalFlowStepApprovalNotifiedModel {
	t.Helper()
	ctx := context.Background()

	v := utils.IdEmailModel{
		Id:    types.StringValue(id),
		Email: types.StringValue(email),
	}
	vObj, diags := v.AsObjectValue(ctx)
	if diags.HasError() {
		t.Fatalf("failed to create user object: %v", diags.Errors())
	}

	return &workflowRulesApprovalFlowStepApprovalNotifiedModel{
		Type:     types.StringValue("user"),
		User:     vObj,
		Group:    types.ObjectNull((&utils.IdNameModel{}).AttributeTypes()),
		Schedule: types.ObjectNull((&utils.IdNameModel{}).AttributeTypes()),
		Webhook:  types.ObjectNull((&utils.IdNameModel{}).AttributeTypes()),
		Channel:  types.ObjectNull((&utils.IdentityOnlyModel{}).AttributeTypes()),
	}
}

// makeWebhookEntity creates a webhook approval entity with the given ID and name.
func makeWebhookEntity(t *testing.T, id, name string) *workflowRulesApprovalFlowStepApprovalNotifiedModel {
	t.Helper()
	ctx := context.Background()

	v := utils.IdNameModel{
		ID:   types.StringValue(id),
		Name: types.StringValue(name),
	}
	vObj, diags := v.AsObjectValue(ctx)
	if diags.HasError() {
		t.Fatalf("failed to create webhook object: %v", diags.Errors())
	}

	return &workflowRulesApprovalFlowStepApprovalNotifiedModel{
		Type:     types.StringValue("Webhook"),
		User:     types.ObjectNull((&utils.IdEmailModel{}).AttributeTypes()),
		Group:    types.ObjectNull((&utils.IdNameModel{}).AttributeTypes()),
		Schedule: types.ObjectNull((&utils.IdNameModel{}).AttributeTypes()),
		Webhook:  vObj,
		Channel:  types.ObjectNull((&utils.IdentityOnlyModel{}).AttributeTypes()),
	}
}

// makeNullEntity creates an approval entity without a user/group/schedule/webhook,
// such as "direct_manager", "integration_owner", etc.
func makeNullEntity(id string) *workflowRulesApprovalFlowStepApprovalNotifiedModel {
	return &workflowRulesApprovalFlowStepApprovalNotifiedModel{
		Type:     types.StringValue(id),
		User:     types.ObjectNull((&utils.IdEmailModel{}).AttributeTypes()),
		Group:    types.ObjectNull((&utils.IdNameModel{}).AttributeTypes()),
		Schedule: types.ObjectNull((&utils.IdNameModel{}).AttributeTypes()),
		Webhook:  types.ObjectNull((&utils.IdNameModel{}).AttributeTypes()),
		Channel:  types.ObjectNull((&utils.IdentityOnlyModel{}).AttributeTypes()),
	}
}

func TestReconcileEntityOrder_ReordersShuffledEntities(t *testing.T) {
	// Simulate the exact bug: plan has entities [A, B, C] but the API
	// returns them as [B, C, A]. Without reconciliation, Terraform would
	// see a different group.id at each index and raise
	// "Provider produced inconsistent result after apply".
	groupA := "bf2fd6cc-a27a-4ac0-b54a-cd3a38d07382"
	groupB := "c2ece26b-489b-4247-812d-89f968cb301c"
	groupC := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"

	planRules := []*workflowRulesModel{
		{
			SortOrder:     types.NumberValue(big.NewFloat(0)),
			UnderDuration: types.NumberValue(big.NewFloat(3600)),
			AnySchedule:   types.BoolValue(true),
			ApprovalFlow: &workflowRulesApprovalFlowModel{
				Steps: []*workflowRulesApprovalFlowStepModel{
					{
						SortOrder: types.NumberValue(big.NewFloat(0)),
						Operator:  types.StringValue("and"),
						ApprovalEntities: []*workflowRulesApprovalFlowStepApprovalNotifiedModel{
							makeGroupEntity(t, groupA, "Group A"),
							makeGroupEntity(t, groupB, "Group B"),
							makeGroupEntity(t, groupC, "Group C"),
						},
					},
				},
			},
		},
	}

	// API returns the same entities but in a different order: [B, C, A]
	resultRules := []*workflowRulesModel{
		{
			SortOrder:     types.NumberValue(big.NewFloat(0)),
			UnderDuration: types.NumberValue(big.NewFloat(3600)),
			AnySchedule:   types.BoolValue(true),
			ApprovalFlow: &workflowRulesApprovalFlowModel{
				Steps: []*workflowRulesApprovalFlowStepModel{
					{
						SortOrder: types.NumberValue(big.NewFloat(0)),
						Operator:  types.StringValue("and"),
						ApprovalEntities: []*workflowRulesApprovalFlowStepApprovalNotifiedModel{
							makeGroupEntity(t, groupB, "Group B"),
							makeGroupEntity(t, groupC, "Group C"),
							makeGroupEntity(t, groupA, "Group A"),
						},
					},
				},
			},
		},
	}

	reconcileEntityOrder(planRules, resultRules)

	got := resultRules[0].ApprovalFlow.Steps[0].ApprovalEntities
	wantIDs := []string{groupA, groupB, groupC}

	if len(got) != len(wantIDs) {
		t.Fatalf("expected %d entities, got %d", len(wantIDs), len(got))
	}

	for i, wantID := range wantIDs {
		gotKey := entitySortKey(got[i])
		wantKey := "directory_group:" + wantID
		if gotKey != wantKey {
			t.Errorf("entity[%d]: got key %q, want %q", i, gotKey, wantKey)
		}
	}
}

func TestReconcileEntityOrder_MixedEntityTypes(t *testing.T) {
	// Plan has [user, group, group]; API returns [group, group, user]
	userID := "11111111-1111-1111-1111-111111111111"
	groupA := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	groupB := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"

	planRules := []*workflowRulesModel{
		{
			SortOrder:     types.NumberValue(big.NewFloat(0)),
			UnderDuration: types.NumberValue(big.NewFloat(3600)),
			AnySchedule:   types.BoolValue(true),
			ApprovalFlow: &workflowRulesApprovalFlowModel{
				Steps: []*workflowRulesApprovalFlowStepModel{
					{
						SortOrder: types.NumberValue(big.NewFloat(0)),
						Operator:  types.StringValue("and"),
						ApprovalEntities: []*workflowRulesApprovalFlowStepApprovalNotifiedModel{
							makeUserEntity(t, userID, "user@example.com"),
							makeGroupEntity(t, groupA, "Group A"),
							makeGroupEntity(t, groupB, "Group B"),
						},
					},
				},
			},
		},
	}

	resultRules := []*workflowRulesModel{
		{
			SortOrder:     types.NumberValue(big.NewFloat(0)),
			UnderDuration: types.NumberValue(big.NewFloat(3600)),
			AnySchedule:   types.BoolValue(true),
			ApprovalFlow: &workflowRulesApprovalFlowModel{
				Steps: []*workflowRulesApprovalFlowStepModel{
					{
						SortOrder: types.NumberValue(big.NewFloat(0)),
						Operator:  types.StringValue("and"),
						ApprovalEntities: []*workflowRulesApprovalFlowStepApprovalNotifiedModel{
							makeGroupEntity(t, groupA, "Group A"),
							makeGroupEntity(t, groupB, "Group B"),
							makeUserEntity(t, userID, "user@example.com"),
						},
					},
				},
			},
		},
	}

	reconcileEntityOrder(planRules, resultRules)

	got := resultRules[0].ApprovalFlow.Steps[0].ApprovalEntities
	wantKeys := []string{
		"user:" + userID,
		"directory_group:" + groupA,
		"directory_group:" + groupB,
	}

	for i, wantKey := range wantKeys {
		gotKey := entitySortKey(got[i])
		if gotKey != wantKey {
			t.Errorf("entity[%d]: got key %q, want %q", i, gotKey, wantKey)
		}
	}
}

func TestReconcileEntityOrder_NotifiedEntities(t *testing.T) {
	groupA := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	groupB := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"

	planRules := []*workflowRulesModel{
		{
			SortOrder:     types.NumberValue(big.NewFloat(0)),
			UnderDuration: types.NumberValue(big.NewFloat(3600)),
			AnySchedule:   types.BoolValue(true),
			ApprovalFlow: &workflowRulesApprovalFlowModel{
				Steps: []*workflowRulesApprovalFlowStepModel{
					{
						SortOrder: types.NumberValue(big.NewFloat(0)),
						Operator:  types.StringValue("and"),
						NotifiedEntities: []*workflowRulesApprovalFlowStepApprovalNotifiedModel{
							makeGroupEntity(t, groupA, "Group A"),
							makeGroupEntity(t, groupB, "Group B"),
						},
					},
				},
			},
		},
	}

	resultRules := []*workflowRulesModel{
		{
			SortOrder:     types.NumberValue(big.NewFloat(0)),
			UnderDuration: types.NumberValue(big.NewFloat(3600)),
			AnySchedule:   types.BoolValue(true),
			ApprovalFlow: &workflowRulesApprovalFlowModel{
				Steps: []*workflowRulesApprovalFlowStepModel{
					{
						SortOrder: types.NumberValue(big.NewFloat(0)),
						Operator:  types.StringValue("and"),
						NotifiedEntities: []*workflowRulesApprovalFlowStepApprovalNotifiedModel{
							makeGroupEntity(t, groupB, "Group B"),
							makeGroupEntity(t, groupA, "Group A"),
						},
					},
				},
			},
		},
	}

	reconcileEntityOrder(planRules, resultRules)

	got := resultRules[0].ApprovalFlow.Steps[0].NotifiedEntities
	wantKeys := []string{
		"directory_group:" + groupA,
		"directory_group:" + groupB,
	}

	for i, wantKey := range wantKeys {
		gotKey := entitySortKey(got[i])
		if gotKey != wantKey {
			t.Errorf("notified entity[%d]: got key %q, want %q", i, gotKey, wantKey)
		}
	}
}

func TestReconcileEntityOrder_EmptyEntities(t *testing.T) {
	planRules := []*workflowRulesModel{
		{
			SortOrder:     types.NumberValue(big.NewFloat(0)),
			UnderDuration: types.NumberValue(big.NewFloat(3600)),
			AnySchedule:   types.BoolValue(true),
			ApprovalFlow: &workflowRulesApprovalFlowModel{
				Steps: []*workflowRulesApprovalFlowStepModel{
					{
						SortOrder: types.NumberValue(big.NewFloat(0)),
						Operator:  types.StringValue("and"),
					},
				},
			},
		},
	}

	resultRules := []*workflowRulesModel{
		{
			SortOrder:     types.NumberValue(big.NewFloat(0)),
			UnderDuration: types.NumberValue(big.NewFloat(3600)),
			AnySchedule:   types.BoolValue(true),
			ApprovalFlow: &workflowRulesApprovalFlowModel{
				Steps: []*workflowRulesApprovalFlowStepModel{
					{
						SortOrder: types.NumberValue(big.NewFloat(0)),
						Operator:  types.StringValue("and"),
					},
				},
			},
		},
	}

	// Should not panic
	reconcileEntityOrder(planRules, resultRules)

	got := resultRules[0].ApprovalFlow.Steps[0].ApprovalEntities
	if len(got) != 0 {
		t.Errorf("expected 0 entities, got %d", len(got))
	}
}

func TestReconcileEntityOrder_MultipleRulesAndSteps(t *testing.T) {
	groupA := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	groupB := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	groupC := "cccccccc-cccc-cccc-cccc-cccccccccccc"
	groupD := "dddddddd-dddd-dddd-dddd-dddddddddddd"

	planRules := []*workflowRulesModel{
		{
			SortOrder:     types.NumberValue(big.NewFloat(0)),
			UnderDuration: types.NumberValue(big.NewFloat(3600)),
			AnySchedule:   types.BoolValue(true),
			ApprovalFlow: &workflowRulesApprovalFlowModel{
				Steps: []*workflowRulesApprovalFlowStepModel{
					{
						SortOrder: types.NumberValue(big.NewFloat(0)),
						Operator:  types.StringValue("and"),
						ApprovalEntities: []*workflowRulesApprovalFlowStepApprovalNotifiedModel{
							makeGroupEntity(t, groupA, "Group A"),
							makeGroupEntity(t, groupB, "Group B"),
						},
					},
				},
			},
		},
		{
			SortOrder:     types.NumberValue(big.NewFloat(1)),
			UnderDuration: types.NumberValue(big.NewFloat(7200)),
			AnySchedule:   types.BoolValue(true),
			ApprovalFlow: &workflowRulesApprovalFlowModel{
				Steps: []*workflowRulesApprovalFlowStepModel{
					{
						SortOrder: types.NumberValue(big.NewFloat(0)),
						Operator:  types.StringValue("and"),
						ApprovalEntities: []*workflowRulesApprovalFlowStepApprovalNotifiedModel{
							makeGroupEntity(t, groupC, "Group C"),
							makeGroupEntity(t, groupD, "Group D"),
						},
					},
				},
			},
		},
	}

	// API returns entities reversed within each step
	resultRules := []*workflowRulesModel{
		{
			SortOrder:     types.NumberValue(big.NewFloat(0)),
			UnderDuration: types.NumberValue(big.NewFloat(3600)),
			AnySchedule:   types.BoolValue(true),
			ApprovalFlow: &workflowRulesApprovalFlowModel{
				Steps: []*workflowRulesApprovalFlowStepModel{
					{
						SortOrder: types.NumberValue(big.NewFloat(0)),
						Operator:  types.StringValue("and"),
						ApprovalEntities: []*workflowRulesApprovalFlowStepApprovalNotifiedModel{
							makeGroupEntity(t, groupB, "Group B"),
							makeGroupEntity(t, groupA, "Group A"),
						},
					},
				},
			},
		},
		{
			SortOrder:     types.NumberValue(big.NewFloat(1)),
			UnderDuration: types.NumberValue(big.NewFloat(7200)),
			AnySchedule:   types.BoolValue(true),
			ApprovalFlow: &workflowRulesApprovalFlowModel{
				Steps: []*workflowRulesApprovalFlowStepModel{
					{
						SortOrder: types.NumberValue(big.NewFloat(0)),
						Operator:  types.StringValue("and"),
						ApprovalEntities: []*workflowRulesApprovalFlowStepApprovalNotifiedModel{
							makeGroupEntity(t, groupD, "Group D"),
							makeGroupEntity(t, groupC, "Group C"),
						},
					},
				},
			},
		},
	}

	reconcileEntityOrder(planRules, resultRules)

	// Rule 0, Step 0
	got0 := resultRules[0].ApprovalFlow.Steps[0].ApprovalEntities
	if entitySortKey(got0[0]) != "directory_group:"+groupA {
		t.Errorf("rule[0] entity[0]: got %q, want group A", entitySortKey(got0[0]))
	}
	if entitySortKey(got0[1]) != "directory_group:"+groupB {
		t.Errorf("rule[0] entity[1]: got %q, want group B", entitySortKey(got0[1]))
	}

	// Rule 1, Step 0
	got1 := resultRules[1].ApprovalFlow.Steps[0].ApprovalEntities
	if entitySortKey(got1[0]) != "directory_group:"+groupC {
		t.Errorf("rule[1] entity[0]: got %q, want group C", entitySortKey(got1[0]))
	}
	if entitySortKey(got1[1]) != "directory_group:"+groupD {
		t.Errorf("rule[1] entity[1]: got %q, want group D", entitySortKey(got1[1]))
	}
}

func TestReconcileEntityOrder_MismatchedRuleStepOrder(t *testing.T) {
	// Plan defines rules in HCL order [sort_order=1, sort_order=0], but
	// converterWorkflow sorts the result by sort_order [0, 1].
	// reconcileEntityOrder must match by sort_order, not slice index.
	groupA := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	groupB := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	groupC := "cccccccc-cccc-cccc-cccc-cccccccccccc"
	groupD := "dddddddd-dddd-dddd-dddd-dddddddddddd"

	// Plan: rule sort_order=1 first, sort_order=0 second (HCL order)
	planRules := []*workflowRulesModel{
		{
			SortOrder:     types.NumberValue(big.NewFloat(1)),
			UnderDuration: types.NumberValue(big.NewFloat(7200)),
			AnySchedule:   types.BoolValue(true),
			ApprovalFlow: &workflowRulesApprovalFlowModel{
				Steps: []*workflowRulesApprovalFlowStepModel{
					{
						SortOrder: types.NumberValue(big.NewFloat(0)),
						Operator:  types.StringValue("and"),
						ApprovalEntities: []*workflowRulesApprovalFlowStepApprovalNotifiedModel{
							makeGroupEntity(t, groupC, "Group C"),
							makeGroupEntity(t, groupD, "Group D"),
						},
					},
				},
			},
		},
		{
			SortOrder:     types.NumberValue(big.NewFloat(0)),
			UnderDuration: types.NumberValue(big.NewFloat(3600)),
			AnySchedule:   types.BoolValue(true),
			ApprovalFlow: &workflowRulesApprovalFlowModel{
				Steps: []*workflowRulesApprovalFlowStepModel{
					{
						SortOrder: types.NumberValue(big.NewFloat(0)),
						Operator:  types.StringValue("and"),
						ApprovalEntities: []*workflowRulesApprovalFlowStepApprovalNotifiedModel{
							makeGroupEntity(t, groupA, "Group A"),
							makeGroupEntity(t, groupB, "Group B"),
						},
					},
				},
			},
		},
	}

	// Result: sorted by sort_order [0, 1], entities shuffled within
	resultRules := []*workflowRulesModel{
		{
			SortOrder:     types.NumberValue(big.NewFloat(0)),
			UnderDuration: types.NumberValue(big.NewFloat(3600)),
			AnySchedule:   types.BoolValue(true),
			ApprovalFlow: &workflowRulesApprovalFlowModel{
				Steps: []*workflowRulesApprovalFlowStepModel{
					{
						SortOrder: types.NumberValue(big.NewFloat(0)),
						Operator:  types.StringValue("and"),
						ApprovalEntities: []*workflowRulesApprovalFlowStepApprovalNotifiedModel{
							makeGroupEntity(t, groupB, "Group B"),
							makeGroupEntity(t, groupA, "Group A"),
						},
					},
				},
			},
		},
		{
			SortOrder:     types.NumberValue(big.NewFloat(1)),
			UnderDuration: types.NumberValue(big.NewFloat(7200)),
			AnySchedule:   types.BoolValue(true),
			ApprovalFlow: &workflowRulesApprovalFlowModel{
				Steps: []*workflowRulesApprovalFlowStepModel{
					{
						SortOrder: types.NumberValue(big.NewFloat(0)),
						Operator:  types.StringValue("and"),
						ApprovalEntities: []*workflowRulesApprovalFlowStepApprovalNotifiedModel{
							makeGroupEntity(t, groupD, "Group D"),
							makeGroupEntity(t, groupC, "Group C"),
						},
					},
				},
			},
		},
	}

	reconcileEntityOrder(planRules, resultRules)

	// Result rule sort_order=0 should match plan rule sort_order=0 → [A, B]
	got0 := resultRules[0].ApprovalFlow.Steps[0].ApprovalEntities
	if entitySortKey(got0[0]) != "directory_group:"+groupA {
		t.Errorf("rule[sort=0] entity[0]: got %q, want group A", entitySortKey(got0[0]))
	}
	if entitySortKey(got0[1]) != "directory_group:"+groupB {
		t.Errorf("rule[sort=0] entity[1]: got %q, want group B", entitySortKey(got0[1]))
	}

	// Result rule sort_order=1 should match plan rule sort_order=1 → [C, D]
	got1 := resultRules[1].ApprovalFlow.Steps[0].ApprovalEntities
	if entitySortKey(got1[0]) != "directory_group:"+groupC {
		t.Errorf("rule[sort=1] entity[0]: got %q, want group C", entitySortKey(got1[0]))
	}
	if entitySortKey(got1[1]) != "directory_group:"+groupD {
		t.Errorf("rule[sort=1] entity[1]: got %q, want group D", entitySortKey(got1[1]))
	}
}

func TestReconcileEntityOrder_NullEntityTypes(t *testing.T) {
	// Entities like direct_manager and integration_owner have no
	// user/group/schedule, but they have distinct type strings so their
	// keys are unique ("direct_manager:", "integration_owner:").
	groupA := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"

	planRules := []*workflowRulesModel{
		{
			SortOrder:     types.NumberValue(big.NewFloat(0)),
			UnderDuration: types.NumberValue(big.NewFloat(3600)),
			AnySchedule:   types.BoolValue(true),
			ApprovalFlow: &workflowRulesApprovalFlowModel{
				Steps: []*workflowRulesApprovalFlowStepModel{
					{
						SortOrder: types.NumberValue(big.NewFloat(0)),
						Operator:  types.StringValue("and"),
						ApprovalEntities: []*workflowRulesApprovalFlowStepApprovalNotifiedModel{
							makeNullEntity("direct_manager"),
							makeGroupEntity(t, groupA, "Group A"),
							makeNullEntity("integration_owner"),
						},
					},
				},
			},
		},
	}

	// API returns them shuffled
	resultRules := []*workflowRulesModel{
		{
			SortOrder:     types.NumberValue(big.NewFloat(0)),
			UnderDuration: types.NumberValue(big.NewFloat(3600)),
			AnySchedule:   types.BoolValue(true),
			ApprovalFlow: &workflowRulesApprovalFlowModel{
				Steps: []*workflowRulesApprovalFlowStepModel{
					{
						SortOrder: types.NumberValue(big.NewFloat(0)),
						Operator:  types.StringValue("and"),
						ApprovalEntities: []*workflowRulesApprovalFlowStepApprovalNotifiedModel{
							makeNullEntity("integration_owner"),
							makeNullEntity("direct_manager"),
							makeGroupEntity(t, groupA, "Group A"),
						},
					},
				},
			},
		},
	}

	reconcileEntityOrder(planRules, resultRules)

	got := resultRules[0].ApprovalFlow.Steps[0].ApprovalEntities
	wantKeys := []string{
		"direct_manager:",
		"directory_group:" + groupA,
		"integration_owner:",
	}

	if len(got) != len(wantKeys) {
		t.Fatalf("expected %d entities, got %d", len(wantKeys), len(got))
	}
	for i, wantKey := range wantKeys {
		gotKey := entitySortKey(got[i])
		if gotKey != wantKey {
			t.Errorf("entity[%d]: got key %q, want %q", i, gotKey, wantKey)
		}
	}
}

func TestReconcileEntityOrder_DuplicateNullEntityTypes(t *testing.T) {
	// Edge case: two entities with the same null-entity type produce
	// identical keys ("direct_manager:"). The reorder must not lose
	// any entries even when keys collide.
	planRules := []*workflowRulesModel{
		{
			SortOrder:     types.NumberValue(big.NewFloat(0)),
			UnderDuration: types.NumberValue(big.NewFloat(3600)),
			AnySchedule:   types.BoolValue(true),
			ApprovalFlow: &workflowRulesApprovalFlowModel{
				Steps: []*workflowRulesApprovalFlowStepModel{
					{
						SortOrder: types.NumberValue(big.NewFloat(0)),
						Operator:  types.StringValue("and"),
						ApprovalEntities: []*workflowRulesApprovalFlowStepApprovalNotifiedModel{
							makeNullEntity("direct_manager"),
							makeNullEntity("direct_manager"),
						},
					},
				},
			},
		},
	}

	resultRules := []*workflowRulesModel{
		{
			SortOrder:     types.NumberValue(big.NewFloat(0)),
			UnderDuration: types.NumberValue(big.NewFloat(3600)),
			AnySchedule:   types.BoolValue(true),
			ApprovalFlow: &workflowRulesApprovalFlowModel{
				Steps: []*workflowRulesApprovalFlowStepModel{
					{
						SortOrder: types.NumberValue(big.NewFloat(0)),
						Operator:  types.StringValue("and"),
						ApprovalEntities: []*workflowRulesApprovalFlowStepApprovalNotifiedModel{
							makeNullEntity("direct_manager"),
							makeNullEntity("direct_manager"),
						},
					},
				},
			},
		},
	}

	// Tag result entities so we can distinguish them after reorder.
	resultA := resultRules[0].ApprovalFlow.Steps[0].ApprovalEntities[0]
	resultB := resultRules[0].ApprovalFlow.Steps[0].ApprovalEntities[1]

	reconcileEntityOrder(planRules, resultRules)

	got := resultRules[0].ApprovalFlow.Steps[0].ApprovalEntities
	if len(got) != 2 {
		t.Fatalf("expected 2 entities, got %d (duplicate key caused entry loss)", len(got))
	}

	// Both original result entities must be preserved, not the same pointer twice.
	if got[0] == got[1] {
		t.Error("both result slots point to the same entity; one was lost due to duplicate key")
	}
	// The two original pointers must both appear in the output.
	gotSet := map[*workflowRulesApprovalFlowStepApprovalNotifiedModel]bool{got[0]: true, got[1]: true}
	if !gotSet[resultA] || !gotSet[resultB] {
		t.Error("original result entities were not preserved")
	}
}

func TestReconcileEntityOrder_ExtraResultEntities(t *testing.T) {
	// The API may return entities that weren't in the plan (e.g. added
	// server-side). These extras must be appended after the reordered
	// plan entities, in their original API response order.
	groupA := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	groupB := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	extraUser := "11111111-1111-1111-1111-111111111111"
	extraGroup := "22222222-2222-2222-2222-222222222222"

	planRules := []*workflowRulesModel{
		{
			SortOrder:     types.NumberValue(big.NewFloat(0)),
			UnderDuration: types.NumberValue(big.NewFloat(3600)),
			AnySchedule:   types.BoolValue(true),
			ApprovalFlow: &workflowRulesApprovalFlowModel{
				Steps: []*workflowRulesApprovalFlowStepModel{
					{
						SortOrder: types.NumberValue(big.NewFloat(0)),
						Operator:  types.StringValue("and"),
						ApprovalEntities: []*workflowRulesApprovalFlowStepApprovalNotifiedModel{
							makeGroupEntity(t, groupA, "Group A"),
							makeGroupEntity(t, groupB, "Group B"),
						},
					},
				},
			},
		},
	}

	// API returns plan entities shuffled, plus two extras
	resultRules := []*workflowRulesModel{
		{
			SortOrder:     types.NumberValue(big.NewFloat(0)),
			UnderDuration: types.NumberValue(big.NewFloat(3600)),
			AnySchedule:   types.BoolValue(true),
			ApprovalFlow: &workflowRulesApprovalFlowModel{
				Steps: []*workflowRulesApprovalFlowStepModel{
					{
						SortOrder: types.NumberValue(big.NewFloat(0)),
						Operator:  types.StringValue("and"),
						ApprovalEntities: []*workflowRulesApprovalFlowStepApprovalNotifiedModel{
							makeUserEntity(t, extraUser, "extra@example.com"),
							makeGroupEntity(t, groupB, "Group B"),
							makeGroupEntity(t, extraGroup, "Extra Group"),
							makeGroupEntity(t, groupA, "Group A"),
						},
					},
				},
			},
		},
	}

	reconcileEntityOrder(planRules, resultRules)

	got := resultRules[0].ApprovalFlow.Steps[0].ApprovalEntities

	// Plan entities first in plan order, then extras in original API order
	wantKeys := []string{
		"directory_group:" + groupA,
		"directory_group:" + groupB,
		"user:" + extraUser,
		"directory_group:" + extraGroup,
	}

	if len(got) != len(wantKeys) {
		t.Fatalf("expected %d entities, got %d", len(wantKeys), len(got))
	}
	for i, wantKey := range wantKeys {
		gotKey := entitySortKey(got[i])
		if gotKey != wantKey {
			t.Errorf("entity[%d]: got key %q, want %q", i, gotKey, wantKey)
		}
	}
}

func TestReconcileEntityOrder_TypeCasingMismatch(t *testing.T) {
	// User config may use "webhook" while the API returns "Webhook".
	// entitySortKey normalizes to lowercase so these still match.
	webhookID := "aaaa1111-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	groupID := "bbbb2222-bbbb-bbbb-bbbb-bbbbbbbbbbbb"

	makeWebhookWithType := func(typeName, id, name string) *workflowRulesApprovalFlowStepApprovalNotifiedModel {
		ctx := context.Background()
		v := utils.IdNameModel{ID: types.StringValue(id), Name: types.StringValue(name)}
		vObj, diags := v.AsObjectValue(ctx)
		if diags.HasError() {
			t.Fatalf("failed to create webhook object: %v", diags.Errors())
		}
		return &workflowRulesApprovalFlowStepApprovalNotifiedModel{
			Type:     types.StringValue(typeName),
			User:     types.ObjectNull((&utils.IdEmailModel{}).AttributeTypes()),
			Group:    types.ObjectNull((&utils.IdNameModel{}).AttributeTypes()),
			Schedule: types.ObjectNull((&utils.IdNameModel{}).AttributeTypes()),
			Webhook:  vObj,
		}
	}

	planRules := []*workflowRulesModel{
		{
			SortOrder:     types.NumberValue(big.NewFloat(0)),
			UnderDuration: types.NumberValue(big.NewFloat(3600)),
			AnySchedule:   types.BoolValue(true),
			ApprovalFlow: &workflowRulesApprovalFlowModel{
				Steps: []*workflowRulesApprovalFlowStepModel{
					{
						SortOrder: types.NumberValue(big.NewFloat(0)),
						Operator:  types.StringValue("and"),
						ApprovalEntities: []*workflowRulesApprovalFlowStepApprovalNotifiedModel{
							makeWebhookWithType("webhook", webhookID, "My Hook"),
							makeGroupEntity(t, groupID, "Group"),
						},
					},
				},
			},
		},
	}

	// API returns "Webhook" (capitalized) and in reversed order
	resultRules := []*workflowRulesModel{
		{
			SortOrder:     types.NumberValue(big.NewFloat(0)),
			UnderDuration: types.NumberValue(big.NewFloat(3600)),
			AnySchedule:   types.BoolValue(true),
			ApprovalFlow: &workflowRulesApprovalFlowModel{
				Steps: []*workflowRulesApprovalFlowStepModel{
					{
						SortOrder: types.NumberValue(big.NewFloat(0)),
						Operator:  types.StringValue("and"),
						ApprovalEntities: []*workflowRulesApprovalFlowStepApprovalNotifiedModel{
							makeGroupEntity(t, groupID, "Group"),
							makeWebhookWithType("Webhook", webhookID, "My Hook"),
						},
					},
				},
			},
		},
	}

	reconcileEntityOrder(planRules, resultRules)

	got := resultRules[0].ApprovalFlow.Steps[0].ApprovalEntities
	wantKeys := []string{
		"webhook:" + webhookID,
		"directory_group:" + groupID,
	}

	if len(got) != len(wantKeys) {
		t.Fatalf("expected %d entities, got %d", len(wantKeys), len(got))
	}
	for i, wantKey := range wantKeys {
		gotKey := entitySortKey(got[i])
		if gotKey != wantKey {
			t.Errorf("entity[%d]: got key %q, want %q", i, gotKey, wantKey)
		}
	}
}

func TestReconcileEntityOrder_WebhookEntities(t *testing.T) {
	// Webhook entities carry an ID via the Webhook object field.
	// Verify they are matched and reordered correctly alongside
	// other entity types.
	webhookA := "aaaa1111-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	webhookB := "bbbb2222-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	groupC := "cccc3333-cccc-cccc-cccc-cccccccccccc"

	planRules := []*workflowRulesModel{
		{
			SortOrder:     types.NumberValue(big.NewFloat(0)),
			UnderDuration: types.NumberValue(big.NewFloat(3600)),
			AnySchedule:   types.BoolValue(true),
			ApprovalFlow: &workflowRulesApprovalFlowModel{
				Steps: []*workflowRulesApprovalFlowStepModel{
					{
						SortOrder: types.NumberValue(big.NewFloat(0)),
						Operator:  types.StringValue("and"),
						ApprovalEntities: []*workflowRulesApprovalFlowStepApprovalNotifiedModel{
							makeWebhookEntity(t, webhookA, "Webhook A"),
							makeGroupEntity(t, groupC, "Group C"),
							makeWebhookEntity(t, webhookB, "Webhook B"),
						},
					},
				},
			},
		},
	}

	// API returns them in a different order
	resultRules := []*workflowRulesModel{
		{
			SortOrder:     types.NumberValue(big.NewFloat(0)),
			UnderDuration: types.NumberValue(big.NewFloat(3600)),
			AnySchedule:   types.BoolValue(true),
			ApprovalFlow: &workflowRulesApprovalFlowModel{
				Steps: []*workflowRulesApprovalFlowStepModel{
					{
						SortOrder: types.NumberValue(big.NewFloat(0)),
						Operator:  types.StringValue("and"),
						ApprovalEntities: []*workflowRulesApprovalFlowStepApprovalNotifiedModel{
							makeGroupEntity(t, groupC, "Group C"),
							makeWebhookEntity(t, webhookB, "Webhook B"),
							makeWebhookEntity(t, webhookA, "Webhook A"),
						},
					},
				},
			},
		},
	}

	reconcileEntityOrder(planRules, resultRules)

	got := resultRules[0].ApprovalFlow.Steps[0].ApprovalEntities
	wantKeys := []string{
		"webhook:" + webhookA,
		"directory_group:" + groupC,
		"webhook:" + webhookB,
	}

	if len(got) != len(wantKeys) {
		t.Fatalf("expected %d entities, got %d", len(wantKeys), len(got))
	}
	for i, wantKey := range wantKeys {
		gotKey := entitySortKey(got[i])
		if gotKey != wantKey {
			t.Errorf("entity[%d]: got key %q, want %q", i, gotKey, wantKey)
		}
	}
}

// makeChannelEntity creates a Slack or Teams channel approval/notified entity.
// entityType should be "SlackChannel" or "TeamsChannel".
func makeChannelEntity(t *testing.T, entityType, channelID string) *workflowRulesApprovalFlowStepApprovalNotifiedModel {
	t.Helper()
	ctx := context.Background()

	v := utils.IdentityOnlyModel{
		Id: types.StringValue(channelID),
	}
	vObj, diags := v.AsObjectValue(ctx)
	if diags.HasError() {
		t.Fatalf("failed to create channel object: %v", diags.Errors())
	}

	return &workflowRulesApprovalFlowStepApprovalNotifiedModel{
		Type:     types.StringValue(entityType),
		User:     types.ObjectNull((&utils.IdEmailModel{}).AttributeTypes()),
		Group:    types.ObjectNull((&utils.IdNameModel{}).AttributeTypes()),
		Schedule: types.ObjectNull((&utils.IdNameModel{}).AttributeTypes()),
		Webhook:  types.ObjectNull((&utils.IdNameModel{}).AttributeTypes()),
		Channel:  vObj,
	}
}

// TestGetWorkflowsRules_SlackChannelApprovalEntityID verifies that a SlackChannel
// approval entity sends the correct channel ID to the API (no surrounding quotes,
// no empty ID due to wrong field access).
func TestGetWorkflowsRules_SlackChannelApprovalEntityID(t *testing.T) {
	ctx := context.Background()

	const channelID = "C1111111111"

	planRules := []*workflowRulesModel{
		{
			SortOrder:     types.NumberValue(big.NewFloat(0)),
			UnderDuration: types.NumberValue(big.NewFloat(3600)),
			AnySchedule:   types.BoolValue(true),
			ApprovalFlow: workflowRulesApprovalFlowModel{
				Steps: []*workflowRulesApprovalFlowStepModel{
					{
						SortOrder: types.NumberValue(big.NewFloat(0)),
						Operator:  types.StringValue("and"),
						ApprovalEntities: []*workflowRulesApprovalFlowStepApprovalNotifiedModel{
							makeChannelEntity(t, "SlackChannel", channelID),
						},
					},
				},
			},
		},
	}

	rules, diags := getWorkflowsRules(ctx, planRules)
	if diags.HasError() {
		t.Fatalf("getWorkflowsRules returned errors: %s", diags.Errors())
	}

	approval := rules[0].ApprovalFlow.Steps[0].ApprovalEntities
	if len(approval) != 1 {
		t.Fatalf("expected 1 approval entity, got %d", len(approval))
	}

	// Marshal to JSON to inspect the channel ID
	b, err := approval[0].MarshalJSON()
	if err != nil {
		t.Fatalf("failed to marshal approval entity: %v", err)
	}
	jsonStr := string(b)
	if !containsChannelID(jsonStr, channelID) {
		t.Errorf("approval entity JSON does not contain channel ID %q: %s", channelID, jsonStr)
	}
}

// TestGetWorkflowsRules_TeamsChannelApprovalEntityID verifies that a TeamsChannel
// approval entity sends the correct channel ID to the API.
func TestGetWorkflowsRules_TeamsChannelApprovalEntityID(t *testing.T) {
	ctx := context.Background()

	const channelID = "19:meeting-channel-id@thread.tacv2"

	planRules := []*workflowRulesModel{
		{
			SortOrder:     types.NumberValue(big.NewFloat(0)),
			UnderDuration: types.NumberValue(big.NewFloat(3600)),
			AnySchedule:   types.BoolValue(true),
			ApprovalFlow: workflowRulesApprovalFlowModel{
				Steps: []*workflowRulesApprovalFlowStepModel{
					{
						SortOrder: types.NumberValue(big.NewFloat(0)),
						Operator:  types.StringValue("and"),
						ApprovalEntities: []*workflowRulesApprovalFlowStepApprovalNotifiedModel{
							makeChannelEntity(t, "TeamsChannel", channelID),
						},
					},
				},
			},
		},
	}

	rules, diags := getWorkflowsRules(ctx, planRules)
	if diags.HasError() {
		t.Fatalf("getWorkflowsRules returned errors: %s", diags.Errors())
	}

	approval := rules[0].ApprovalFlow.Steps[0].ApprovalEntities
	if len(approval) != 1 {
		t.Fatalf("expected 1 approval entity, got %d", len(approval))
	}

	b, err := approval[0].MarshalJSON()
	if err != nil {
		t.Fatalf("failed to marshal approval entity: %v", err)
	}
	jsonStr := string(b)
	if !containsChannelID(jsonStr, channelID) {
		t.Errorf("approval entity JSON does not contain channel ID %q: %s", channelID, jsonStr)
	}
}

// TestGetWorkflowsRules_SlackChannelNotifiedEntityID verifies that a SlackChannel
// notified entity sends the correct channel ID to the API. This specifically tests
// the bug fix where entity.Schedule was mistakenly used instead of entity.Channel.
func TestGetWorkflowsRules_SlackChannelNotifiedEntityID(t *testing.T) {
	ctx := context.Background()

	const channelID = "C2222222222"

	planRules := []*workflowRulesModel{
		{
			SortOrder:     types.NumberValue(big.NewFloat(0)),
			UnderDuration: types.NumberValue(big.NewFloat(3600)),
			AnySchedule:   types.BoolValue(true),
			ApprovalFlow: workflowRulesApprovalFlowModel{
				Steps: []*workflowRulesApprovalFlowStepModel{
					{
						SortOrder: types.NumberValue(big.NewFloat(0)),
						Operator:  types.StringValue("and"),
						NotifiedEntities: []*workflowRulesApprovalFlowStepApprovalNotifiedModel{
							makeChannelEntity(t, "SlackChannel", channelID),
						},
						ApprovalEntities: []*workflowRulesApprovalFlowStepApprovalNotifiedModel{
							makeNullEntity("Automatic"),
						},
					},
				},
			},
		},
	}

	rules, diags := getWorkflowsRules(ctx, planRules)
	if diags.HasError() {
		t.Fatalf("getWorkflowsRules returned errors: %s", diags.Errors())
	}

	notified := rules[0].ApprovalFlow.Steps[0].NotifiedEntities
	if len(notified) != 1 {
		t.Fatalf("expected 1 notified entity, got %d", len(notified))
	}

	b, err := notified[0].MarshalJSON()
	if err != nil {
		t.Fatalf("failed to marshal notified entity: %v", err)
	}
	jsonStr := string(b)
	if !containsChannelID(jsonStr, channelID) {
		t.Errorf("notified entity JSON does not contain channel ID %q: %s", channelID, jsonStr)
	}
}

// TestGetWorkflowsRules_TeamsChannelNotifiedEntityID verifies that a TeamsChannel
// notified entity sends the correct channel ID to the API. This specifically tests
// the bug fix where entity.Schedule was mistakenly used instead of entity.Channel.
func TestGetWorkflowsRules_TeamsChannelNotifiedEntityID(t *testing.T) {
	ctx := context.Background()

	const channelID = "19:teams-notified-channel@thread.tacv2"

	planRules := []*workflowRulesModel{
		{
			SortOrder:     types.NumberValue(big.NewFloat(0)),
			UnderDuration: types.NumberValue(big.NewFloat(3600)),
			AnySchedule:   types.BoolValue(true),
			ApprovalFlow: workflowRulesApprovalFlowModel{
				Steps: []*workflowRulesApprovalFlowStepModel{
					{
						SortOrder: types.NumberValue(big.NewFloat(0)),
						Operator:  types.StringValue("and"),
						NotifiedEntities: []*workflowRulesApprovalFlowStepApprovalNotifiedModel{
							makeChannelEntity(t, "TeamsChannel", channelID),
						},
						ApprovalEntities: []*workflowRulesApprovalFlowStepApprovalNotifiedModel{
							makeNullEntity("Automatic"),
						},
					},
				},
			},
		},
	}

	rules, diags := getWorkflowsRules(ctx, planRules)
	if diags.HasError() {
		t.Fatalf("getWorkflowsRules returned errors: %s", diags.Errors())
	}

	notified := rules[0].ApprovalFlow.Steps[0].NotifiedEntities
	if len(notified) != 1 {
		t.Fatalf("expected 1 notified entity, got %d", len(notified))
	}

	b, err := notified[0].MarshalJSON()
	if err != nil {
		t.Fatalf("failed to marshal notified entity: %v", err)
	}
	jsonStr := string(b)
	if !containsChannelID(jsonStr, channelID) {
		t.Errorf("notified entity JSON does not contain channel ID %q: %s", channelID, jsonStr)
	}
}

// TestGetWorkflowsRules_MixedChannelEntities verifies that a step with both
// approval and notified channel entities is handled correctly.
func TestGetWorkflowsRules_MixedChannelEntities(t *testing.T) {
	ctx := context.Background()

	const (
		slackApprovalID = "C3333333333"
		teamsNotifiedID = "19:teams-notified@thread.tacv2"
	)

	planRules := []*workflowRulesModel{
		{
			SortOrder:     types.NumberValue(big.NewFloat(0)),
			UnderDuration: types.NumberValue(big.NewFloat(3600)),
			AnySchedule:   types.BoolValue(true),
			ApprovalFlow: workflowRulesApprovalFlowModel{
				Steps: []*workflowRulesApprovalFlowStepModel{
					{
						SortOrder: types.NumberValue(big.NewFloat(0)),
						Operator:  types.StringValue("and"),
						NotifiedEntities: []*workflowRulesApprovalFlowStepApprovalNotifiedModel{
							makeChannelEntity(t, "TeamsChannel", teamsNotifiedID),
						},
						ApprovalEntities: []*workflowRulesApprovalFlowStepApprovalNotifiedModel{
							makeChannelEntity(t, "SlackChannel", slackApprovalID),
						},
					},
				},
			},
		},
	}

	rules, diags := getWorkflowsRules(ctx, planRules)
	if diags.HasError() {
		t.Fatalf("getWorkflowsRules returned errors: %s", diags.Errors())
	}

	step := rules[0].ApprovalFlow.Steps[0]

	// Check approval entity
	approvalJSON, _ := step.ApprovalEntities[0].MarshalJSON()
	if !containsChannelID(string(approvalJSON), slackApprovalID) {
		t.Errorf("approval entity missing SlackChannel ID %q: %s", slackApprovalID, string(approvalJSON))
	}

	// Check notified entity
	notifiedJSON, _ := step.NotifiedEntities[0].MarshalJSON()
	if !containsChannelID(string(notifiedJSON), teamsNotifiedID) {
		t.Errorf("notified entity missing TeamsChannel ID %q: %s", teamsNotifiedID, string(notifiedJSON))
	}
}

// containsChannelID checks whether a JSON string contains a given channel ID value.
func containsChannelID(jsonStr, channelID string) bool {
	// The ID appears as a JSON string value: "id":"<channelID>"
	return len(jsonStr) > 0 && (contains(jsonStr, `"id":"`+channelID+`"`) ||
		contains(jsonStr, `"Id":"`+channelID+`"`))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestEntitySortKey_ChannelEntities verifies that entitySortKey correctly
// extracts the channel ID for SlackChannel and TeamsChannel entities.
func TestEntitySortKey_ChannelEntities(t *testing.T) {
	const (
		slackChannelID = "C4444444444"
		teamsChannelID = "19:teams-key-test@thread.tacv2"
	)

	slackEntity := makeChannelEntity(t, "SlackChannel", slackChannelID)
	teamsEntity := makeChannelEntity(t, "TeamsChannel", teamsChannelID)

	gotSlack := entitySortKey(slackEntity)
	wantSlack := "slackchannel:" + slackChannelID
	if gotSlack != wantSlack {
		t.Errorf("SlackChannel sort key = %q, want %q", gotSlack, wantSlack)
	}

	gotTeams := entitySortKey(teamsEntity)
	wantTeams := "teamschannel:" + teamsChannelID
	if gotTeams != wantTeams {
		t.Errorf("TeamsChannel sort key = %q, want %q", gotTeams, wantTeams)
	}
}

// TestReconcileEntityOrder_ChannelEntities verifies that Slack and Teams channel
// entities are correctly reordered when the API returns them in a different order.
func TestReconcileEntityOrder_ChannelEntities(t *testing.T) {
	slackA := "C5555555555"
	slackB := "C6666666666"
	teamsC := "19:teams-reorder-c@thread.tacv2"

	planRules := []*workflowRulesModel{
		{
			SortOrder:     types.NumberValue(big.NewFloat(0)),
			UnderDuration: types.NumberValue(big.NewFloat(3600)),
			AnySchedule:   types.BoolValue(true),
			ApprovalFlow: workflowRulesApprovalFlowModel{
				Steps: []*workflowRulesApprovalFlowStepModel{
					{
						SortOrder: types.NumberValue(big.NewFloat(0)),
						Operator:  types.StringValue("and"),
						ApprovalEntities: []*workflowRulesApprovalFlowStepApprovalNotifiedModel{
							makeChannelEntity(t, "SlackChannel", slackA),
							makeChannelEntity(t, "TeamsChannel", teamsC),
							makeChannelEntity(t, "SlackChannel", slackB),
						},
					},
				},
			},
		},
	}

	// API returns entities in a different order: [slackB, teamsC, slackA]
	resultRules := []*workflowRulesModel{
		{
			SortOrder:     types.NumberValue(big.NewFloat(0)),
			UnderDuration: types.NumberValue(big.NewFloat(3600)),
			AnySchedule:   types.BoolValue(true),
			ApprovalFlow: workflowRulesApprovalFlowModel{
				Steps: []*workflowRulesApprovalFlowStepModel{
					{
						SortOrder: types.NumberValue(big.NewFloat(0)),
						Operator:  types.StringValue("and"),
						ApprovalEntities: []*workflowRulesApprovalFlowStepApprovalNotifiedModel{
							makeChannelEntity(t, "SlackChannel", slackB),
							makeChannelEntity(t, "TeamsChannel", teamsC),
							makeChannelEntity(t, "SlackChannel", slackA),
						},
					},
				},
			},
		},
	}

	reconcileEntityOrder(planRules, resultRules)

	got := resultRules[0].ApprovalFlow.Steps[0].ApprovalEntities
	wantKeys := []string{
		"slackchannel:" + slackA,
		"teamschannel:" + teamsC,
		"slackchannel:" + slackB,
	}

	if len(got) != len(wantKeys) {
		t.Fatalf("expected %d entities, got %d", len(wantKeys), len(got))
	}
	for i, wantKey := range wantKeys {
		gotKey := entitySortKey(got[i])
		if gotKey != wantKey {
			t.Errorf("entity[%d]: got key %q, want %q", i, gotKey, wantKey)
		}
	}
}

// TestReconcileEntityOrder_ChannelAndGroupMixed verifies that channel and
// group entities interleaved are reconciled correctly.
func TestReconcileEntityOrder_ChannelAndGroupMixed(t *testing.T) {
	slackID := "C7777777777"
	groupID := "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee"

	planRules := []*workflowRulesModel{
		{
			SortOrder:     types.NumberValue(big.NewFloat(0)),
			UnderDuration: types.NumberValue(big.NewFloat(3600)),
			AnySchedule:   types.BoolValue(true),
			ApprovalFlow: workflowRulesApprovalFlowModel{
				Steps: []*workflowRulesApprovalFlowStepModel{
					{
						SortOrder: types.NumberValue(big.NewFloat(0)),
						Operator:  types.StringValue("and"),
						NotifiedEntities: []*workflowRulesApprovalFlowStepApprovalNotifiedModel{
							makeChannelEntity(t, "SlackChannel", slackID),
							makeGroupEntity(t, groupID, "My Group"),
						},
					},
				},
			},
		},
	}

	// API returns [group, slack] instead of [slack, group]
	resultRules := []*workflowRulesModel{
		{
			SortOrder:     types.NumberValue(big.NewFloat(0)),
			UnderDuration: types.NumberValue(big.NewFloat(3600)),
			AnySchedule:   types.BoolValue(true),
			ApprovalFlow: workflowRulesApprovalFlowModel{
				Steps: []*workflowRulesApprovalFlowStepModel{
					{
						SortOrder: types.NumberValue(big.NewFloat(0)),
						Operator:  types.StringValue("and"),
						NotifiedEntities: []*workflowRulesApprovalFlowStepApprovalNotifiedModel{
							makeGroupEntity(t, groupID, "My Group"),
							makeChannelEntity(t, "SlackChannel", slackID),
						},
					},
				},
			},
		},
	}

	reconcileEntityOrder(planRules, resultRules)

	got := resultRules[0].ApprovalFlow.Steps[0].NotifiedEntities
	wantKeys := []string{
		"slackchannel:" + slackID,
		"directory_group:" + groupID,
	}

	if len(got) != len(wantKeys) {
		t.Fatalf("expected %d entities, got %d", len(wantKeys), len(got))
	}
	for i, wantKey := range wantKeys {
		gotKey := entitySortKey(got[i])
		if gotKey != wantKey {
			t.Errorf("notified entity[%d]: got key %q, want %q", i, gotKey, wantKey)
		}
	}
}
