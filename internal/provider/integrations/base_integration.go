package integrations

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
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

	result, _, rDiags := ConvertBaseIntegrationResultToBaseModel(ctx, &integrationResp.JSON200.Result)
	diags.Append(rDiags...)
	if diags.HasError() {
		return BaseIntegrationResourceModel{}, diags
	}

	tflog.Trace(ctx, "Created a entitle integration resource")
	return result, diags
}

func UpdateIntegration(
	ctx context.Context,
	cli *client.ClientWithResponses,
	base BaseIntegrationResourceModel,
	appName applicationName,
	parsedConnectionJson *map[string]interface{},
	resp *resource.UpdateResponse,
) *BaseIntegrationResourceModel {
	uid, err := uuid.Parse(base.ID.String())
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Failed to parse the given id to UUID format, got error: %v", err),
		)
		return nil
	}

	body, bDiags := BuildUpdateBodyFromPlan(ctx, base, appName, parsedConnectionJson)
	resp.Diagnostics.Append(bDiags...)
	if resp.Diagnostics.HasError() {
		return nil
	}

	integrationResp, err := cli.IntegrationsUpdateWithResponse(ctx, uid, body)
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiConnection.Error(),
			fmt.Sprintf("Unable to update the integration by the id (%s), got error: %s", uid.String(), err),
		)
		return nil
	}

	if err = utils.HTTPResponseToError(integrationResp.HTTPResponse.StatusCode, integrationResp.Body); err != nil {
		if errors.Is(err, utils.ErrNotFound) {
			tflog.Debug(ctx, "Resource no longer exists, removing from state")
			resp.State.RemoveResource(ctx)
			return nil
		}
		resp.Diagnostics.AddError(
			utils.ErrApiResponse.Error(),
			fmt.Sprintf(
				"Failed to update the Integration by the id (%s), status code: %d, %s",
				uid.String(),
				integrationResp.HTTPResponse.StatusCode,
				err.Error(),
			),
		)
		return nil
	}

	result, _, rDiags := ConvertBaseIntegrationResultToBaseModel(ctx, &integrationResp.JSON200.Result)
	resp.Diagnostics.Append(rDiags...)
	if resp.Diagnostics.HasError() {
		return nil
	}

	return &result
}

