package integrations

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/entitleio/terraform-provider-entitle/internal/client"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/utils"
)

func CreateIntegration(
	ctx context.Context,
	cli *client.ClientWithResponses,
	base BaseIntegrationResourceModel,
	appName applicationName,
	parsedConnectionJson map[string]interface{},
) (BaseIntegrationResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	body, diags := BuildCreateBodyFromPlan(ctx, base, appName, &parsedConnectionJson)
	if diags.HasError() {
		return BaseIntegrationResourceModel{}, diags
	}

	integrationResp, err := cli.IntegrationsCreateWithResponse(ctx, body)
	if err != nil {
		diags.AddError(
			utils.ErrApiConnection.Error(),
			fmt.Sprintf("Unable to create the integration, got error: %v", err),
		)
		return BaseIntegrationResourceModel{}, diags
	}

	if err = utils.HTTPResponseToError(integrationResp.HTTPResponse.StatusCode, integrationResp.Body); err != nil {
		diags.AddError(
			utils.ErrApiResponse.Error(),
			fmt.Sprintf(
				"Failed to create the Integration, %s, status code: %d, %s",
				string(integrationResp.Body),
				integrationResp.HTTPResponse.StatusCode,
				err.Error(),
			),
		)
		return BaseIntegrationResourceModel{}, diags
	}

	agentTokenName := ""
	if base.AgentToken != nil {
		agentTokenName = base.AgentToken.Name.ValueString()
	}

	result, _, rDiags := ConvertBaseIntegrationResultToBaseModel(ctx, &integrationResp.JSON200.Result, agentTokenName)
	diags.Append(rDiags...)
	if diags.HasError() {
		return BaseIntegrationResourceModel{}, diags
	}

	tflog.Trace(ctx, "created a entitle integration resource")
	return result, diags
}

func UpdateIntegration(
	ctx context.Context,
	cli *client.ClientWithResponses,
	base BaseIntegrationResourceModel,
	appName applicationName,
	parsedConnectionJson map[string]interface{},
) (BaseIntegrationResourceModel, diag.Diagnostics) {
	var diags diag.Diagnostics

	uid, err := uuid.Parse(base.ID.String())
	if err != nil {
		diags.AddError(
			"Client Error",
			fmt.Sprintf("Failed to parse the given id to UUID format, got error: %v", err),
		)
		return BaseIntegrationResourceModel{}, diags
	}

	body, bDiags := BuildUpdateBodyFromPlan(ctx, base, appName, &parsedConnectionJson)
	diags.Append(bDiags...)
	if diags.HasError() {
		return BaseIntegrationResourceModel{}, diags
	}

	integrationResp, err := cli.IntegrationsUpdateWithResponse(ctx, uid, body)
	if err != nil {
		diags.AddError(
			utils.ErrApiConnection.Error(),
			fmt.Sprintf("Unable to update the integration by the id (%s), got error: %s", uid.String(), err),
		)
		return BaseIntegrationResourceModel{}, diags
	}

	if err = utils.HTTPResponseToError(integrationResp.HTTPResponse.StatusCode, integrationResp.Body); err != nil {
		diags.AddError(
			utils.ErrApiResponse.Error(),
			fmt.Sprintf(
				"Failed to update the Integration by the id (%s), status code: %d, %s",
				uid.String(),
				integrationResp.HTTPResponse.StatusCode,
				err.Error(),
			),
		)
		return BaseIntegrationResourceModel{}, diags
	}

	agentTokenName := ""
	if base.AgentToken != nil {
		agentTokenName = base.AgentToken.Name.ValueString()
	}

	result, _, rDiags := ConvertBaseIntegrationResultToBaseModel(ctx, &integrationResp.JSON200.Result, agentTokenName)
	diags.Append(rDiags...)
	if diags.HasError() {
		return BaseIntegrationResourceModel{}, diags
	}

	return result, diags
}

