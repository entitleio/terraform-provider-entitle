package workflows

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/utils"
	"math/big"
	"net/http"
	"sort"

	"github.com/entitleio/terraform-provider-entitle/internal/client"
	"github.com/entitleio/terraform-provider-entitle/internal/validators"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
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
		MarkdownDescription: "A workflow in Entitle is a generic description of Just-In-Time permissions approval " +
			"process, which is triggered after the permissions were requested by a user. Who should approve by " +
			"approval order, to whom, and for how long. After the workflow is defined, it can be assigned to " +
			"multiple entities which are part of the Just-In-Time permissions approval process: integrations, " +
			"resources, roles and bundles." +
			"\n\nEvery workflow is comprised of multiple rules. Their order of is important, the first rule to be " +
			"validated sets the actual approval process for the permissions request.",
		Description: "A workflow in Entitle is a generic description of Just-In-Time permissions approval " +
			"process, which is triggered after the permissions were requested by a user. Who should approve by " +
			"approval order, to whom, and for how long. After the workflow is defined, it can be assigned to " +
			"multiple entities which are part of the Just-In-Time permissions approval process: integrations, " +
			"resources, roles and bundles." +
			"\n\nEvery workflow is comprised of multiple rules. Their order of is important, the first rule to be " +
			"validated sets the actual approval process for the permissions request.",
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
				MarkdownDescription: "name",
				Description:         "name",
			},
			"rules": schema.ListNestedAttribute{
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"any_schedule": schema.BoolAttribute{
							Computed:            true,
							Description:         "any_schedule",
							MarkdownDescription: "any_schedule",
						},
						"sort_order": schema.NumberAttribute{
							Computed:            true,
							Description:         "sort_order",
							MarkdownDescription: "sort_order",
						},
						"under_duration": schema.NumberAttribute{
							Computed:            true,
							Description:         "under_duration",
							MarkdownDescription: "under_duration",
						},
						"in_groups": schema.ListNestedAttribute{
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"id": schema.StringAttribute{
										Computed:            true,
										Description:         "id",
										MarkdownDescription: "id",
									},
									"name": schema.StringAttribute{
										Computed:            true,
										Description:         "name",
										MarkdownDescription: "name",
									},
								},
							},
							Computed:            true,
							Description:         "in_groups",
							MarkdownDescription: "in_groups",
						},
						"in_schedules": schema.ListNestedAttribute{
							NestedObject: schema.NestedAttributeObject{
								Attributes: map[string]schema.Attribute{
									"id": schema.StringAttribute{
										Computed:            true,
										Description:         "id",
										MarkdownDescription: "id",
									},
									"name": schema.StringAttribute{
										Computed:            true,
										Description:         "name",
										MarkdownDescription: "name",
									},
								},
							},
							Computed:            true,
							Description:         "in_schedules",
							MarkdownDescription: "in_schedules",
						},
						"approval_flow": schema.SingleNestedAttribute{
							Attributes: map[string]schema.Attribute{
								"steps": schema.ListNestedAttribute{
									NestedObject: schema.NestedAttributeObject{
										Attributes: map[string]schema.Attribute{
											"operator": schema.StringAttribute{
												Computed:            true,
												Description:         "operator",
												MarkdownDescription: "operator",
											},
											"sort_order": schema.NumberAttribute{
												Computed:            true,
												Description:         "sort_order",
												MarkdownDescription: "sort_order",
											},
											"notified_entities": schema.ListNestedAttribute{
												NestedObject: schema.NestedAttributeObject{
													Attributes: map[string]schema.Attribute{
														"type": schema.StringAttribute{
															Computed:            true,
															Description:         "",
															MarkdownDescription: "",
														},
														"user": schema.SingleNestedAttribute{
															Attributes: map[string]schema.Attribute{
																"id": schema.StringAttribute{
																	Computed:            true,
																	Description:         "",
																	MarkdownDescription: "",
																},
																"email": schema.StringAttribute{
																	Computed:            true,
																	Description:         "email",
																	MarkdownDescription: "email",
																},
															},
															Computed:            true,
															Description:         "user",
															MarkdownDescription: "user",
														},
														"group": schema.SingleNestedAttribute{
															Attributes: map[string]schema.Attribute{
																"id": schema.StringAttribute{
																	Computed:            true,
																	Description:         "",
																	MarkdownDescription: "",
																},
																"name": schema.StringAttribute{
																	Computed:            true,
																	Description:         "",
																	MarkdownDescription: "",
																},
															},
															Computed:            true,
															Description:         "group",
															MarkdownDescription: "group",
														},
														"schedule": schema.SingleNestedAttribute{
															Attributes: map[string]schema.Attribute{
																"id": schema.StringAttribute{
																	Computed:            true,
																	Description:         "",
																	MarkdownDescription: "",
																},
																"name": schema.StringAttribute{
																	Computed:            true,
																	Description:         "",
																	MarkdownDescription: "",
																},
															},
															Computed:            true,
															Description:         "group",
															MarkdownDescription: "group",
														},
														"value": schema.SingleNestedAttribute{
															Attributes: map[string]schema.Attribute{
																"notified": schema.StringAttribute{
																	Computed:            true,
																	Description:         "",
																	MarkdownDescription: "",
																},
															},
															Computed:            true,
															Description:         "value",
															MarkdownDescription: "value",
														},
													},
												},
												Computed:            true,
												Description:         "in_groups",
												MarkdownDescription: "in_groups",
											},
											"approval_entities": schema.ListNestedAttribute{
												NestedObject: schema.NestedAttributeObject{
													Attributes: map[string]schema.Attribute{
														"type": schema.StringAttribute{
															Computed:            true,
															Description:         "",
															MarkdownDescription: "",
														},
														"user": schema.SingleNestedAttribute{
															Attributes: map[string]schema.Attribute{
																"id": schema.StringAttribute{
																	Computed:            true,
																	Description:         "",
																	MarkdownDescription: "",
																},
																"email": schema.StringAttribute{
																	Computed:            true,
																	Description:         "email",
																	MarkdownDescription: "email",
																},
															},
															Computed:            true,
															Description:         "user",
															MarkdownDescription: "user",
														},
														"group": schema.SingleNestedAttribute{
															Attributes: map[string]schema.Attribute{
																"id": schema.StringAttribute{
																	Computed:            true,
																	Description:         "",
																	MarkdownDescription: "",
																},
																"name": schema.StringAttribute{
																	Computed:            true,
																	Description:         "",
																	MarkdownDescription: "",
																},
															},
															Computed:            true,
															Description:         "group",
															MarkdownDescription: "group",
														},
														"schedule": schema.SingleNestedAttribute{
															Attributes: map[string]schema.Attribute{
																"id": schema.StringAttribute{
																	Computed:            true,
																	Description:         "",
																	MarkdownDescription: "",
																},
																"name": schema.StringAttribute{
																	Computed:            true,
																	Description:         "",
																	MarkdownDescription: "",
																},
															},
															Computed:            true,
															Description:         "group",
															MarkdownDescription: "group",
														},
														"value": schema.SingleNestedAttribute{
															Attributes: map[string]schema.Attribute{
																"approval": schema.StringAttribute{
																	Computed:            true,
																	Description:         "",
																	MarkdownDescription: "",
																},
															},
															Computed:            true,
															Description:         "value",
															MarkdownDescription: "value",
														},
													},
												},
												Computed:            true,
												Description:         "in_groups",
												MarkdownDescription: "in_groups",
											},
										},
									},
									Computed:            true,
									Description:         "in_groups",
									MarkdownDescription: "in_groups",
								},
							},
							Computed:            true,
							Description:         "approval_flow",
							MarkdownDescription: "approval_flow",
						},
					},
				},
				Computed:            true,
				Description:         "rules",
				MarkdownDescription: "rules",
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

	uid, err := uuid.Parse(data.Id.String())
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("failed to parse the entitle policy id (%s) to UUID, got error: %s", data.Id.String(), err),
		)
		return
	}

	workflowResp, err := d.client.WorkflowsShowWithResponse(ctx, uid)
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Unable to get the bundle by the id (%s), got error: %s", uid.String(), err),
		)
		return
	}

	if workflowResp.HTTPResponse.StatusCode != 200 {
		if workflowResp.HTTPResponse.StatusCode == http.StatusUnauthorized ||
			workflowResp.HTTPResponse.StatusCode == http.StatusBadRequest {
			resp.Diagnostics.AddError(
				"Client Error",
				"unauthorized token, update the entitle token and retry please",
			)
			return
		}

		errBody, _ := utils.GetErrorBody(workflowResp.Body)
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf(
				"failed to get the workflow by the id (%s), status code: %d%s",
				uid.String(),
				workflowResp.HTTPResponse.StatusCode,
				errBody.GetMessage(),
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
			"failed the given schema data is nil",
		)

		return WorkflowDataSourceModel{}, diags
	}

	var rules []*workflowRulesModel
	if len(data.Rules) > 0 {
		rules = make([]*workflowRulesModel, 0)
		for _, rule := range data.Rules {
			ruleModel := &workflowRulesModel{
				AnySchedule: types.BoolValue(rule.AnySchedule),
				ApprovalFlow: workflowRulesApprovalFlowModel{
					Steps: make([]*workflowRulesApprovalFlowStepModel, 0),
				},
				InGroups:      make([]*utils.IdNameModel, 0),
				InSchedules:   make([]*utils.IdNameModel, 0),
				SortOrder:     types.NumberValue(big.NewFloat(float64(rule.SortOrder))),
				UnderDuration: types.NumberValue(big.NewFloat(float64(rule.UnderDuration))),
			}

			for _, inGroup := range rule.InGroups {
				ruleModel.InGroups = append(ruleModel.InGroups, &utils.IdNameModel{
					ID:   utils.TrimmedStringValue(inGroup.Id.String()),
					Name: utils.TrimmedStringValue(inGroup.Name),
				})
			}

			for _, inSchedule := range rule.InSchedules {
				ruleModel.InSchedules = append(ruleModel.InSchedules, &utils.IdNameModel{
					ID:   utils.TrimmedStringValue(inSchedule.Id.String()),
					Name: utils.TrimmedStringValue(inSchedule.Name),
				})
			}

			if len(rule.ApprovalFlow.Steps) > 0 {
				for _, step := range rule.ApprovalFlow.Steps {
					flowStep := &workflowRulesApprovalFlowStepModel{
						ApprovalEntities: make([]*workflowRulesApprovalFlowStepApprovalNotifiedModel, 0),
						NotifiedEntities: make([]*workflowRulesApprovalFlowStepApprovalNotifiedModel, 0),
						Operator:         utils.TrimmedStringValue(string(step.Operator)),
						SortOrder:        types.NumberValue(big.NewFloat(float64(step.SortOrder))),
					}

					for _, entity := range step.NotifiedEntities {
						var stepMap = make(map[string]interface{}, 0)

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
								Value:    types.ObjectNull((&workflowRulesApprovalFlowStepNotifiedEntityModel{}).attributeTypes()),
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

							emailString, err := utils.GetEmailString(val.Entity.Email)
							if err != nil {
								diags.AddError(
									"Failed to convert the user email to string",
									err.Error(),
								)

								return WorkflowDataSourceModel{}, diags
							}

							v := utils.IdEmailModel{
								Id:    utils.TrimmedStringValue(val.Entity.Id.String()),
								Email: utils.TrimmedStringValue(emailString),
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
								Value:    types.ObjectNull((&workflowRulesApprovalFlowStepNotifiedEntityModel{}).attributeTypes()),
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
								Value:    types.ObjectNull((&workflowRulesApprovalFlowStepNotifiedEntityModel{}).attributeTypes()),
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

							v := workflowRulesApprovalFlowStepNotifiedEntityModel{
								Notified: types.StringPointerValue(val.Entity),
							}

							if val.Entity == nil {
								v = workflowRulesApprovalFlowStepNotifiedEntityModel{
									Notified: types.StringNull(),
								}
							}

							vObj, diagsAs := v.AsObjectValue(ctx)
							if diagsAs.HasError() {
								diags.Append(diagsAs...)
								return WorkflowDataSourceModel{}, diags
							}

							flowStep.NotifiedEntities = append(flowStep.NotifiedEntities, &workflowRulesApprovalFlowStepApprovalNotifiedModel{
								Type:     utils.TrimmedStringValue(string(val.Type)),
								Value:    vObj,
								User:     types.ObjectNull((&utils.IdEmailModel{}).AttributeTypes()),
								Schedule: types.ObjectNull((&utils.IdNameModel{}).AttributeTypes()),
								Group:    types.ObjectNull((&utils.IdNameModel{}).AttributeTypes()),
							})
						}
					}

					for _, entity := range step.ApprovalEntities {
						var stepMap = make(map[string]interface{}, 0)
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
								Value:    types.ObjectNull((&workflowRulesApprovalFlowStepApprovalEntityModel{}).attributeTypes()),
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

							emailString, err := utils.GetEmailString(val.Entity.Email)
							if err != nil {
								diags.AddError(
									"Failed to convert the user email to string",
									err.Error(),
								)

								return WorkflowDataSourceModel{}, diags
							}

							v := utils.IdEmailModel{
								Id:    utils.TrimmedStringValue(val.Entity.Id.String()),
								Email: utils.TrimmedStringValue(emailString),
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
								Value:    types.ObjectNull((&workflowRulesApprovalFlowStepApprovalEntityModel{}).attributeTypes()),
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
								Value:    types.ObjectNull((&workflowRulesApprovalFlowStepApprovalEntityModel{}).attributeTypes()),
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

							v := workflowRulesApprovalFlowStepApprovalEntityModel{
								Approval: types.StringPointerValue(val.Entity),
							}

							if val.Entity == nil {
								v = workflowRulesApprovalFlowStepApprovalEntityModel{
									Approval: types.StringNull(),
								}
							}

							vObj, diagsAs := v.AsObjectValue(ctx)
							if diagsAs.HasError() {
								diags.Append(diagsAs...)
								return WorkflowDataSourceModel{}, diags
							}

							flowStep.ApprovalEntities = append(flowStep.ApprovalEntities, &workflowRulesApprovalFlowStepApprovalNotifiedModel{
								Type:     utils.TrimmedStringValue(string(val.Type)),
								User:     types.ObjectNull((&utils.IdEmailModel{}).AttributeTypes()),
								Schedule: types.ObjectNull((&utils.IdNameModel{}).AttributeTypes()),
								Group:    types.ObjectNull((&utils.IdNameModel{}).AttributeTypes()),
								Value:    vObj,
							})
						}
					}

					ruleModel.ApprovalFlow.Steps = append(ruleModel.ApprovalFlow.Steps, flowStep)
					sort.Slice(ruleModel.ApprovalFlow.Steps, func(i, j int) bool {
						sortOrderI := ruleModel.ApprovalFlow.Steps[i].SortOrder.ValueBigFloat()
						sortOrderJ := ruleModel.ApprovalFlow.Steps[j].SortOrder.ValueBigFloat()

						iValue, _ := sortOrderI.Float32()
						jValue, _ := sortOrderJ.Float32()

						return iValue < jValue
					})
				}
			}

			rules = append(rules, ruleModel)
		}
	}

	return WorkflowDataSourceModel{
		Id:    utils.TrimmedStringValue(data.Id.String()),
		Name:  utils.TrimmedStringValue(data.Name),
		Rules: rules,
	}, diags
}