func ReadIntegration(
	ctx context.Context,
	cli *client.ClientWithResponses,
	base BaseIntegrationResourceModel,
	resp *resource.ReadResponse,
) (BaseIntegrationResourceModel, string, bool) {
	uid, err := uuid.Parse(base.ID.String())
	if err != nil {
		resp.Diagnostics.AddError(
			"Client Error",
			fmt.Sprintf("Failed to parse the resource id (%s) to UUID, got error: %s", base.ID.String(), err),
		)
		return BaseIntegrationResourceModel{}, "", false
	}

	integrationResp, err := cli.IntegrationsShowWithResponse(ctx, uid)
	if err != nil {
		resp.Diagnostics.AddError(
			utils.ErrApiConnection.Error(),
			fmt.Sprintf("Unable to get the integration by the id (%s), got error: %s", uid.String(), err),
		)
		return BaseIntegrationResourceModel{}, "", false
	}

	if err = utils.HTTPResponseToError(integrationResp.HTTPResponse.StatusCode, integrationResp.Body); err != nil {
		if errors.Is(err, utils.ErrNotFound) {
			tflog.Debug(ctx, "Resource no longer exists, removing from state")
			resp.State.RemoveResource(ctx)
			return BaseIntegrationResourceModel{}, "", false
		}
		resp.Diagnostics.AddError(
			utils.ErrApiResponse.Error(),
			fmt.Sprintf(
				"Failed to get the Integration by the id (%s), status code: %d, %s",
				uid.String(),
				integrationResp.HTTPResponse.StatusCode,
				err.Error(),
			),
		)
		return BaseIntegrationResourceModel{}, "", false
	}

	result, appName, rDiags := ConvertBaseIntegrationResultToBaseModel(ctx, &integrationResp.JSON200.Result)
	resp.Diagnostics.Append(rDiags...)
	if resp.Diagnostics.HasError() {
		return BaseIntegrationResourceModel{}, "", false
	}

	return result, appName, true
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
	if data.WorkflowID.ValueString() != "" {
		id, err := uuid.Parse(data.WorkflowID.ValueString())
		if err != nil {
			diags.AddError(
				"Client Error",
				fmt.Sprintf("Failed to parse given workflow id to UUID, got error: %v", err),
			)
			return client.IntegrationsUpdateBodySchema{}, diags
		}
		workflow.Id = id
	}

	maintainers, mDiags := BuildUpdateMaintainersFromPlan(ctx, data.Maintainers)
	if mDiags.HasError() {
		diags.Append(mDiags...)
		return client.IntegrationsUpdateBodySchema{}, diags
	}

	prereqs, pDiags := BuildUpdatePrerequisitePermissionsFromPlan(ctx, data.PrerequisitePermissions)
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
		Owner:                                &client.UserEntitySchema{Id: data.OwnerID.ValueString()},
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
	if plan.AgentToken.ValueStringPointer() != nil {
		agentToken = &client.NameSchema{Name: plan.AgentToken.ValueString()}
	}

	maintainers, mDiags := BuildCreateMaintainersFromPlan(ctx, plan.Maintainers)
	if mDiags.HasError() {
		diags.Append(mDiags...)
		return client.IntegrationCreateBodySchema{}, diags
	}

	prereqs, pDiags := BuildCreatePrerequisitePermissionsFromPlan(ctx, plan.PrerequisitePermissions)
	if pDiags.HasError() {
		diags.Append(pDiags...)
		return client.IntegrationCreateBodySchema{}, diags
	}

	workflowID, err := uuid.Parse(plan.WorkflowID.ValueString())
	if err != nil {
		diags.AddError("Client Error", fmt.Sprintf("Failed to parse given workflow id to UUID, got error: %v", err))

		return client.IntegrationCreateBodySchema{}, diags
	}

	return client.IntegrationCreateBodySchema{
		AgentToken:                           agentToken,
		AllowChangingAccountPermissions:      utils.BoolOrDefault(plan.AllowChangingAccountPermissions, defaultIntegrationAllowChangingAccountPermissions),
		AllowCreatingAccounts:                utils.BoolOrDefault(plan.AllowCreatingAccounts, defaultIntegrationAllowCreatingAccounts),
		Requestable:                          utils.BoolPtrOrDefault(plan.Requestable, defaultIntegrationAllowRequests),
		RequestableByDefault:                 utils.BoolPtrOrDefault(plan.RequestableByDefault, defaultIntegrationAllowRequestsByDefault),
		AllowedDurations:                     allowedDurations,
		Application:                          client.NameSchema{Name: appName.String()},
		AutoAssignRecommendedMaintainers:     utils.BoolOrDefault(plan.AutoAssignRecommendedMaintainers, defaultIntegrationAutoAssignRecommendedMaintainers),
		AutoAssignRecommendedOwners:          utils.BoolOrDefault(plan.AutoAssignRecommendedOwners, defaultIntegrationAutoAssignRecommendedOwners),
		ConnectionJson:                       parsedConnectionJson,
		Maintainers:                          &maintainers,
		Name:                                 plan.Name.ValueString(),
		NotifyAboutExternalPermissionChanges: utils.BoolOrDefault(plan.NotifyAboutExternalPermissionChanges, defaultIntegrationNotifyAboutExternalPermissionChanges),
		Owner:                                client.UserEntitySchema{Id: utils.TrimPrefixSuffix(plan.OwnerID.ValueString())},
		Readonly:                             utils.BoolOrDefault(plan.Readonly, defaultIntegrationReadonly),
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

	maintainerModels, maintainerDiags := getIntegrationMaintainers(data.Maintainers)
	if maintainerDiags.HasError() {
		diags.Append(maintainerDiags...)
		return BaseIntegrationResourceModel{}, "", diags
	}

	prerequisitePermissionElementType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"default": types.BoolType,
			"role_id": types.StringType,
		},
	}

	prerequisitePermissionModels := make([]utils.ResourcePrerequisitePermissionModel, 0)
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

				prerequisitePermissionModels = append(prerequisitePermissionModels,
					utils.ResourcePrerequisitePermissionModel{
						Default: types.BoolValue(v.Default),
						RoleID:  utils.TrimmedStringValue(v.Role.Id.String()),
					},
				)
			}
		}
	}

	var prerequisitePermissionsSet types.Set
	if len(prerequisitePermissionModels) == 0 {
		prerequisitePermissionsSet = types.SetNull(prerequisitePermissionElementType)
	} else {
		var ppDiags diag.Diagnostics
		prerequisitePermissionsSet, ppDiags = types.SetValueFrom(ctx, prerequisitePermissionElementType, prerequisitePermissionModels)
		if ppDiags.HasError() {
			diags.Append(ppDiags...)
			return BaseIntegrationResourceModel{}, "", diags
		}
	}

	maintainerElementType := types.ObjectType{
		AttrTypes: map[string]attr.Type{
			"type": types.StringType,
			"id":   types.StringType,
		},
	}

	var maintainersSet types.Set
	if len(maintainerModels) == 0 {
		maintainersSet = types.SetNull(maintainerElementType)
	} else {
		var setDiags diag.Diagnostics
		maintainersSet, setDiags = types.SetValueFrom(ctx, maintainerElementType, maintainerModels)
		if setDiags.HasError() {
			diags.Append(setDiags...)
			return BaseIntegrationResourceModel{}, "", diags
		}
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
		OwnerID:                              utils.TrimmedStringNullValue(data.Owner.Id.String()),
		WorkflowID:                           utils.TrimmedStringNullValue(data.Workflow.Id.String()),
		Maintainers:                          maintainersSet,
		PrerequisitePermissions:              prerequisitePermissionsSet,
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

	// Use the same default fallback as BuildCreateBodyFromPlan/BuildUpdateBodyFromPlan
	// so that an attribute the user left unset is validated against the value that
	// will actually be sent/kept, not against the zero value of a null/unknown Bool.
	if utils.BoolOrDefault(baseData.NotifyAboutExternalPermissionChanges, defaultIntegrationNotifyAboutExternalPermissionChanges) {
		diags.AddError("Client Error", "Virtual integrations cannot set notifyAboutExternalPermissions to true")
	}
	if utils.BoolOrDefault(baseData.AutoAssignRecommendedMaintainers, defaultIntegrationAutoAssignRecommendedMaintainers) {
		diags.AddError("Client Error", "Virtual integrations cannot set autoAssignRecommendedResourceMaintainers to true")
	}
	if utils.BoolOrDefault(baseData.AutoAssignRecommendedOwners, defaultIntegrationAutoAssignRecommendedOwners) {
		diags.AddError("Client Error", "Virtual integrations cannot set autoAssignRecommendedResourceOwner to true")
	}
	if utils.BoolOrDefault(baseData.Readonly, defaultIntegrationReadonly) {
		diags.AddError("Client Error", "Virtual integrations cannot set readonly to true")
	}
	if !utils.BoolOrDefault(baseData.Requestable, defaultIntegrationAllowRequests) {
		diags.AddError("Client Error", "Virtual integrations cannot set requestable to false")
	}
	if !utils.BoolOrDefault(baseData.RequestableByDefault, defaultIntegrationAllowRequestsByDefault) {
		diags.AddError("Client Error", "Virtual integrations cannot set requestableByDefault to false")
	}

	return diags
}