func ReadIntegration(
	ctx context.Context,
	cli *client.ClientWithResponses,
	base BaseIntegrationResourceModel,
) (BaseIntegrationResourceModel, string, diag.Diagnostics) {
	var diags diag.Diagnostics

	uid, err := uuid.Parse(base.ID.String())
	if err != nil {
		diags.AddError(
			"Client Error",
			fmt.Sprintf("Failed to parse the resource id (%s) to UUID, got error: %s", base.ID.String(), err),
		)
		return BaseIntegrationResourceModel{}, "", diags
	}

	integrationResp, err := cli.IntegrationsShowWithResponse(ctx, uid)
	if err != nil {
		diags.AddError(
			utils.ErrApiConnection.Error(),
			fmt.Sprintf("Unable to get the integration by the id (%s), got error: %s", uid.String(), err),
		)
		return BaseIntegrationResourceModel{}, "", diags
	}

	if err = utils.HTTPResponseToError(integrationResp.HTTPResponse.StatusCode, integrationResp.Body); err != nil {
		diags.AddError(
			utils.ErrApiResponse.Error(),
			fmt.Sprintf(
				"Failed to get the Integration by the id (%s), status code: %d, %s",
				uid.String(),
				integrationResp.HTTPResponse.StatusCode,
				err.Error(),
			),
		)
		return BaseIntegrationResourceModel{}, "", diags
	}

	agentTokenName := ""
	if base.AgentToken != nil {
		agentTokenName = base.AgentToken.Name.ValueString()
	}

	return ConvertBaseIntegrationResultToBaseModel(ctx, &integrationResp.JSON200.Result, agentTokenName)
}

func DeleteIntegration(ctx context.Context, cli *client.ClientWithResponses, data BaseIntegrationResourceModel, resp *resource.DeleteResponse) {
	parsedUUID, err := uuid.Parse(data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Failed to parse the resource id (%s) to UUID, got error: %v; the integration was NOT deleted", data.ID.String(), err),
		)
		return
	}

	httpResp, err := cli.IntegrationsDestroyWithResponse(ctx, parsedUUID)
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiConnection.Error(),
			fmt.Sprintf("Unable to delete integrations, id: (%s), got error: %v", data.ID.String(), err),
		)
		return
	}

	err = utils.HTTPResponseToError(httpResp.HTTPResponse.StatusCode, httpResp.Body, utils.WithIgnoreNotFound())
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiResponse.Error(),
			fmt.Sprintf(
				"Failed to delete the Integration by the id (%s), status code: %d, %s",
				data.ID.String(),
				httpResp.HTTPResponse.StatusCode,
				err.Error(),
			),
		)
		return
	}
}

// BuildUpdateBodyFromPlan constructs the full IntegrationsUpdateBodySchema from the base plan.
func BuildUpdateBodyFromPlan(
	ctx context.Context,
	data BaseIntegrationResourceModel,
	applicationName applicationName,
	parsedConnectionJson *map[string]interface{},
) (client.IntegrationsUpdateBodySchema, diag.Diagnostics) {
	var diags diag.Diagnostics

	if vDiags := ValidateVirtualApplicationConstraints(data, applicationName); vDiags.HasError() {
		diags.Append(vDiags...)
		return client.IntegrationsUpdateBodySchema{}, diags
	}

	var allowedDurations *[]client.EnumAllowedDurations
	aDurations, aDiags := utils.GetEnumAllowedDurationsSliceFromNumberSet(ctx, data.AllowedDurations)
	if aDiags.HasError() {
		diags.Append(aDiags...)
		return client.IntegrationsUpdateBodySchema{}, diags
	}
	if aDurations != nil {
		allowedDurations = &aDurations
	}

	var workflow client.IdParamsSchema
	if data.Workflow.ID.ValueString() != "" {
		id, err := uuid.Parse(data.Workflow.ID.ValueString())
		if err != nil {
			diags.AddError("Client Error", fmt.Sprintf("Failed to parse given workflow id to UUID, got error: %v", err))
			return client.IntegrationsUpdateBodySchema{}, diags
		}
		workflow.Id = id
	}

	maintainers, mDiags := BuildUpdateMaintainersFromPlan(data.Maintainers)
	if mDiags.HasError() {
		diags.Append(mDiags...)
		return client.IntegrationsUpdateBodySchema{}, diags
	}

	prereqs, pDiags := BuildUpdatePrerequisitePermissionsFromPlan(data.PrerequisitePermissions)
	if pDiags.HasError() {
		diags.Append(pDiags...)
		return client.IntegrationsUpdateBodySchema{}, diags
	}

	return client.IntegrationsUpdateBodySchema{
		AllowedDurations:                     allowedDurations,
		AutoAssignRecommendedMaintainers:     utils.BoolPointer(data.AutoAssignRecommendedMaintainers.ValueBool()),
		AutoAssignRecommendedOwners:          utils.BoolPointer(data.AutoAssignRecommendedOwners.ValueBool()),
		ConnectionJson:                       parsedConnectionJson,
		Maintainers:                          &maintainers,
		Name:                                 utils.StringPointer(data.Name.ValueString()),
		NotifyAboutExternalPermissionChanges: utils.BoolPointer(data.NotifyAboutExternalPermissionChanges.ValueBool()),
		Owner:                                &client.UserEntitySchema{Id: data.Owner.Id.ValueString()},
		Workflow:                             &workflow,
		PrerequisitePermissions:              prereqs,
		Requestable:                          data.Requestable.ValueBoolPointer(),
		RequestableByDefault:                 data.RequestableByDefault.ValueBoolPointer(),
	}, diags
}

