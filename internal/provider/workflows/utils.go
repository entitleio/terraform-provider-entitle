package workflows

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"

	"github.com/entitleio/terraform-provider-entitle/internal/client"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/utils"
)

func getWorkflowsRules(
	ctx context.Context,
	planRules []*workflowRulesModel,
) ([]client.WorkflowRuleSchema, diag.Diagnostics) {
	var diags diag.Diagnostics
	var rules = make([]client.WorkflowRuleSchema, 0)
	for _, rule := range planRules {
		sortOrder := float32(0)
		if !rule.SortOrder.IsNull() && !rule.SortOrder.IsUnknown() {
			val, err := rule.SortOrder.ToNumberValue(ctx)
			diags.Append(err...)
			if diags.HasError() {
				return rules, diags
			}

			sortOrder, _ = val.ValueBigFloat().Float32()
		}

		underDuration := float32(0)
		if !rule.UnderDuration.IsNull() && !rule.UnderDuration.IsUnknown() {
			val, err := rule.UnderDuration.ToNumberValue(ctx)
			diags.Append(err...)
			if diags.HasError() {
				return rules, diags
			}

			underDuration, _ = val.ValueBigFloat().Float32()
		}

		inGroups := make([]client.GroupEntitySchema, 0, len(rule.InGroups))
		for _, group := range rule.InGroups {
			if group.ID.IsNull() || group.ID.IsUnknown() {
				continue
			}

			inGroups = append(
				inGroups,
				client.GroupEntitySchema{
					Id: utils.TrimPrefixSuffix(group.ID.String()),
				},
			)
		}

		inSchedules := make([]client.ScheduleEntitySchema, 0, len(rule.InSchedules))
		for _, s := range rule.InSchedules {
			if s.ID.IsNull() || s.ID.IsUnknown() {
				continue
			}

			inSchedules = append(
				inSchedules,
				client.ScheduleEntitySchema{
					Id: utils.TrimPrefixSuffix(s.ID.String()),
				},
			)
		}

		steps := make([]client.ApprovalFlowSchema, 0, len(rule.ApprovalFlow.Steps))
		for _, step := range rule.ApprovalFlow.Steps {
			val, diagsTo := step.SortOrder.ToNumberValue(ctx)
			if diagsTo.HasError() {
				diags.Append(diagsTo...)
				return rules, diags
			}

			sort, _ := val.ValueBigFloat().Float32()

			approvalEntities := make([]client.ApprovalFlowSchema_ApprovalEntities_Item, 0, len(step.ApprovalEntities))
			if len(step.ApprovalEntities) > 0 {
				for _, entity := range step.ApprovalEntities {
					if entity.Type.IsNull() || entity.Type.IsUnknown() {
						continue
					}

					switch entity.Type.ValueString() {
					case "schedule", string(client.OnCallIntegrationSchedule):
						if entity.Schedule.IsNull() {
							diags.AddError(
								"Client Error",
								"failed missing schedule data for type schedule approval entities",
							)
							return rules, diags
						}

						target := &utils.IdNameModel{}
						diagsAs := entity.Schedule.As(ctx, target, basetypes.ObjectAsOptions{
							UnhandledUnknownAsEmpty: true,
						})
						if diagsAs.HasError() {
							diags.Append(diagsAs...)
							return rules, diags
						}

						item, err := convertScheduleToApprovalFlowSchema(target)
						if err != nil {
							diags.AddError(
								"Client Error",
								"failed to convert schedule to approval flow schema",
							)

							return rules, diags
						}

						approvalEntities = append(approvalEntities, item)
					case "user", string(client.EnumApprovalEntityUserUserUser):
						if entity.User.IsNull() {
							diags.AddError(
								"Client Error",
								"failed missing user data for type approval entity",
							)
							return rules, diags
						}

						target := &utils.IdEmailModel{}
						diagsAs := entity.User.As(ctx, target, basetypes.ObjectAsOptions{
							UnhandledUnknownAsEmpty: true,
						})
						if diagsAs.HasError() {
							diags.Append(diagsAs...)
							return rules, diags
						}

						item, err := convertUserToApprovalFlowSchema(target)
						if err != nil {
							diags.AddError(
								"Client Error",
								"failed to convert user to approval flow schema",
							)

							return rules, diags
						}

						approvalEntities = append(approvalEntities, item)
					case "group", string(client.DirectoryGroup):
						if entity.Group.IsNull() {
							diags.AddError(
								"Client Error",
								"failed missing group data for type approval entity",
							)
							return rules, diags
						}

						target := &utils.IdNameModel{}
						diagsAs := entity.Group.As(ctx, target, basetypes.ObjectAsOptions{
							UnhandledUnknownAsEmpty: true,
						})
						if diagsAs.HasError() {
							diags.Append(diagsAs...)
							return rules, diags
						}

						item, err := convertDirectoryGroupToApprovalFlowSchema(target)
						if err != nil {
							diags.AddError(
								"Client Error",
								"failed to convert group to approval flow schema",
							)

							return rules, diags
						}

						approvalEntities = append(approvalEntities, item)
					case "webhook", "Webhook":
						if entity.Webhook.IsNull() {
							diags.AddError(
								"Client Error",
								"failed missing webhook data for type approval entity",
							)
							return rules, diags
						}

						target := &utils.IdNameModel{}
						diagsAs := entity.Webhook.As(ctx, target, basetypes.ObjectAsOptions{
							UnhandledUnknownAsEmpty: true,
						})
						if diagsAs.HasError() {
							diags.Append(diagsAs...)
							return rules, diags
						}

						item, err := convertWebhookToApprovalFlowSchema(target)
						if err != nil {
							diags.AddError(
								"Client Error",
								"failed to convert webhook to approval flow schema",
							)

							return rules, diags
						}

						approvalEntities = append(approvalEntities, item)
					case "approval", string(client.EnumApprovalEntityWithoutEntityDirectManager),
						string(client.EnumApprovalEntityWithoutEntityIntegrationOwner),
						string(client.EnumApprovalEntityWithoutEntityIntegrationMaintainer),
						string(client.EnumApprovalEntityWithoutEntityResourceMaintainer),
						string(client.EnumApprovalEntityWithoutEntityResourceOwner),
						string(client.EnumApprovalEntityWithoutEntityTeamMember),
						string(client.EnumApprovalEntityWithoutEntityAutomatic):
						item, err := convertApprovalToApprovalFlowSchema(
							entity.Type.ValueString(),
							nil,
						)
						if err != nil {
							diags.AddError(
								"Client Error",
								"failed to convert the approval entity to approval flow schema",
							)

							return rules, diags
						}

						approvalEntities = append(approvalEntities, item)
					default:
						diags.AddError(
							"Unsupported entity type",
							fmt.Sprintf("unsupported entity type check the provider version and api, input: %s", entity.Type.ValueString()),
						)

						return rules, diags
					}

				}
			}

			notifiedEntities := make([]client.ApprovalFlowSchema_NotifiedEntities_Item, 0, len(step.NotifiedEntities))
			if len(step.NotifiedEntities) > 0 {
				for _, entity := range step.NotifiedEntities {
					if entity.Type.IsNull() || entity.Type.IsUnknown() {
						continue
					}

					switch entity.Type.ValueString() {
					case "user", string(client.EnumApprovalEntityUserUserUser):
						if entity.User.IsNull() {
							diags.AddError(
								"Client Error",
								"failed missing user data for type approval entity",
							)
							return rules, diags
						}

						target := &utils.IdEmailModel{}
						diagsAs := entity.User.As(ctx, target, basetypes.ObjectAsOptions{
							UnhandledUnknownAsEmpty: true,
						})
						if diagsAs.HasError() {
							diags.Append(diagsAs...)
							return rules, diags
						}

						t := client.ApprovalFlowSchema_NotifiedEntities_Item{}
						err := t.FromApprovalEntityUserSchema(client.ApprovalEntityUserSchema{
							Type: client.EnumApprovalEntityUserUserUser,
							Entity: client.UserEntitySchema{
								Id: target.Id.String(),
							},
						})
						if err != nil {
							diags.AddError(
								"Client Error",
								fmt.Sprintf("failed to convert user to approval to notified entity, error: %v", err),
							)

							return rules, diags
						}

						notifiedEntities = append(notifiedEntities, t)
					case "group", string(client.DirectoryGroup):
						if entity.Group.IsNull() {
							diags.AddError(
								"Client Error",
								"failed missing group data for type group approval entities",
							)
							return rules, diags
						}

						target := &utils.IdNameModel{}
						diagsAs := entity.Group.As(ctx, target, basetypes.ObjectAsOptions{
							UnhandledUnknownAsEmpty: true,
						})
						if diagsAs.HasError() {
							diags.Append(diagsAs...)
							return rules, diags
						}

						t := client.ApprovalFlowSchema_NotifiedEntities_Item{}
						err := t.FromApprovalEntityGroupSchema(client.ApprovalEntityGroupSchema{
							Type: client.DirectoryGroup,
							Entity: client.GroupEntitySchema{
								Id: target.ID.String(),
							},
						})
						if err != nil {
							diags.AddError(
								"Client Error",
								fmt.Sprintf("failed to convert user to approval to notified entity, error: %v", err),
							)

							return rules, diags
						}

						notifiedEntities = append(notifiedEntities, t)
					case "schedule", string(client.OnCallIntegrationSchedule):
						if entity.Schedule.IsNull() {
							diags.AddError(
								"Client Error",
								"failed missing schedule data for type schedule approval entities",
							)
							return rules, diags
						}

						target := &utils.IdNameModel{}
						diagsAs := entity.Schedule.As(ctx, target, basetypes.ObjectAsOptions{
							UnhandledUnknownAsEmpty: true,
						})
						if diagsAs.HasError() {
							diags.Append(diagsAs...)
							return rules, diags
						}

						t := client.ApprovalFlowSchema_NotifiedEntities_Item{}
						err := t.FromApprovalEntityScheduleSchema(client.ApprovalEntityScheduleSchema{
							Type: client.OnCallIntegrationSchedule,
							Entity: client.ScheduleEntitySchema{
								Id: target.ID.String(),
							},
						})
						if err != nil {
							diags.AddError(
								"Client Error",
								fmt.Sprintf("failed to convert user to approval to notified entity, error: %v", err),
							)

							return rules, diags
						}

						notifiedEntities = append(notifiedEntities, t)
					case "webhook", "Webhook":
						if entity.Webhook.IsNull() {
							diags.AddError(
								"Client Error",
								"failed missing webhook data for type notified entity",
							)
							return rules, diags
						}

						target := &utils.IdNameModel{}
						diagsAs := entity.Webhook.As(ctx, target, basetypes.ObjectAsOptions{
							UnhandledUnknownAsEmpty: true,
						})
						if diagsAs.HasError() {
							diags.Append(diagsAs...)
							return rules, diags
						}

						item, err := convertWebhookToNotifiedFlowSchema(target)
						if err != nil {
							diags.AddError(
								"Client Error",
								"failed to convert webhook to notified flow schema",
							)

							return rules, diags
						}

						notifiedEntities = append(notifiedEntities, item)
					case "notified", string(client.EnumNotifiedEntityWithoutEntityDirectManager),
						string(client.EnumNotifiedEntityWithoutEntityIntegrationMaintainer),
						string(client.EnumNotifiedEntityWithoutEntityIntegrationOwner),
						string(client.EnumNotifiedEntityWithoutEntityResourceMaintainer),
						string(client.EnumNotifiedEntityWithoutEntityResourceOwner),
						string(client.EnumNotifiedEntityWithoutEntityTeamMember):
						item, err := convertApprovalToNotifiedFlowSchema(
							entity.Type.ValueString(),
							nil,
						)
						if err != nil {
							diags.AddError(
								"Client Error",
								"failed to convert the approval entity to approval flow schema",
							)

							return rules, diags
						}

						notifiedEntities = append(notifiedEntities, item)
					default:
						diags.AddError(
							"Unsupported entity type",
							fmt.Sprintf("unsupported entity type check the provider version and api, input: %s", entity.Type.ValueString()),
						)

						return rules, diags
					}
				}
			}

			steps = append(steps, client.ApprovalFlowSchema{
				ApprovalEntities: approvalEntities,
				NotifiedEntities: notifiedEntities,
				Operator:         client.EnumApprovalFlowOperator(step.Operator.ValueString()),
				SortOrder:        sort,
			})
		}

		item := client.WorkflowRuleSchema{
			AnySchedule: rule.AnySchedule.ValueBool(),
			ApprovalFlow: client.WorkflowApprovalFlowSchema{
				Steps: steps,
			},
			InGroups:      inGroups,
			InSchedules:   inSchedules,
			SortOrder:     sortOrder,
			UnderDuration: client.EnumAllowedDurations(underDuration),
		}

		if len(inSchedules) > 0 && item.AnySchedule {
			diags.AddError(
				"Invalid Input",
				"not allowed to put in_schedules values when the any_schedule parameter is true",
			)

			return rules, diags
		}

		rules = append(rules, item)
	}

	return rules, diags
}

