package policies

import (
	"context"
	"fmt"

	"github.com/entitleio/terraform-provider-entitle/internal/client"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/utils"
	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/diag"
)

func getBundlesAsPlanned(
	planBundles []client.IdParamsSchema,
	bundles []client.PolicyBundleResponseSchema,
) []*utils.IdNameModel {
	var result []*utils.IdNameModel
	if len(planBundles) > 0 {
		result = make([]*utils.IdNameModel, len(planBundles))

		for index, plan := range planBundles {
			for _, bundle := range bundles {
				if bundle.Id.String() != plan.Id.String() {
					continue
				}

				result[index] = &utils.IdNameModel{
					ID:   utils.TrimmedStringValue(bundle.Id.String()),
					Name: utils.TrimmedStringValue(bundle.Name),
				}
			}
		}
	}

	return result
}

func getRolesAsPlanned(
	ctx context.Context,
	planRoles []client.IdParamsSchema,
	data []client.PolicyRoleResponseSchema,
) ([]*utils.Role, diag.Diagnostics) {
	var diags diag.Diagnostics

	var roles []*utils.Role
	if len(planRoles) > 0 {
		roles = make([]*utils.Role, len(planRoles))

		for index, plan := range planRoles {
			for _, role := range data {
				if role.Id.String() != plan.Id.String() {
					continue
				}

				roleModel, diagsGetRoles := utils.GetRole(ctx, role.Id.String(), role.Name, role.Resource)
				diags.Append(diagsGetRoles...)
				if diags.HasError() {
					return nil, diags
				}

				roles[index] = roleModel
			}
		}
	}

	return roles, diags
}

func getInGroupsAsPlanned(
	planInGroups []client.InGroupSchema,
	inGroups []client.PolicyGroupResponseSchema,
) []*PolicyInGroupModel {
	var result []*PolicyInGroupModel

	if len(planInGroups) > 0 {
		result = make([]*PolicyInGroupModel, len(planInGroups))

		for index, plan := range planInGroups {
			for _, group := range inGroups {
				if group.Id.String() != plan.Id {
					continue
				}

				result[index] = &PolicyInGroupModel{
					ID:   utils.TrimmedStringValue(group.Id.String()),
					Name: utils.TrimmedStringValue(group.Name),
					Type: utils.TrimmedStringValue(string(group.Type)),
				}
			}

		}
	}

	return result
}

// getInGroups converts PolicyGroupResponseSchema slices to PolicyInGroupModel slices.
func getInGroups(
	inGroups []client.PolicyGroupResponseSchema,
) []*PolicyInGroupModel {
	var result []*PolicyInGroupModel
	if len(inGroups) > 0 {
		result = make([]*PolicyInGroupModel, len(inGroups))

		for index, inGroup := range inGroups {
			result[index] = &PolicyInGroupModel{
				ID:   utils.TrimmedStringValue(inGroup.Id.String()),
				Name: utils.TrimmedStringValue(inGroup.Name),
				Type: utils.TrimmedStringValue(string(inGroup.Type)),
			}
		}
	}

	return result
}

// getBundles converts PolicyBundleResponseSchema slices to IdNameModel slices.
func getBundles(
	bundles []client.PolicyBundleResponseSchema,
) []*utils.IdNameModel {
	var result []*utils.IdNameModel
	if len(bundles) > 0 {
		result = make([]*utils.IdNameModel, 0)

		for _, bundle := range bundles {
			result = append(result, &utils.IdNameModel{
				ID:   utils.TrimmedStringValue(bundle.Id.String()),
				Name: utils.TrimmedStringValue(bundle.Name),
			})
		}
	}

	return result
}

// getRoles converts PolicyRoleResponseSchema slices to policyRoleModel slices and handles the resource transformation.
func getRoles(
	ctx context.Context,
	data []client.PolicyRoleResponseSchema,
) ([]*utils.Role, diag.Diagnostics) {
	var diags diag.Diagnostics

	var roles []*utils.Role
	if len(data) > 0 {
		roles = make([]*utils.Role, 0)

		for _, role := range data {
			roleModel, diagsGetRoles := utils.GetRole(ctx, role.Id.String(), role.Name, role.Resource)
			diags.Append(diagsGetRoles...)
			if diags.HasError() {
				return nil, diags
			}

			// Append the role model to the roles slice.
			roles = append(roles, roleModel)
		}
	}

	return roles, diags
}

// getInGroupsFromPlan converts PolicyInGroupModel slices to InGroupSchema slices.
func getInGroupsFromPlan(inGroups []*PolicyInGroupModel) []client.InGroupSchema {
	result := make([]client.InGroupSchema, len(inGroups))
	if len(inGroups) > 0 {
		for index, group := range inGroups {

			groupType := client.EnumPolicyGroupType(group.Type.ValueString())
			if !group.ID.IsNull() && !group.ID.IsUnknown() {
				result[index] = client.InGroupSchema{
					Id:   group.ID.ValueString(),
					Type: groupType,
				}
			}
		}
	}

	return result
}

// getRolesFromPlan converts policyRoleModel slices to IdParamsSchema slices.
func getRolesFromPlan(roles []*utils.Role) ([]client.IdParamsSchema, diag.Diagnostics) {
	var diags diag.Diagnostics
	result := make([]client.IdParamsSchema, 0)

	for _, role := range roles {
		if !role.ID.IsNull() && !role.ID.IsUnknown() {
			parsedUUID, err := uuid.Parse(role.ID.ValueString())
			if err != nil {
				diags.AddError(
					"Client Error",
					fmt.Sprintf(
						"failed to parse the role id (%s) to UUID, got error: %s",
						role.ID.String(),
						err,
					),
				)
				return nil, diags
			}

			result = append(result, client.IdParamsSchema{
				Id: parsedUUID,
			})
		}
	}

	return result, diags
}

// getBundlesFromPlan converts IdNameModel slices to IdParamsSchema slices.
func getBundlesFromPlan(bundles []*utils.IdNameModel) ([]client.IdParamsSchema, diag.Diagnostics) {
	var diags diag.Diagnostics
	result := make([]client.IdParamsSchema, 0)

	for _, bundle := range bundles {
		if !bundle.ID.IsNull() && !bundle.ID.IsUnknown() {
			parsedUUID, err := uuid.Parse(bundle.ID.ValueString())
			if err != nil {
				diags.AddError(
					"Client Error",
					fmt.Sprintf(
						"failed to parse the bundle id (%s) to UUID, got error: %s",
						bundle.ID.String(),
						err,
					),
				)
				return nil, diags
			}

			result = append(result, client.IdParamsSchema{
				Id: parsedUUID,
			})
		}
	}

	return result, diags
}