// BuildCreateBodyFromPlan constructs the full IntegrationCreateBodySchema from the base plan.
// parseConnectionJson is called with connectionJson and the result is set on the returned body,
// so the caller only needs a single call to obtain a ready-to-send body.
func BuildCreateBodyFromPlan(
	ctx context.Context,
	plan BaseIntegrationResourceModel,
	appName applicationName,
	parsedConnectionJson *map[string]interface{},
) (client.IntegrationCreateBodySchema, diag.Diagnostics) {
	var diags diag.Diagnostics

	if vDiags := ValidateVirtualApplicationConstraints(plan, appName); vDiags.HasError() {
		diags.Append(vDiags...)
		return client.IntegrationCreateBodySchema{}, diags
	}

	var allowedDurations *[]client.EnumAllowedDurations
	aDurations, aDiags := utils.GetEnumAllowedDurationsSliceFromNumberSet(ctx, plan.AllowedDurations)
	if aDiags.HasError() {
		diags.Append(aDiags...)
		return client.IntegrationCreateBodySchema{}, diags
	}
	if aDurations != nil {
		allowedDurations = &aDurations
	}

	var agentToken *client.NameSchema
	if plan.AgentToken != nil {
		agentToken = &client.NameSchema{Name: plan.AgentToken.Name.ValueString()}
	}

	maintainers, mDiags := BuildCreateMaintainersFromPlan(plan.Maintainers)
	if mDiags.HasError() {
		diags.Append(mDiags...)
		return client.IntegrationCreateBodySchema{}, diags
	}

	prereqs, pDiags := BuildCreatePrerequisitePermissionsFromPlan(plan.PrerequisitePermissions)
	if pDiags.HasError() {
		diags.Append(pDiags...)
		return client.IntegrationCreateBodySchema{}, diags
	}

	workflowID, err := uuid.Parse(plan.Workflow.ID.ValueString())
	if err != nil {
		diags.AddError("Client Error", fmt.Sprintf("Failed to parse given workflow id to UUID, got error: %v", err))

		return client.IntegrationCreateBodySchema{}, diags
	}

	return client.IntegrationCreateBodySchema{
		AgentToken:                           agentToken,
		AllowChangingAccountPermissions:      plan.AllowChangingAccountPermissions.ValueBool(),
		AllowCreatingAccounts:                plan.AllowCreatingAccounts.ValueBool(),
		Requestable:                          plan.Requestable.ValueBoolPointer(),
		RequestableByDefault:                 plan.RequestableByDefault.ValueBoolPointer(),
		AllowedDurations:                     allowedDurations,
		Application:                          client.NameSchema{Name: appName.String()},
		AutoAssignRecommendedMaintainers:     plan.AutoAssignRecommendedMaintainers.ValueBool(),
		AutoAssignRecommendedOwners:          plan.AutoAssignRecommendedOwners.ValueBool(),
		ConnectionJson:                       parsedConnectionJson,
		Maintainers:                          &maintainers,
		Name:                                 plan.Name.ValueString(),
		NotifyAboutExternalPermissionChanges: plan.NotifyAboutExternalPermissionChanges.ValueBool(),
		Owner:                                client.UserEntitySchema{Id: utils.TrimPrefixSuffix(plan.Owner.Id.ValueString())},
		Readonly:                             plan.Readonly.ValueBool(),
		Workflow:                             client.IdParamsSchema{Id: workflowID},
		PrerequisitePermissions:              prereqs,
	}, diags
}