// convertUserToApprovalFlowSchema is a function that converts a user to an Approval Flow Schema.
func convertUserToApprovalFlowSchema(user *utils.IdEmailModel) (client.ApprovalFlowSchema_ApprovalEntities_Item, error) {
	// Create an ApprovalEntityUserSchema with user information
	schemaUser := client.ApprovalEntityUserSchema{
		Type: client.EnumApprovalEntityUserUserUser,
		Entity: client.UserEntitySchema{
			Id: user.Id.ValueString(),
		},
	}

	// Create an empty ApprovalFlowSchema_ApprovalEntities_Item
	item := client.ApprovalFlowSchema_ApprovalEntities_Item{}

	// Merge the ApprovalEntityUserSchema into the item
	err := item.MergeApprovalEntityUserSchema(schemaUser)
	if err != nil {
		return item, err
	}
	return item, nil
}

// convertDirectoryGroupToApprovalFlowSchema is a function that converts a directory group to an Approval Flow Schema.
func convertDirectoryGroupToApprovalFlowSchema(directoryGroup *utils.IdNameModel) (client.ApprovalFlowSchema_ApprovalEntities_Item, error) {
	// Create an ApprovalEntityGroupSchema with directory group information
	schemaGroup := client.ApprovalEntityGroupSchema{
		Type: client.DirectoryGroup,
		Entity: client.GroupEntitySchema{
			Id: directoryGroup.ID.ValueString(),
		},
	}

	// Create an empty ApprovalFlowSchema_ApprovalEntities_Item
	item := client.ApprovalFlowSchema_ApprovalEntities_Item{}

	// Merge the ApprovalEntityGroupSchema into the item
	err := item.MergeApprovalEntityGroupSchema(schemaGroup)
	if err != nil {
		return item, err
	}

	return item, nil
}

