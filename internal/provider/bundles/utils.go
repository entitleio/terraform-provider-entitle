package bundles

import (
	"context"

	"github.com/entitleio/terraform-provider-entitle/internal/client"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/utils"
	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// getWorkflow extracts and converts workflow information from the API response into an IdNameModel.
//
// It returns the IdNameModel containing the workflow ID and Name.
// If the API response does not contain a valid ID, it returns nil.
func getWorkflow(data client.WorkflowResponseSchema) *utils.IdNameModel {
	// Extract and convert the workflow information from the API response
	if len(data.Id.String()) > 0 {
		return &utils.IdNameModel{
			ID:   utils.TrimmedStringValue(data.Id.String()),
			Name: utils.TrimmedStringValue(data.Name),
		}
	}

	return nil
}

func getRolesAsPlanned(
	ctx context.Context,
	planRoles []client.IdParamsSchema,
	data []client.BundleItemResponseSchema,
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

// getRoles processes and converts a slice of bundle item responses into bundle role models.
//
// It returns a slice of bundleRoleModel structs containing role information.
// The function also returns a diag.Diagnostics in case of any errors during conversion.
func getRoles(
	ctx context.Context,
	data []client.BundleItemResponseSchema,
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
