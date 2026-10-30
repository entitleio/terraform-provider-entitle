package resources

import (
	"context"

	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/entitleio/terraform-provider-entitle/docs"
	"github.com/entitleio/terraform-provider-entitle/internal/client"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/bulk"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/bulksettings"
)

// NewResourceSettingsResource creates the entitle_resource_settings resource:
// the same settings applied to many resources with PATCH /public/v1/resources/bulk.
// The behaviour is shared with entitle_integration_settings (see package bulksettings).
func NewResourceSettingsResource() resource.Resource {
	return bulksettings.New(bulksettings.Kind{
		TypeSuffix:          "_resource_settings",
		IDsAttribute:        "resource_ids",
		Entity:              "resource",
		Entities:            "resources",
		MarkdownDescription: docs.ResourceSettingsResourceMarkdownDescription,
		Fetch:               fetchResourceSettings,
		Submit: func(c *client.ClientWithResponses, u client.BulkSettingsUpdate) bulk.SubmitFunc {
			return bulk.ResourcesUpdate(c, u.AsResourceUpdate())
		},
	})
}

// fetchResourceSettings reads resources (100 per list request) into the common settings view.
func fetchResourceSettings(ctx context.Context, c *client.ClientWithResponses, parallelism int, ids []string) (map[string]*bulksettings.Entity, error) {
	found, err := fetchResourcesForBulk(ctx, c, parallelism, ids)
	if err != nil {
		return nil, err
	}

	out := make(map[string]*bulksettings.Entity, len(found))
	for id, res := range found {
		maintainers, err := bulksettings.ParseMaintainers(res.Maintainers)
		if err != nil {
			return nil, err
		}
		e := &bulksettings.Entity{
			Requestable: res.Requestable,
			Owner:       bulksettings.IdentityOf(res.Owner),
			Maintainers: maintainers,
		}
		if res.Workflow != nil {
			e.WorkflowID = res.Workflow.Id.String()
		}
		out[id] = e
	}
	return out, nil
}