// convertScheduleToApprovalFlowSchema is a function that converts a schedule to an Approval Flow Schema.
func convertScheduleToApprovalFlowSchema(onCallIntegrationSchedule *utils.IdNameModel) (client.ApprovalFlowSchema_ApprovalEntities_Item, error) {
	// Create an ApprovalEntityScheduleSchema with schedule information
	schemaScheduleEntitySchema := client.ApprovalEntityScheduleSchema{
		Type: client.OnCallIntegrationSchedule,
		Entity: client.ScheduleEntitySchema{
			Id: onCallIntegrationSchedule.ID.ValueString(),
		},
	}

	// Create an empty ApprovalFlowSchema_ApprovalEntities_Item
	item := client.ApprovalFlowSchema_ApprovalEntities_Item{}

	// Merge the ApprovalEntityScheduleSchema into the item
	err := item.MergeApprovalEntityScheduleSchema(schemaScheduleEntitySchema)
	if err != nil {
		return item, err
	}

	return item, nil
}

// convertApprovalToApprovalFlowSchema is a function that converts an approval to an Approval Flow Schema.
func convertApprovalToApprovalFlowSchema(t string, val *string) (client.ApprovalFlowSchema_ApprovalEntities_Item, error) {
	// Create an ApprovalEntityNullSchema with approval information
	schemaScheduleEntitySchema := client.ApprovalEntityNullSchema{
		Type:   client.EnumApprovalEntityWithoutEntity(t),
		Entity: val,
	}

	// Create an empty ApprovalFlowSchema_ApprovalEntities_Item
	item := client.ApprovalFlowSchema_ApprovalEntities_Item{}

	// Merge the ApprovalEntityNullSchema into the item
	err := item.MergeApprovalEntityNullSchema(schemaScheduleEntitySchema)
	if err != nil {
		return item, err
	}

	return item, nil
}

