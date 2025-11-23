package workflows

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"math/big"
	"sort"

	"github.com/entitleio/terraform-provider-entitle/docs"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/utils"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/entitleio/terraform-provider-entitle/internal/client"
	"github.com/entitleio/terraform-provider-entitle/internal/validators"
)

// Ensure that the provider-defined types fully satisfy the framework interfaces.
var _ datasource.DataSource = &WorkflowDataSource{}

// WorkflowDataSource defines the data source implementation for the Terraform utils.
type WorkflowDataSource struct {
	client *client.ClientWithResponses
}

// NewWorkflowDataSource creates a new instance of the WorkflowDataSource.
func NewWorkflowDataSource() datasource.DataSource {
	return &WorkflowDataSource{}
}

// WorkflowDataSourceModel defines the data model for FullWorkflowResultResponseSchema.
type WorkflowDataSourceModel struct {
	Id    types.String          `tfsdk:"id" json:"id"`
	Name  types.String          `tfsdk:"name" json:"name"`
	Rules []*workflowRulesModel `tfsdk:"rules" json:"rules"`
}

// Metadata sets the data source's metadata, such as its type name.
func (d *WorkflowDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_workflow"
}

func (d *WorkflowDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: docs.WorkflowDataSourceMarkdownDescription,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Required:            true,
				MarkdownDescription: "Entitle Workflow identifier in uuid format",
				Description:         "Entitle Workflow identifier in uuid format",
				Validators: []validator.String{
					validators.UUID{},
				},
			},
			"name": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Workflow name",
				Description:         "Workflow name",
			},
			"rules": schema.ListNestedAttribute{
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"any_schedule": schema.BoolAttribute{
							Computed:            true,
							Description:         "Indicates whether this rule applies regardless of scheduling constraints.",
							MarkdownDescription: "Indicates whether this rule applies regardless of scheduling constraints.",
						},
						"sort_order": schema.NumberAttribute{
							Computed:            true,
							Description:         "The order in which the rule is evaluated",
							MarkdownDescription: "The order in which the rule is evaluated",
						},
						"under_duration": schema.NumberAttribute{
							Computed:            true,
							Description:         "Maximum duration this rule is valid for",
							MarkdownDescription: "Maximum duration this rule is valid for",
						},
						"in_groups": schema.ListNestedAttribute{
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"id": schema.StringAttribute{
										Computed:            true,
										Description:         "Group's unique identifier",
										MarkdownDescription: "Group's unique identifier",
									},
									"name": schema.StringAttribute{
										Computed:            true,
										Description:         "Group's name",
										MarkdownDescription: "Group's name",
									},
								},
							},
							Computed:            true,
							Description:         "Groups for which the rule applies",
							MarkdownDescription: "Groups for which the rule applies",
						},
						"in_schedules": schema.ListNestedAttribute{
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"id": schema.StringAttribute{
										Computed:            true,
										Description:         "Schedule's unique identifier",
										MarkdownDescription: "Schedule's unique identifier",
									},
									"name": schema.StringAttribute{
										Computed:            true,
										Description:         "Schedule's name",
										MarkdownDescription: "Schedule's name",
									},
								},
							},
							Computed:            true,
							Description:         "Schedules for which the rule applies",
							MarkdownDescription: "Schedules for which the rule applies",
						},
						"approval_flow": schema.SingleNestedAttribute{
							Attributes: map[string]schema.Attribute{
								"steps": schema.ListNestedAttribute{
									NestedObject: schema.NestedAttributeObject{
										Attributes: map[string]schema.Attribute{
											"operator": schema.StringAttribute{
												Computed:            true,
												Description:         "Approval step operator",
												MarkdownDescription: "Approval step operator",
											},
											"sort_order": schema.NumberAttribute{
												Computed:            true,
												Description:         "Step execution order",
												MarkdownDescription: "Step execution order",
											},
											"notified_entities": schema.ListNestedAttribute{
												NestedObject: schema.NestedAttributeObject{
													Attributes: map[string]schema.Attribute{
														"type": schema.StringAttribute{
															Computed:            true,
															Description:         "Entity type",
															MarkdownDescription: "Entity type",
														},
														"user": schema.SingleNestedAttribute{
															Attributes: map[string]schema.Attribute{
																"id": schema.StringAttribute{
																	Computed:            true,
																	Description:         "Notified user's unique identifier",
																	MarkdownDescription: "Notified user's unique identifier",
																},
																"email": schema.StringAttribute{
																	Computed:            true,
																	Description:         "Notified user's email",
																	MarkdownDescription: "Notified user's email",
																},
															},
															Computed:            true,
															Description:         "Notified user details",
															MarkdownDescription: "Notified user details",
														},
														"group": schema.SingleNestedAttribute{
															Attributes: map[string]schema.Attribute{
																"id": schema.StringAttribute{
																	Computed:            true,
																	Description:         "Notified group's unique identifier",
																	MarkdownDescription: "Notified group's unique identifier",
																},
																"name": schema.StringAttribute{
																	Computed:            true,
																	Description:         "Notified group's name",
																	MarkdownDescription: "Notified group's name",
																},
															},
															Computed:            true,
															Description:         "Notified group details",
															MarkdownDescription: "Notified group details",
														},
														"schedule": schema.SingleNestedAttribute{
															Attributes: map[string]schema.Attribute{
																"id": schema.StringAttribute{
																	Computed:            true,
																	Description:         "Schedule unique identifier",
																	MarkdownDescription: "Schedule unique identifier",
																},
																"name": schema.StringAttribute{
																	Computed:            true,
																	Description:         "Schedule name",
																	MarkdownDescription: "Schedule name",
																},
															},
															Computed:            true,
															Description:         "Notified schedule details",
															MarkdownDescription: "Notified schedule details",
														},
													},
												},
												Computed:            true,
												Description:         "Entities to notify when the step is triggered",
												MarkdownDescription: "Entities to notify when the step is triggered",
											},
											"approval_entities": schema.ListNestedAttribute{
												NestedObject: schema.NestedAttributeObject{
													Attributes: map[string]schema.Attribute{
														"type": schema.StringAttribute{
															Computed:            true,
															Description:         "Approver type",
															MarkdownDescription: "Approver type",
														},
														"user": schema.SingleNestedAttribute{
															Attributes: map[string]schema.Attribute{
																"id": schema.StringAttribute{
																	Computed:            true,
																	Description:         "Approver user's unique identifier",
																	MarkdownDescription: "Approver user's unique identifier",
																},
																"email": schema.StringAttribute{
																	Computed:            true,
																	Description:         "Approver user's email address",
																	MarkdownDescription: "Approver user's email address",
																},
															},
															Computed:            true,
															Description:         "Approver user details",
															MarkdownDescription: "Approver user details",
														},
														"group": schema.SingleNestedAttribute{
															Attributes: map[string]schema.Attribute{
																"id": schema.StringAttribute{
																	Computed:            true,
																	Description:         "Approver group's unique identifier",
																	MarkdownDescription: "Approver group's unique identifier",
																},
																"name": schema.StringAttribute{
																	Computed:            true,
																	Description:         "Approver group's name",
																	MarkdownDescription: "Approver group's name",
																},
															},
															Computed:            true,
															Description:         "Approver group details",
															MarkdownDescription: "Approver group details",
														},
														"schedule": schema.SingleNestedAttribute{
															Attributes: map[string]schema.Attribute{
																"id": schema.StringAttribute{
																	Computed:            true,
																	Description:         "Schedule ID",
																	MarkdownDescription: "Schedule ID",
																},
																"name": schema.StringAttribute{
																	Computed:            true,
																	Description:         "Schedule name",
																	MarkdownDescription: "Schedule name",
																},
															},
															Computed:            true,
															Description:         "Approver schedule details",
															MarkdownDescription: "Approver schedule details",
														},
													},
												},
												Computed:            true,
												Description:         "Entities that must approve the step",
												MarkdownDescription: "Entities that must approve the step",
											},
										},
									},
									Computed:            true,
									Description:         "Ordered steps in the approval process",
									MarkdownDescription: "Ordered steps in the approval process",
								},
							},
							Computed:            true,
							Description:         "Defines the approval process if the rule matches",
							MarkdownDescription: "Defines the approval process if the rule matches",
						},
					},
				},
				Computed:            true,
				Description:         "List of workflow rules that determine how approval is handled",
				MarkdownDescription: "List of workflow rules that determine how approval is handled",
			},
		},
	}
}