// ConvertBaseIntegrationResultToBaseModel converts an API IntegrationResultSchema into a
// BaseIntegrationResourceModel. This is shared across all integration resource types so that
// each resource only needs to handle its own extra fields (e.g. connection_json).
func ConvertBaseIntegrationResultToBaseModel(
	ctx context.Context,
	data *client.IntegrationResultSchema,
	agentTokenName string,
) (BaseIntegrationResourceModel, string, diag.Diagnostics) {
	var diags diag.Diagnostics

	if data == nil {
		diags.AddError("No data", "Failed: the given schema data is nil")
		return BaseIntegrationResourceModel{}, "", diags
	}

	allowedDurationsValues, advDiags := utils.GetNumberSetFromAllowedDurations(data.AllowedDurations)
	if advDiags.HasError() {
		diags.Append(advDiags...)
		return BaseIntegrationResourceModel{}, "", diags
	}

	marshalJSON, err := data.Owner.Email.MarshalJSON()
	if err != nil {
		diags.AddError("No data", fmt.Sprintf("failed to marshal owner email, error: %v", err))
		return BaseIntegrationResourceModel{}, "", diags
	}

	maintainers, maintainerDiags := utils.GetMaintainers(ctx, data.Maintainers)
	if maintainerDiags.HasError() {
		diags.Append(maintainerDiags...)
		return BaseIntegrationResourceModel{}, "", diags
	}

	var agentToken *utils.NameModel
	if len(agentTokenName) != 0 {
		agentToken = &utils.NameModel{Name: utils.TrimmedStringValue(agentTokenName)}
	}

	var prerequisitePermissions []utils.PrerequisitePermissionModel
	if data.PrerequisitePermissions != nil {
		for _, pp := range *data.PrerequisitePermissions {
			for _, item := range pp {
				v, err := item.AsPrerequisiteRolePermissionResponseSchema()
				if err != nil {
					diags.AddError(
						"No data",
						fmt.Sprintf("failed to unmarshal the prerequisite permissions data, err: %s", err.Error()),
					)
					return BaseIntegrationResourceModel{}, "", diags
				}

				roleModel, diagsGetRoles := utils.GetRole(ctx, v.Role.Id.String(), v.Role.Name, v.Role.Resource)
				if diagsGetRoles.HasError() {
					diags.Append(diagsGetRoles...)
					return BaseIntegrationResourceModel{}, "", diags
				}

				prerequisitePermissions = append(prerequisitePermissions,
					utils.PrerequisitePermissionModel{
						Default: types.BoolValue(v.Default),
						Role:    roleModel,
					},
				)
			}
		}
	}

	if len(maintainers) == 0 {
		maintainers = nil
	}

	return BaseIntegrationResourceModel{
		ID:                                   utils.TrimmedStringValue(data.Id.String()),
		Name:                                 utils.TrimmedStringValue(data.Name),
		AllowedDurations:                     allowedDurationsValues,
		AllowChangingAccountPermissions:      types.BoolValue(data.AllowChangingAccountPermissions),
		AllowCreatingAccounts:                types.BoolValue(data.AllowCreatingAccounts),
		Readonly:                             types.BoolValue(data.Readonly),
		Requestable:                          types.BoolValue(data.Requestable),
		RequestableByDefault:                 types.BoolValue(data.RequestableByDefault),
		AutoAssignRecommendedMaintainers:     types.BoolValue(data.AutoAssignRecommendedMaintainers),
		AutoAssignRecommendedOwners:          types.BoolValue(data.AutoAssignRecommendedOwners),
		NotifyAboutExternalPermissionChanges: types.BoolValue(data.NotifyAboutExternalPermissionChanges),
		Owner: &utils.IdEmailModel{
			Id:    utils.TrimmedStringValue(data.Owner.Id.String()),
			Email: utils.TrimmedStringValue(string(marshalJSON)),
		},
		AgentToken: agentToken,
		Workflow: &utils.IdNameModel{
			ID:   utils.TrimmedStringValue(data.Workflow.Id.String()),
			Name: utils.TrimmedStringValue(data.Workflow.Name),
		},
		Maintainers:             maintainers,
		PrerequisitePermissions: prerequisitePermissions,
	}, strings.ToLower(data.Application.Name), diags
}