// convertApprovalToNotifiedFlowSchema is a function that converts an approval to an Approval Flow Schema.
func convertApprovalToNotifiedFlowSchema(t string, val *string) (client.ApprovalFlowSchema_NotifiedEntities_Item, error) {
	// Create an ApprovalEntityNullSchema with approval information
	schemaScheduleEntitySchema := client.NotifiedApprovalEntityWithoutEntitySchema{
		Type:   client.EnumNotifiedEntityWithoutEntity(t),
		Entity: val,
	}

	// Create an empty ApprovalFlowSchema_ApprovalEntities_Item
	item := client.ApprovalFlowSchema_NotifiedEntities_Item{}

	// Merge the ApprovalEntityNullSchema into the item
	err := item.MergeNotifiedApprovalEntityWithoutEntitySchema(schemaScheduleEntitySchema)
	if err != nil {
		return item, err
	}

	return item, nil
}

// webhookEntitySchema represents the webhook entity structure for API requests.
type webhookEntitySchema struct {
	Entity struct {
		Id string `json:"id"`
	} `json:"entity"`
	Type string `json:"type"`
}

// convertWebhookToApprovalFlowSchema converts a webhook to an Approval Flow Schema.
func convertWebhookToApprovalFlowSchema(webhook *utils.IdNameModel) (client.ApprovalFlowSchema_ApprovalEntities_Item, error) {
	schema := webhookEntitySchema{
		Type: "Webhook",
	}
	schema.Entity.Id = webhook.ID.ValueString()

	item := client.ApprovalFlowSchema_ApprovalEntities_Item{}
	b, err := json.Marshal(schema)
	if err != nil {
		return item, err
	}

	err = item.UnmarshalJSON(b)
	return item, err
}

