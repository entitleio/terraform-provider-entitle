package resources

import (
	"context"
	"fmt"
	"strings"

	"github.com/entitleio/terraform-provider-entitle/internal/client"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/bulk"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/utils"
)

// fetchResourcesForBulk reads the given resources with the list endpoint,
// up to client.MaxListIDs per request and parallelism requests at a time.
// The result is keyed by lower-cased ID; resources that do not exist are absent.
func fetchResourcesForBulk(ctx context.Context, c *client.ClientWithResponses, parallelism int, ids []string) (map[string]*client.IntegrationResourceResultSchema, error) {
	found, err := bulk.FetchByIDs(ctx, parallelism, ids,
		func(ctx context.Context, chunk []string) ([]client.IntegrationResourceResultSchema, error) {
			resp, err := c.ResourcesListWithResponse(ctx, client.ListQuery{IDs: chunk, PerPage: client.MaxListPerPage})
			if err != nil {
				return nil, fmt.Errorf("listing resources: %w", err)
			}
			if err := utils.HTTPResponseToError(resp.StatusCode(), resp.Body); err != nil {
				return nil, fmt.Errorf("listing resources: %w", err)
			}
			return resp.Result, nil
		},
		func(r client.IntegrationResourceResultSchema) string { return r.Id.String() },
	)
	if err != nil {
		return nil, err
	}

	out := make(map[string]*client.IntegrationResourceResultSchema, len(found))
	for id, res := range found {
		out[id] = &res
	}
	return out, nil
}

// inspectResources is the shared bulk.InspectFunc body for resource bulk
// resources: it fetches every wanted resource in batches and lets check decide
// each one's presence. Missing resources are left out (they do not exist).
func inspectResources(
	ctx context.Context,
	c *client.ClientWithResponses,
	parallelism int,
	want map[string][]string,
	check func(res *client.IntegrationResourceResultSchema, values []string) (bulk.Presence, error),
) (map[string]bulk.Presence, error) {
	found, err := fetchResourcesForBulk(ctx, c, parallelism, bulk.SortedKeys(want))
	if err != nil {
		return nil, err
	}

	out := make(map[string]bulk.Presence, len(want))
	for id, values := range want {
		res, ok := found[strings.ToLower(id)]
		if !ok {
			continue
		}
		p, err := check(res, values)
		if err != nil {
			return nil, fmt.Errorf("resource %s: %w", id, err)
		}
		p.Exists = true
		out[id] = p
	}
	return out, nil
}