// Configure configures the data source with the provider's client.
func (d *WorkflowDataSource) Configure(
	ctx context.Context,
	req datasource.ConfigureRequest,
	resp *datasource.ConfigureResponse,
) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.ClientWithResponses)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *http.Client, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = c
}

// Read reads data from the external source and sets it in Terraform state.
func (d *WorkflowDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data WorkflowDataSourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	uid := uuid.MustParse(data.Id.String())

	workflowResp, err := d.client.WorkflowsShowWithResponse(ctx, uid)
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiConnection.Error(),
			fmt.Sprintf("Unable to get the bundle by the id (%s), got error: %s", uid.String(), err),
		)
		return
	}

	err = utils.HTTPResponseToError(workflowResp.HTTPResponse.StatusCode, workflowResp.Body)
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiResponse.Error(),
			fmt.Sprintf(
				"Failed to get the Workflow by the id (%s), status code: %d, %s",
				uid.String(),
				workflowResp.HTTPResponse.StatusCode,
				err.Error(),
			),
		)
		return
	}

	data, diags := converterWorkflow(ctx, &workflowResp.JSON200.Result)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "read a entitle workflow data source")

	// Save data into Terraform state
	diagsResult := resp.State.Set(ctx, &data)
	resp.Diagnostics.Append(diagsResult...)
	if resp.Diagnostics.HasError() {
		return
	}
}