// convertWebhookToNotifiedFlowSchema converts a webhook to a Notified Flow Schema.
func convertWebhookToNotifiedFlowSchema(webhook *utils.IdNameModel) (client.ApprovalFlowSchema_NotifiedEntities_Item, error) {
	schema := webhookEntitySchema{
		Type: "Webhook",
	}
	schema.Entity.Id = webhook.ID.ValueString()

	item := client.ApprovalFlowSchema_NotifiedEntities_Item{}
	b, err := json.Marshal(schema)
	if err != nil {
		return item, err
	}

	err = item.UnmarshalJSON(b)
	return item, err
}

// entitySortKey returns a string key that uniquely identifies an approval/notified entity
// by its type and entity ID, used for matching entities between plan and API response.
func entitySortKey(entity *workflowRulesApprovalFlowStepApprovalNotifiedModel) string {
	t := entity.Type.ValueString()
	id := ""

	if !entity.User.IsNull() && !entity.User.IsUnknown() {
		if idAttr, ok := entity.User.Attributes()["id"]; ok {
			if strVal, ok := idAttr.(basetypes.StringValue); ok {
				id = strVal.ValueString()
			}
		}
	} else if !entity.Group.IsNull() && !entity.Group.IsUnknown() {
		if idAttr, ok := entity.Group.Attributes()["id"]; ok {
			if strVal, ok := idAttr.(basetypes.StringValue); ok {
				id = strVal.ValueString()
			}
		}
	} else if !entity.Schedule.IsNull() && !entity.Schedule.IsUnknown() {
		if idAttr, ok := entity.Schedule.Attributes()["id"]; ok {
			if strVal, ok := idAttr.(basetypes.StringValue); ok {
				id = strVal.ValueString()
			}
		}
	} else if !entity.Webhook.IsNull() && !entity.Webhook.IsUnknown() {
		if idAttr, ok := entity.Webhook.Attributes()["id"]; ok {
			if strVal, ok := idAttr.(basetypes.StringValue); ok {
				id = strVal.ValueString()
			}
		}
	}

	return t + ":" + id
}

