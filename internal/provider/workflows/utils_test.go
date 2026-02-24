package workflows

import (
	"context"
	"math/big"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/entitleio/terraform-provider-entitle/internal/provider/utils"
)

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
	}
}

// makeNullEntity creates an approval entity without a user/group/schedule,
// such as "direct_manager", "integration_owner", etc.
func makeNullEntity(id string) *workflowRulesApprovalFlowStepApprovalNotifiedModel {
	return &workflowRulesApprovalFlowStepApprovalNotifiedModel{
		Type:     types.StringValue(id),
		User:     types.ObjectNull((&utils.IdEmailModel{}).AttributeTypes()),
		Group:    types.ObjectNull((&utils.IdNameModel{}).AttributeTypes()),
		Schedule: types.ObjectNull((&utils.IdNameModel{}).AttributeTypes()),
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
			ApprovalFlow: workflowRulesApprovalFlowModel{
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
			ApprovalFlow: workflowRulesApprovalFlowModel{
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
			ApprovalFlow: workflowRulesApprovalFlowModel{
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
			ApprovalFlow: workflowRulesApprovalFlowModel{
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
			ApprovalFlow: workflowRulesApprovalFlowModel{
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
			ApprovalFlow: workflowRulesApprovalFlowModel{
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
			ApprovalFlow: workflowRulesApprovalFlowModel{
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
			ApprovalFlow: workflowRulesApprovalFlowModel{
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
			ApprovalFlow: workflowRulesApprovalFlowModel{
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
			ApprovalFlow: workflowRulesApprovalFlowModel{
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
			ApprovalFlow: workflowRulesApprovalFlowModel{
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
			ApprovalFlow: workflowRulesApprovalFlowModel{
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
			ApprovalFlow: workflowRulesApprovalFlowModel{
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
			ApprovalFlow: workflowRulesApprovalFlowModel{
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
			ApprovalFlow: workflowRulesApprovalFlowModel{
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
			ApprovalFlow: workflowRulesApprovalFlowModel{
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