// ValidateVirtualApplicationConstraints checks that virtual application integrations do not
// have settings that are incompatible with their type. Returns diagnostics with errors if any
// constraint is violated. Call this in Create (virtual apps cannot be created from Terraform
// with connection_json, and several flags must hold specific values).
func ValidateVirtualApplicationConstraints(baseData BaseIntegrationResourceModel, appName applicationName) diag.Diagnostics {
	var diags diag.Diagnostics

	if appName != applicationVirtual {
		return diags
	}

	if baseData.NotifyAboutExternalPermissionChanges.ValueBool() {
		diags.AddError("Client Error", "Virtual integrations cannot set notifyAboutExternalPermissions to true")
	}
	if baseData.AutoAssignRecommendedMaintainers.ValueBool() {
		diags.AddError("Client Error", "Virtual integrations cannot set autoAssignRecommendedResourceMaintainers to true")
	}
	if baseData.AutoAssignRecommendedOwners.ValueBool() {
		diags.AddError("Client Error", "Virtual integrations cannot set autoAssignRecommendedResourceOwner to true")
	}
	if baseData.Readonly.ValueBool() {
		diags.AddError("Client Error", "Virtual integrations cannot set readonly to true")
	}
	if !baseData.Requestable.ValueBool() {
		diags.AddError("Client Error", "Virtual integrations cannot set requestable to false")
	}
	if !baseData.RequestableByDefault.ValueBool() {
		diags.AddError("Client Error", "Virtual integrations cannot set requestableByDefault to false")
	}

	return diags
}

// BuildCreateMaintainersFromPlan converts plan maintainer models into the API create body items.
func BuildCreateMaintainersFromPlan(
	plan []*utils.MaintainerModel,
) ([]client.IntegrationCreateBodySchema_Maintainers_Item, diag.Diagnostics) {
	var diags diag.Diagnostics
	maintainers := make([]client.IntegrationCreateBodySchema_Maintainers_Item, 0)

	for _, maintainer := range plan {
		if maintainer.Type.IsNull() || maintainer.Type.IsUnknown() {
			continue
		}

		if maintainer.Entity.IsNull() {
			diags.AddError("Client Error", "Missing data for entity maintainer")
			return nil, diags
		}

		entityID, ok := extractMaintainerEntityID(maintainer, &diags)
		if !ok {
			return nil, diags
		}

		switch maintainer.Type.ValueString() {
		case utils.MaintainerTypeUser:
			item := client.IntegrationCreateBodySchema_Maintainers_Item{}
			if err := item.MergeUserMaintainerSchema(client.UserMaintainerSchema{
				Type: client.EnumMaintainerTypeUserUser,
				User: client.UserEntitySchema{Id: entityID},
			}); err != nil {
				diags.AddError("Client Error", fmt.Sprintf("Failed to merge user maintainer data, error: %v", err))
				return nil, diags
			}
			maintainers = append(maintainers, item)

		case utils.MaintainerTypeGroup:
			item := client.IntegrationCreateBodySchema_Maintainers_Item{}
			if err := item.MergeGroupMaintainerSchema(client.GroupMaintainerSchema{
				Type:  client.EnumMaintainerTypeGroupGroup,
				Group: client.GroupEntitySchema{Id: entityID},
			}); err != nil {
				diags.AddError("Client Error", "Failed to merge group maintainer")
				return nil, diags
			}
			maintainers = append(maintainers, item)

		default:
			diags.AddError("Client Error", "Invalid maintainer type only support user and group")
			return nil, diags
		}
	}

	return maintainers, diags
}

// BuildUpdateMaintainersFromPlan converts plan maintainer models into the API update body items.
func BuildUpdateMaintainersFromPlan(
	plan []*utils.MaintainerModel,
) ([]client.IntegrationsUpdateBodySchema_Maintainers_Item, diag.Diagnostics) {
	var diags diag.Diagnostics
	maintainers := make([]client.IntegrationsUpdateBodySchema_Maintainers_Item, 0)

	for _, maintainer := range plan {
		if maintainer.Type.IsNull() || maintainer.Type.IsUnknown() {
			continue
		}

		if maintainer.Entity.IsNull() {
			diags.AddError("Client Error", "Missing data for entity maintainer")
			return nil, diags
		}

		entityID, ok := extractMaintainerEntityID(maintainer, &diags)
		if !ok {
			return nil, diags
		}

		switch maintainer.Type.ValueString() {
		case utils.MaintainerTypeUser:
			item := client.IntegrationsUpdateBodySchema_Maintainers_Item{}
			if err := item.MergeUserMaintainerSchema(client.UserMaintainerSchema{
				Type: client.EnumMaintainerTypeUserUser,
				User: client.UserEntitySchema{Id: entityID},
			}); err != nil {
				diags.AddError("Client Error", fmt.Sprintf("Failed to merge user maintainer data, error: %v", err))
				return nil, diags
			}
			maintainers = append(maintainers, item)

		case utils.MaintainerTypeGroup:
			item := client.IntegrationsUpdateBodySchema_Maintainers_Item{}
			if err := item.MergeGroupMaintainerSchema(client.GroupMaintainerSchema{
				Type:  client.EnumMaintainerTypeGroupGroup,
				Group: client.GroupEntitySchema{Id: entityID},
			}); err != nil {
				diags.AddError("Client Error", "Failed to merge group maintainer")
				return nil, diags
			}
			maintainers = append(maintainers, item)

		default:
			diags.AddError("Client Error", "Invalid maintainer type only support user and group")
			return nil, diags
		}
	}

	return maintainers, diags
}