// maintainerItemPtr is the pointer constraint used by buildMaintainersFromPlan. It requires
// that *T exposes the two Merge methods that all generated maintainer union types share.
type maintainerItemPtr[T any] interface {
	*T
	MergeUserMaintainerSchema(client.UserMaintainerSchema) error
	MergeGroupMaintainerSchema(client.GroupMaintainerSchema) error
}

// buildMaintainersFromPlan is the shared generic core for BuildCreateMaintainersFromPlan
// and BuildUpdateMaintainersFromPlan. T is the value type (e.g.
// IntegrationCreateBodySchema_Maintainers_Item); PT is its pointer (*T), which carries
// the Merge methods.
func buildMaintainersFromPlan[T any, PT maintainerItemPtr[T]](ctx context.Context, plan types.Set) ([]T, diag.Diagnostics) {
	var diags diag.Diagnostics
	maintainers := make([]T, 0)

	if plan.IsNull() || plan.IsUnknown() {
		return maintainers, diags
	}

	var planMaintainerModels []IntegrationMaintainerModel
	if elemDiags := plan.ElementsAs(ctx, &planMaintainerModels, false); elemDiags.HasError() {
		diags.Append(elemDiags...)
		return nil, diags
	}

	for _, maintainer := range planMaintainerModels {
		if maintainer.Type.IsNull() || maintainer.Type.IsUnknown() {
			continue
		}

		if maintainer.ID.IsNull() || maintainer.ID.IsUnknown() {
			diags.AddError("Client Error", "Missing id for maintainer")
			return nil, diags
		}

		var item T
		pt := PT(&item)
		switch maintainer.Type.ValueString() {
		case utils.MaintainerTypeUser:
			if err := pt.MergeUserMaintainerSchema(client.UserMaintainerSchema{
				Type: client.EnumMaintainerTypeUserUser,
				User: client.UserEntitySchema{Id: maintainer.ID.ValueString()},
			}); err != nil {
				diags.AddError("Client Error", fmt.Sprintf("Failed to merge user maintainer data, error: %v", err))
				return nil, diags
			}

		case utils.MaintainerTypeGroup:
			if err := pt.MergeGroupMaintainerSchema(client.GroupMaintainerSchema{
				Type:  client.EnumMaintainerTypeGroupGroup,
				Group: client.GroupEntitySchema{Id: maintainer.ID.ValueString()},
			}); err != nil {
				diags.AddError("Client Error", "Failed to merge group maintainer")
				return nil, diags
			}

		default:
			diags.AddError("Client Error", "Invalid maintainer type only support user and group")
			return nil, diags
		}

		maintainers = append(maintainers, item)
	}

	return maintainers, diags
}