// reorderEntities reorders result entities to match the order of plan entities.
// Entities are matched by their type and entity ID.
func reorderEntities(
	planEntities []*workflowRulesApprovalFlowStepApprovalNotifiedModel,
	resultEntities []*workflowRulesApprovalFlowStepApprovalNotifiedModel,
) []*workflowRulesApprovalFlowStepApprovalNotifiedModel {
	if len(planEntities) == 0 || len(resultEntities) == 0 {
		return resultEntities
	}

	// Use a queue per key so duplicate keys (e.g. two "direct_manager"
	// entities) are consumed one at a time instead of overwritten.
	resultQueues := make(map[string][]*workflowRulesApprovalFlowStepApprovalNotifiedModel, len(resultEntities))
	for _, entity := range resultEntities {
		key := entitySortKey(entity)
		resultQueues[key] = append(resultQueues[key], entity)
	}

	reordered := make([]*workflowRulesApprovalFlowStepApprovalNotifiedModel, 0, len(resultEntities))
	consumed := make(map[string]int, len(resultQueues))

	for _, planEntity := range planEntities {
		key := entitySortKey(planEntity)
		idx := consumed[key]
		if queue, ok := resultQueues[key]; ok && idx < len(queue) {
			reordered = append(reordered, queue[idx])
			consumed[key] = idx + 1
		}
	}

	// Append any remaining result entities not consumed above.
	for key, queue := range resultQueues {
		for i := consumed[key]; i < len(queue); i++ {
			reordered = append(reordered, queue[i])
		}
	}

	return reordered
}

// reconcileEntityOrder reorders approval_entities and notified_entities in the result
// to match the plan order, preventing "inconsistent result after apply" errors when
// the API returns entities in a different order than the plan.
func reconcileEntityOrder(
	planRules []*workflowRulesModel,
	resultRules []*workflowRulesModel,
) {
	for ruleIdx := range resultRules {
		if ruleIdx >= len(planRules) {
			break
		}

		for stepIdx := range resultRules[ruleIdx].ApprovalFlow.Steps {
			if stepIdx >= len(planRules[ruleIdx].ApprovalFlow.Steps) {
				break
			}

			planStep := planRules[ruleIdx].ApprovalFlow.Steps[stepIdx]
			resultStep := resultRules[ruleIdx].ApprovalFlow.Steps[stepIdx]

			resultStep.ApprovalEntities = reorderEntities(
				planStep.ApprovalEntities, resultStep.ApprovalEntities,
			)
			resultStep.NotifiedEntities = reorderEntities(
				planStep.NotifiedEntities, resultStep.NotifiedEntities,
			)
		}
	}
}

// convertFullWorkflowResultResponseSchemaToModel is a utility function used to convert the API response data
// (of type client.FullWorkflowResultResponseSchema) to a Terraform resource model (of type WorkflowResourceModel).
//
// It extracts and transforms data from the API response into a format that can be stored in Terraform state.
// It returns the converted model and any diagnostic information if there are errors during the conversion.
func convertFullWorkflowResultResponseSchemaToModel(
	ctx context.Context,
	data *client.FullWorkflowResultResponseSchema,
) (WorkflowResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	// Check if the API response data is nil
	if data == nil {
		diags.AddError(
			"No data",
			"Failed: the given schema data is nil",
		)

		return WorkflowResourceModel{}, diags
	}

	// Convert the data using the converterWorkflow function
	w, diags := converterWorkflow(ctx, data)
	if diags.HasError() {
		return WorkflowResourceModel{}, diags
	}

	// Create the Terraform resource model using the extracted data
	return WorkflowResourceModel{
		ID:    w.Id,
		Name:  w.Name,
		Rules: w.Rules,
	}, diags
}
