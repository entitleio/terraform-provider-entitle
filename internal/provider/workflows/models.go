package workflows

import (
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/entitleio/terraform-provider-entitle/internal/provider/utils"
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

type workflowRulesApprovalFlowStepApprovalNotifiedModel struct {
	Type     types.String `tfsdk:"type" json:"type"`
	User     types.Object `tfsdk:"user" json:"user"`
	Group    types.Object `tfsdk:"group" json:"group"`
	Schedule types.Object `tfsdk:"schedule" json:"schedule"`
}