// BuildCreateMaintainersFromPlan converts plan maintainer models into the API create body items.
func BuildCreateMaintainersFromPlan(
	ctx context.Context,
	plan types.Set,
) ([]client.IntegrationCreateBodySchema_Maintainers_Item, diag.Diagnostics) {
	return buildMaintainersFromPlan[
		client.IntegrationCreateBodySchema_Maintainers_Item,
		*client.IntegrationCreateBodySchema_Maintainers_Item,
	](ctx, plan)
}

// BuildUpdateMaintainersFromPlan converts plan maintainer models into the API update body items.
func BuildUpdateMaintainersFromPlan(
	ctx context.Context,
	plan types.Set,
) ([]client.IntegrationsUpdateBodySchema_Maintainers_Item, diag.Diagnostics) {
	return buildMaintainersFromPlan[
		client.IntegrationsUpdateBodySchema_Maintainers_Item,
		*client.IntegrationsUpdateBodySchema_Maintainers_Item,
	](ctx, plan)
}

// prereqPermItemPtr is the pointer constraint used by buildPrerequisitePermissionsFromPlan.
// It requires that *T exposes MergePrerequisitePermissionCreateBodySchema, which all
// generated prerequisite-permission union types share.
type prereqPermItemPtr[T any] interface {
	*T
	MergePrerequisitePermissionCreateBodySchema(client.PrerequisitePermissionCreateBodySchema) error
}

// buildPrerequisitePermissionsFromPlan is the shared generic core for
// BuildCreatePrerequisitePermissionsFromPlan and BuildUpdatePrerequisitePermissionsFromPlan.
// T is the value type; PT is its pointer (*T), which carries the Merge method.
func buildPrerequisitePermissionsFromPlan[T any, PT prereqPermItemPtr[T]](
	ctx context.Context,
	plan types.Set,
) (*[][]T, diag.Diagnostics) {
	var diags diag.Diagnostics

	if plan.IsNull() || plan.IsUnknown() {
		return nil, diags
	}

	var planModels []utils.ResourcePrerequisitePermissionModel
	if elemDiags := plan.ElementsAs(ctx, &planModels, false); elemDiags.HasError() {
		diags.Append(elemDiags...)
		return nil, diags
	}

	if len(planModels) == 0 {
		return nil, diags
	}

	ppData := make([][]T, 0, len(planModels))
	for _, pp := range planModels {
		if pp.RoleID.IsNull() || pp.RoleID.IsUnknown() {
			continue
		}

		var item T
		if err := PT(&item).MergePrerequisitePermissionCreateBodySchema(client.PrerequisitePermissionCreateBodySchema{
			Default: pp.Default.ValueBool(),
			Role:    map[string]interface{}{"id": pp.RoleID.ValueString()},
		}); err != nil {
			diags.AddError("Client Error", fmt.Sprintf("Failed to merge prerequisite permission data, error: %v", err))
			return nil, diags
		}

		ppData = append(ppData, []T{item})
	}

	return &ppData, diags
}

