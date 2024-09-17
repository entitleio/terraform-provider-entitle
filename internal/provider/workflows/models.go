package workflows

import (
	"context"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/utils"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

type workflowRulesModel struct {
	SortOrder     types.Number                   `tfsdk:"sort_order" json:"sortOrder"`
	UnderDuration types.Number                   `tfsdk:"under_duration" json:"underDuration"`
	ApprovalFlow  workflowRulesApprovalFlowModel `tfsdk:"approval_flow" json:"approvalFlow"`
	InGroups      []*utils.IdNameModel           `tfsdk:"in_groups" json:"inGroups"`
	InSchedules   []*utils.IdNameModel           `tfsdk:"in_schedules" json:"inSchedules"`
	AnySchedule   types.Bool                     `tfsdk:"any_schedule" json:"anySchedule"`
}

type workflowRulesApprovalFlowModel struct {
	Steps []*workflowRulesApprovalFlowStepModel `tfsdk:"steps" json:"steps"`
}

type workflowRulesApprovalFlowStepModel struct {
	SortOrder        types.Number                                          `tfsdk:"sort_order"`
	Operator         types.String                                          `tfsdk:"operator"`
	ApprovalEntities []*workflowRulesApprovalFlowStepApprovalNotifiedModel `tfsdk:"approval_entities"`
	NotifiedEntities []*workflowRulesApprovalFlowStepApprovalNotifiedModel `tfsdk:"notified_entities"`
}

type workflowRulesApprovalFlowStepNotifiedEntityModel struct {
	Notified types.String `tfsdk:"notified" json:"notified"`
}

func (m workflowRulesApprovalFlowStepNotifiedEntityModel) attributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"notified": types.StringType,
	}
}

func (m workflowRulesApprovalFlowStepNotifiedEntityModel) AsObjectValue(
	ctx context.Context,
) (basetypes.ObjectValue, diag.Diagnostics) {
	return types.ObjectValueFrom(ctx, m.attributeTypes(), m)
}

type workflowRulesApprovalFlowStepApprovalEntityModel struct {
	Approval types.String `tfsdk:"approval" json:"approval"`
}

func (m workflowRulesApprovalFlowStepApprovalEntityModel) attributeTypes() map[string]attr.Type {
	return map[string]attr.Type{
		"approval": types.StringType,
	}
}

func (m workflowRulesApprovalFlowStepApprovalEntityModel) AsObjectValue(
	ctx context.Context,
) (basetypes.ObjectValue, diag.Diagnostics) {
	return types.ObjectValueFrom(ctx, m.attributeTypes(), m)
}

type workflowRulesApprovalFlowStepApprovalNotifiedModel struct {
	Type     types.String `tfsdk:"type" json:"type"`
	User     types.Object `tfsdk:"user" json:"user"`
	Group    types.Object `tfsdk:"group" json:"group"`
	Schedule types.Object `tfsdk:"schedule" json:"schedule"`
	Value    types.Object `tfsdk:"value" json:"value"`
}
