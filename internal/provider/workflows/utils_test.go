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
			ApprovalFlow: workflowRulesApprovalFlowModel{
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
							},
							{
								Type:     types.StringValue("group"),
								User:     nullUser,
								Group:    groupObj,
								Schedule: nullSchedule,
							},
							{
								Type:     types.StringValue("schedule"),
								User:     nullUser,
								Group:    nullGroup,
								Schedule: scheduleObj,
							},
						},
						ApprovalEntities: []*workflowRulesApprovalFlowStepApprovalNotifiedModel{
							{
								Type:     types.StringValue("Automatic"),
								User:     nullUser,
								Group:    nullGroup,
								Schedule: nullSchedule,
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