// BuildCreatePrerequisitePermissionsFromPlan converts plan prerequisite permissions into API create body items.
// Returns nil if the plan set is null, unknown, or empty.
func BuildCreatePrerequisitePermissionsFromPlan(
	ctx context.Context,
	plan types.Set,
) (*[][]client.IntegrationCreateBodySchema_PrerequisitePermissions_Item, diag.Diagnostics) {
	return buildPrerequisitePermissionsFromPlan[
		client.IntegrationCreateBodySchema_PrerequisitePermissions_Item,
		*client.IntegrationCreateBodySchema_PrerequisitePermissions_Item,
	](ctx, plan)
}

// BuildUpdatePrerequisitePermissionsFromPlan converts plan prerequisite permissions into API update body items.
// Returns nil if the plan set is null, unknown, or empty.
func BuildUpdatePrerequisitePermissionsFromPlan(
	ctx context.Context,
	plan types.Set,
) (*[][]client.IntegrationsUpdateBodySchema_PrerequisitePermissions_Item, diag.Diagnostics) {
	return buildPrerequisitePermissionsFromPlan[
		client.IntegrationsUpdateBodySchema_PrerequisitePermissions_Item,
		*client.IntegrationsUpdateBodySchema_PrerequisitePermissions_Item,
	](ctx, plan)
}

// getIntegrationMaintainers converts the API's maintainer union list into the flattened
// IntegrationMaintainerModel used by BaseIntegrationResourceModel ({type, id} per entry),
// instead of the shared utils.MaintainerModel type+entity shape used elsewhere.
func getIntegrationMaintainers[T utils.MaintainerInterface](
	maintainers []T,
) ([]IntegrationMaintainerModel, diag.Diagnostics) {
	var diags diag.Diagnostics
	result := make([]IntegrationMaintainerModel, 0, len(maintainers))

	for _, item := range maintainers {
		data, err := item.MarshalJSON()
		if err != nil {
			diags.AddError("Failed to marshal maintainer data", err.Error())
			return nil, diags
		}

		var body utils.MaintainerCommonResponseSchema
		if err := json.Unmarshal(data, &body); err != nil {
			diags.AddError(
				fmt.Sprintf("Failed to unmarshal the maintainer data (%s)", data),
				err.Error(),
			)
			return nil, diags
		}

		switch strings.ToLower(body.Type) {
		case utils.MaintainerTypeUser:
			responseSchema, err := item.AsMaintainerUserResponseSchema()
			if err != nil {
				diags.AddError(
					"No data",
					fmt.Sprintf("failed to convert response schema to user response schema, error: %v", err),
				)
				return nil, diags
			}

			result = append(result, IntegrationMaintainerModel{
				Type: utils.TrimmedStringValue(body.Type),
				ID:   utils.TrimmedStringValue(responseSchema.User.Id.String()),
			})
		case utils.MaintainerTypeGroup:
			responseSchema, err := item.AsMaintainerGroupResponseSchema()
			if err != nil {
				diags.AddError(
					"No data",
					fmt.Sprintf("failed to convert response schema to group response schema, error: %v", err),
				)
				return nil, diags
			}

			result = append(result, IntegrationMaintainerModel{
				Type: utils.TrimmedStringValue(body.Type),
				ID:   utils.TrimmedStringValue(responseSchema.Group.Id.String()),
			})
		default:
			diags.AddError("failed invalid type for maintainer", body.Type)
			return nil, diags
		}
	}

	return result, diags
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
			fmt.Sprintf("Failed to parse connection_json as JSON, error: %v", err),
		)
		return nil, diags
	}

	return data, diags
}

// configureIntegrationResource is the shared Configure implementation for all integration
// resource types. It asserts that ProviderData is a *client.ClientWithResponses and assigns
// it to the given pointer, or adds an error diagnostic if the type is unexpected.
func configureIntegrationResource(
	providerData any,
	target **client.ClientWithResponses,
	diagsOut *diag.Diagnostics,
) {
	if providerData == nil {
		return
	}

	c, ok := providerData.(*client.ClientWithResponses)
	if !ok {
		diagsOut.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *client.ClientWithResponses, got: %T. Please report this issue to the provider developers.", providerData),
		)
		return
	}

	*target = c
}