func converterWorkflow(
	ctx context.Context,
	data *client.FullWorkflowResultResponseSchema,
) (WorkflowDataSourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	if data == nil {
		diags.AddError(
			"No data",
			"Given schema data is nil",
		)

		return WorkflowDataSourceModel{}, diags
	}

	var rules []*workflowRulesModel
	if len(data.Rules) > 0 {
		rules = make([]*workflowRulesModel, 0, len(data.Rules))
		for _, rule := range data.Rules {
			ruleModel := &workflowRulesModel{
				AnySchedule: types.BoolValue(rule.AnySchedule),
				ApprovalFlow: workflowRulesApprovalFlowModel{
					Steps: make([]*workflowRulesApprovalFlowStepModel, 0),
				},
				SortOrder:     types.NumberValue(big.NewFloat(float64(rule.SortOrder))),
				UnderDuration: types.NumberValue(big.NewFloat(float64(rule.UnderDuration))),
			}

			if len(rule.InGroups) > 0 {
				ruleModel.InGroups = make([]*utils.IdNameModel, 0, len(rule.InGroups))

				for _, inGroup := range rule.InGroups {
					ruleModel.InGroups = append(ruleModel.InGroups, &utils.IdNameModel{
						ID:   utils.TrimmedStringValue(inGroup.Id.String()),
						Name: utils.TrimmedStringValue(inGroup.Name),
					})
				}
			}

			if len(rule.InSchedules) > 0 {
				ruleModel.InSchedules = make([]*utils.IdNameModel, 0, len(rule.InSchedules))

				for _, inSchedule := range rule.InSchedules {
					ruleModel.InSchedules = append(ruleModel.InSchedules, &utils.IdNameModel{
						ID:   utils.TrimmedStringValue(inSchedule.Id.String()),
						Name: utils.TrimmedStringValue(inSchedule.Name),
					})
				}
			}

			if len(rule.ApprovalFlow.Steps) > 0 {
				for _, step := range rule.ApprovalFlow.Steps {
					flowStep := &workflowRulesApprovalFlowStepModel{
						ApprovalEntities: make([]*workflowRulesApprovalFlowStepApprovalNotifiedModel, 0, len(step.ApprovalEntities)),
						Operator:         utils.TrimmedStringValue(string(step.Operator)),
						SortOrder:        types.NumberValue(big.NewFloat(float64(step.SortOrder))),
					}

					if len(step.NotifiedEntities) > 0 {
						flowStep.NotifiedEntities = make([]*workflowRulesApprovalFlowStepApprovalNotifiedModel, 0, len(step.NotifiedEntities))

						for _, entity := range step.NotifiedEntities {
							var stepMap = make(map[string]interface{})

							jsonData, err := entity.MarshalJSON()
							if err != nil {
								diags.AddError(
									"Failed to marshal step data",
									err.Error(),
								)

								return WorkflowDataSourceModel{}, diags
							}

							err = json.Unmarshal(jsonData, &stepMap)
							if err != nil {
								diags.AddError(
									"Failed to marshal step data",
									err.Error(),
								)

								return WorkflowDataSourceModel{}, diags
							}

							typeStep, ok := stepMap["type"]
							if !ok {
								continue
							}

							switch fmt.Sprintf("%v", typeStep) {
							case string(client.OnCallIntegrationSchedule):
								val, err := entity.AsApprovalEntityScheduleResponseSchema()
								if err != nil {
									diags.AddError(
										"Failed to convert entity to schedule type",
										err.Error(),
									)

									return WorkflowDataSourceModel{}, diags
								}

								v := utils.IdNameModel{
									ID:   utils.TrimmedStringValue(val.Entity.Id.String()),
									Name: utils.TrimmedStringValue(val.Entity.Name),
								}

								vObj, diagsAs := v.AsObjectValue(ctx)
								if diagsAs.HasError() {
									diags.Append(diagsAs...)
									return WorkflowDataSourceModel{}, diags
								}

								flowStep.NotifiedEntities = append(flowStep.NotifiedEntities, &workflowRulesApprovalFlowStepApprovalNotifiedModel{
									Type:     utils.TrimmedStringValue(string(val.Type)),
									Schedule: vObj,
									User:     types.ObjectNull((&utils.IdEmailModel{}).AttributeTypes()),
									Group:    types.ObjectNull((&utils.IdNameModel{}).AttributeTypes()),
								})
							case string(client.EnumApprovalEntityUserUserUser):
								val, err := entity.AsApprovalEntityUserResponseSchema()
								if err != nil {
									diags.AddError(
										"Failed to convert entity to user type",
										err.Error(),
									)

									return WorkflowDataSourceModel{}, diags
								}

								v := utils.IdEmailModel{
									Id:    utils.TrimmedStringValue(val.Entity.Id.String()),
									Email: utils.GetEmailStringValue(val.Entity.Email),
								}

								vObj, diagsAs := v.AsObjectValue(ctx)
								if diagsAs.HasError() {
									diags.Append(diagsAs...)
									return WorkflowDataSourceModel{}, diags
								}

								flowStep.NotifiedEntities = append(flowStep.NotifiedEntities, &workflowRulesApprovalFlowStepApprovalNotifiedModel{
									Type:     utils.TrimmedStringValue(string(val.Type)),
									User:     vObj,
									Group:    types.ObjectNull((&utils.IdNameModel{}).AttributeTypes()),
									Schedule: types.ObjectNull((&utils.IdNameModel{}).AttributeTypes()),
								})
							case string(client.DirectoryGroup):
								val, err := entity.AsApprovalEntityGroupResponseSchema()
								if err != nil {
									diags.AddError(
										"Failed to convert entity to directory group type",
										err.Error(),
									)

									return WorkflowDataSourceModel{}, diags
								}

								v := utils.IdNameModel{
									ID:   utils.TrimmedStringValue(val.Entity.Id.String()),
									Name: utils.TrimmedStringValue(val.Entity.Name),
								}

								vObj, diagsAs := v.AsObjectValue(ctx)
								if diagsAs.HasError() {
									diags.Append(diagsAs...)
									return WorkflowDataSourceModel{}, diags
								}

								flowStep.NotifiedEntities = append(flowStep.NotifiedEntities, &workflowRulesApprovalFlowStepApprovalNotifiedModel{
									Type:     utils.TrimmedStringValue(string(val.Type)),
									Group:    vObj,
									User:     types.ObjectNull((&utils.IdEmailModel{}).AttributeTypes()),
									Schedule: types.ObjectNull((&utils.IdNameModel{}).AttributeTypes()),
								})
							case string(client.EnumApprovalEntityWithoutEntityDirectManager),
								string(client.EnumApprovalEntityWithoutEntityIntegrationOwner),
								string(client.EnumApprovalEntityWithoutEntityIntegrationMaintainer),
								string(client.EnumApprovalEntityWithoutEntityResourceMaintainer),
								string(client.EnumApprovalEntityWithoutEntityResourceOwner),
								string(client.EnumApprovalEntityWithoutEntityTeamMember),
								string(client.EnumApprovalEntityWithoutEntityAutomatic):
								val, err := entity.AsNotifiedEntityNullResponseSchema()
								if err != nil {
									diags.AddError(
										"Failed to convert entity to notified entity",
										err.Error(),
									)

									return WorkflowDataSourceModel{}, diags
								}

								flowStep.NotifiedEntities = append(flowStep.NotifiedEntities, &workflowRulesApprovalFlowStepApprovalNotifiedModel{
									Type:     utils.TrimmedStringValue(string(val.Type)),
									User:     types.ObjectNull((&utils.IdEmailModel{}).AttributeTypes()),
									Schedule: types.ObjectNull((&utils.IdNameModel{}).AttributeTypes()),
									Group:    types.ObjectNull((&utils.IdNameModel{}).AttributeTypes()),
								})
							}
						}
					}

					for _, entity := range step.ApprovalEntities {
						var stepMap = make(map[string]interface{})
						jsonData, err := entity.MarshalJSON()
						if err != nil {
							diags.AddError(
								"Failed to marshal step data",
								err.Error(),
							)

							return WorkflowDataSourceModel{}, diags
						}

						err = json.Unmarshal(jsonData, &stepMap)
						if err != nil {
							diags.AddError(
								"Failed to marshal step data",
								err.Error(),
							)

							return WorkflowDataSourceModel{}, diags
						}

						typeStep, ok := stepMap["type"]
						if !ok {
							continue
						}

						switch fmt.Sprintf("%v", typeStep) {
						case string(client.OnCallIntegrationSchedule):
							val, err := entity.AsApprovalEntityScheduleResponseSchema()
							if err != nil {
								diags.AddError(
									"Failed to convert entity to schedule type",
									err.Error(),
								)

								return WorkflowDataSourceModel{}, diags
							}

							v := utils.IdNameModel{
								ID:   utils.TrimmedStringValue(val.Entity.Id.String()),
								Name: utils.TrimmedStringValue(val.Entity.Name),
							}

							vObj, diagsAs := v.AsObjectValue(ctx)
							if diagsAs.HasError() {
								diags.Append(diagsAs...)
								return WorkflowDataSourceModel{}, diags
							}

							flowStep.ApprovalEntities = append(flowStep.ApprovalEntities, &workflowRulesApprovalFlowStepApprovalNotifiedModel{
								Type:     utils.TrimmedStringValue(string(val.Type)),
								Schedule: vObj,
								User:     types.ObjectNull((&utils.IdEmailModel{}).AttributeTypes()),
								Group:    types.ObjectNull((&utils.IdNameModel{}).AttributeTypes()),
							})
						case string(client.EnumApprovalEntityUserUserUser):
							val, err := entity.AsApprovalEntityUserResponseSchema()
							if err != nil {
								diags.AddError(
									"Failed to convert entity to user type",
									err.Error(),
								)

								return WorkflowDataSourceModel{}, diags
							}

							v := utils.IdEmailModel{
								Id:    utils.TrimmedStringValue(val.Entity.Id.String()),
								Email: utils.GetEmailStringValue(val.Entity.Email),
							}

							vObj, diagsAs := v.AsObjectValue(ctx)
							if diagsAs.HasError() {
								diags.Append(diagsAs...)
								return WorkflowDataSourceModel{}, diags
							}

							flowStep.ApprovalEntities = append(flowStep.ApprovalEntities, &workflowRulesApprovalFlowStepApprovalNotifiedModel{
								Type:     utils.TrimmedStringValue(string(val.Type)),
								User:     vObj,
								Schedule: types.ObjectNull((&utils.IdNameModel{}).AttributeTypes()),
								Group:    types.ObjectNull((&utils.IdNameModel{}).AttributeTypes()),
							})
						case string(client.DirectoryGroup):
							val, err := entity.AsApprovalEntityGroupResponseSchema()
							if err != nil {
								diags.AddError(
									"Failed to convert entity to directory group type",
									err.Error(),
								)

								return WorkflowDataSourceModel{}, diags
							}

							v := utils.IdNameModel{
								ID:   utils.TrimmedStringValue(val.Entity.Id.String()),
								Name: utils.TrimmedStringValue(val.Entity.Name),
							}

							vObj, diagsAs := v.AsObjectValue(ctx)
							if diagsAs.HasError() {
								diags.Append(diagsAs...)
								return WorkflowDataSourceModel{}, diags
							}

							flowStep.ApprovalEntities = append(flowStep.ApprovalEntities, &workflowRulesApprovalFlowStepApprovalNotifiedModel{
								Type:     utils.TrimmedStringValue(string(val.Type)),
								Group:    vObj,
								User:     types.ObjectNull((&utils.IdEmailModel{}).AttributeTypes()),
								Schedule: types.ObjectNull((&utils.IdNameModel{}).AttributeTypes()),
							})
						case string(client.EnumApprovalEntityWithoutEntityDirectManager),
							string(client.EnumApprovalEntityWithoutEntityIntegrationOwner),
							string(client.EnumApprovalEntityWithoutEntityIntegrationMaintainer),
							string(client.EnumApprovalEntityWithoutEntityResourceMaintainer),
							string(client.EnumApprovalEntityWithoutEntityResourceOwner),
							string(client.EnumApprovalEntityWithoutEntityTeamMember),
							string(client.EnumApprovalEntityWithoutEntityAutomatic):
							val, err := entity.AsApprovalEntityNullResponseSchema()
							if err != nil {
								diags.AddError(
									"Failed to convert entity to notified entity",
									err.Error(),
								)

								return WorkflowDataSourceModel{}, diags
							}

							flowStep.ApprovalEntities = append(flowStep.ApprovalEntities, &workflowRulesApprovalFlowStepApprovalNotifiedModel{
								Type:     utils.TrimmedStringValue(string(val.Type)),
								User:     types.ObjectNull((&utils.IdEmailModel{}).AttributeTypes()),
								Schedule: types.ObjectNull((&utils.IdNameModel{}).AttributeTypes()),
								Group:    types.ObjectNull((&utils.IdNameModel{}).AttributeTypes()),
							})
						}
					}

					ruleModel.ApprovalFlow.Steps = append(ruleModel.ApprovalFlow.Steps, flowStep)
					sort.SliceStable(ruleModel.ApprovalFlow.Steps, func(i, j int) bool {
						sortOrderI := ruleModel.ApprovalFlow.Steps[i].SortOrder.ValueBigFloat()
						sortOrderJ := ruleModel.ApprovalFlow.Steps[j].SortOrder.ValueBigFloat()

						return sortOrderI.Cmp(sortOrderJ) == -1
					})
				}
			}

			rules = append(rules, ruleModel)
		}
	}

	sort.SliceStable(rules, func(i, j int) bool {
		sortOrderI := rules[i].SortOrder.ValueBigFloat()
		sortOrderJ := rules[j].SortOrder.ValueBigFloat()

		return sortOrderI.Cmp(sortOrderJ) == -1
	})

	return WorkflowDataSourceModel{
		Id:    utils.TrimmedStringValue(data.Id.String()),
		Name:  utils.TrimmedStringValue(data.Name),
		Rules: rules,
	}, diags
}