// BuildCreatePrerequisitePermissionsFromPlan converts plan prerequisite permissions into API create body items.
// Returns nil if the plan slice is empty.
func BuildCreatePrerequisitePermissionsFromPlan(
	plan []utils.PrerequisitePermissionModel,
) (*[][]client.IntegrationCreateBodySchema_PrerequisitePermissions_Item, diag.Diagnostics) {
	var diags diag.Diagnostics

	if len(plan) == 0 {
		return nil, diags
	}

	ppData := make([][]client.IntegrationCreateBodySchema_PrerequisitePermissions_Item, 0, len(plan))
	for _, pp := range plan {
		if pp.Role.ID.IsNull() || pp.Role.ID.IsUnknown() {
			continue
		}

		item := client.IntegrationCreateBodySchema_PrerequisitePermissions_Item{}
		if err := item.MergePrerequisitePermissionCreateBodySchema(client.PrerequisitePermissionCreateBodySchema{
			Default: pp.Default.ValueBool(),
			Role:    map[string]interface{}{"id": pp.Role.ID.ValueString()},
		}); err != nil {
			diags.AddError("Client Error", fmt.Sprintf("Failed to merge prerequisite permission data, error: %v", err))
			return nil, diags
		}

		ppData = append(ppData, []client.IntegrationCreateBodySchema_PrerequisitePermissions_Item{item})
	}

	return &ppData, diags
}

// BuildUpdatePrerequisitePermissionsFromPlan converts plan prerequisite permissions into API update body items.
// Returns nil if the plan slice is empty.
func BuildUpdatePrerequisitePermissionsFromPlan(
	plan []utils.PrerequisitePermissionModel,
) (*[][]client.IntegrationsUpdateBodySchema_PrerequisitePermissions_Item, diag.Diagnostics) {
	var diags diag.Diagnostics

	if len(plan) == 0 {
		return nil, diags
	}

	ppData := make([][]client.IntegrationsUpdateBodySchema_PrerequisitePermissions_Item, 0, len(plan))
	for _, pp := range plan {
		if pp.Role.ID.IsNull() || pp.Role.ID.IsUnknown() {
			continue
		}

		item := client.IntegrationsUpdateBodySchema_PrerequisitePermissions_Item{}
		if err := item.MergePrerequisitePermissionCreateBodySchema(client.PrerequisitePermissionCreateBodySchema{
			Default: pp.Default.ValueBool(),
			Role:    map[string]interface{}{"id": pp.Role.ID.ValueString()},
		}); err != nil {
			diags.AddError("Client Error", fmt.Sprintf("Failed to merge prerequisite permission data, error: %v", err))
			return nil, diags
		}

		ppData = append(ppData, []client.IntegrationsUpdateBodySchema_PrerequisitePermissions_Item{item})
	}

	return &ppData, diags
}

// extractMaintainerEntityID pulls the "id" string out of a MaintainerModel's entity object.
// It adds an error to diags and returns false if the extraction fails.
func extractMaintainerEntityID(maintainer *utils.MaintainerModel, diags *diag.Diagnostics) (string, bool) {
	idAttr := maintainer.Entity.Attributes()["id"]
	strVal, ok := idAttr.(basetypes.StringValue)
	if !ok {
		diags.AddError("Client Error", "Missing data for entity maintainer id")
		return "", false
	}
	return strVal.ValueString(), true
}

// ParseConnectionJson validates and parses a connection_json string into the typed map form
// expected by the API. Returns nil and error diagnostics if the value is absent or not valid JSON.
func ParseConnectionJson(v string) (map[string]interface{}, diag.Diagnostics) {
	var diags diag.Diagnostics

	if v == "" {
		return nil, diags
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(v), &data); err != nil {
		diags.AddError(
			"Client Error",
			fmt.Sprintf("Failed to parse given connection json to json, %s, error: %v", v, err),
		)
		return nil, diags
	}

	return data, diags
}
