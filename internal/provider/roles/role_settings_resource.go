package roles

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/entitleio/terraform-provider-entitle/docs"
	"github.com/entitleio/terraform-provider-entitle/internal/client"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/bulk"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/bulksettings"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/utils"
)

// NewRoleSettingsResource creates the entitle_role_settings resource: the same
// requestable / workflow settings applied to many roles with PATCH /public/v1/roles/bulk.
// The behaviour is shared with entitle_resource_settings and
// entitle_integration_settings (see package bulksettings); roles have no owner
// or maintainers.
func NewRoleSettingsResource() resource.Resource {
	return bulksettings.New(roleSettingsKind())
}

// roleSettingsKind describes entitle_role_settings to the shared settings implementation.
func roleSettingsKind() bulksettings.Kind {
	return bulksettings.Kind{
		TypeSuffix:          "_role_settings",
		IDsAttribute:        "role_ids",
		Entity:              "role",
		Entities:            "roles",
		MarkdownDescription: docs.RoleSettingsResourceMarkdownDescription,
		Fetch:               fetchRoleSettings,
		Submit: func(c *client.ClientWithResponses, u client.BulkSettingsUpdate) bulk.SubmitFunc {
			return func(ctx context.Context, ids []string) (*client.BulkSubmitResponse, error) {
				return c.RolesBulkUpdateWithResponse(ctx, client.BulkUpdateRolesRequest{Ids: ids, Update: u.AsRoleUpdate()})
			}
		},
	}
}

// fetchRoleSettings reads roles (100 per list request) into the common settings view.
func fetchRoleSettings(ctx context.Context, c *client.ClientWithResponses, parallelism int, ids []string) (map[string]*bulksettings.Entity, error) {
	found, err := bulk.FetchByIDs(ctx, parallelism, ids,
		func(ctx context.Context, chunk []string) ([]client.IntegrationResourceRoleListItemResponseSchema, error) {
			resp, err := c.RolesListWithResponse(ctx, client.ListQuery{IDs: chunk, PerPage: client.MaxListPerPage})
			if err != nil {
				return nil, fmt.Errorf("listing roles: %w", err)
			}
			if err := utils.HTTPResponseToError(resp.StatusCode(), resp.Body); err != nil {
				return nil, fmt.Errorf("listing roles: %w", err)
			}
			return resp.Result, nil
		},
		func(r client.IntegrationResourceRoleListItemResponseSchema) string { return r.Id.String() },
	)
	if err != nil {
		return nil, err
	}

	out := make(map[string]*bulksettings.Entity, len(found))
	for id, role := range found {
		e := &bulksettings.Entity{Requestable: utils.BoolValue(role.Requestable)}
		if role.Workflow != nil {
			e.WorkflowID = role.Workflow.Id.String()
		}
		out[id] = e
	}
	return out, nil
}
