package integrations

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

// NewIntegrationSettingsResource creates the entitle_integration_settings resource:
// the same settings applied to many integrations with PATCH /public/v1/integrations/bulk.
// The behaviour is shared with entitle_resource_settings (see package bulksettings).
func NewIntegrationSettingsResource() resource.Resource {
	return bulksettings.New(integrationSettingsKind())
}

// integrationSettingsKind describes entitle_integration_settings to the shared settings implementation.
func integrationSettingsKind() bulksettings.Kind {
	return bulksettings.Kind{
		TypeSuffix:          "_integration_settings",
		IDsAttribute:        "integration_ids",
		Entity:              "integration",
		Entities:            "integrations",
		MarkdownDescription: docs.IntegrationSettingsResourceMarkdownDescription,
		Fetch:               fetchIntegrationSettings,
		Submit: func(c *client.ClientWithResponses, u client.BulkSettingsUpdate) bulk.SubmitFunc {
			return func(ctx context.Context, ids []string) (*client.BulkSubmitResponse, error) {
				return c.IntegrationsBulkUpdateWithResponse(ctx, client.BulkUpdateIntegrationsRequest{Ids: ids, Update: u})
			}
		},
	}
}

// fetchIntegrationSettings reads integrations (100 per list request) into the common settings view.
func fetchIntegrationSettings(ctx context.Context, c *client.ClientWithResponses, parallelism int, ids []string) (map[string]*bulksettings.Entity, error) {
	found, err := bulk.FetchByIDs(ctx, parallelism, ids,
		func(ctx context.Context, chunk []string) ([]client.BulkIntegrationItem, error) {
			resp, err := c.IntegrationsBulkListWithResponse(ctx, client.ListQuery{IDs: chunk, PerPage: client.MaxListPerPage})
			if err != nil {
				return nil, fmt.Errorf("listing integrations: %w", err)
			}
			if err := utils.HTTPResponseToError(resp.StatusCode(), resp.Body); err != nil {
				return nil, fmt.Errorf("listing integrations: %w", err)
			}
			return resp.Result, nil
		},
		func(it client.BulkIntegrationItem) string { return it.Id.String() },
	)
	if err != nil {
		return nil, err
	}

	out := make(map[string]*bulksettings.Entity, len(found))
	for id, it := range found {
		maintainers, err := bulksettings.ParseMaintainers(it.Maintainers)
		if err != nil {
			return nil, fmt.Errorf("integration %s: reading maintainers: %w", id, err)
		}
		e := &bulksettings.Entity{
			Requestable: it.Requestable,
			Owner:       bulksettings.IdentityOf(it.Owner),
			Maintainers: maintainers,
		}
		if it.Workflow != nil {
			e.WorkflowID = it.Workflow.Id.String()
		}
		out[id] = e
	}
	return out, nil
}
