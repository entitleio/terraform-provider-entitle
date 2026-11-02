package client

// Hand-written list calls for resources, roles and integrations. They support
// the newer query parameters (ids, idsOperator, afterId) that are not yet in
// the OpenAPI specification used to generate entitle.gen.go, and the resources
// list returns full items (same fields as GET /public/v1/resources/{id}).

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	openapi_types "github.com/oapi-codegen/runtime/types"
)

// MaxListIDs is the maximum number of IDs accepted by the ids list filter.
const MaxListIDs = 100

// MaxListPerPage is the maximum page size of list endpoints.
const MaxListPerPage = 100

// IDsOperator controls how the ids filter is matched.
type IDsOperator string

const (
	IDsOperatorIs    IDsOperator = "is"
	IDsOperatorIsNot IDsOperator = "isNot"
)

// ListQuery holds the query parameters shared by the list endpoints. Zero
// values are omitted. Not every endpoint supports every parameter.
type ListQuery struct {
	IDs         []string
	IDsOperator IDsOperator
	AfterID     string
	Page        int
	PerPage     int
	Search      string
	ExternalID  string

	IntegrationID string // resources, roles
	ResourceID    string // roles
}

// Values encodes the query. ids is sent as a comma-separated list.
func (q ListQuery) Values() url.Values {
	v := url.Values{}
	if len(q.IDs) > 0 {
		v.Set("ids", strings.Join(q.IDs, ","))
	}
	if q.IDsOperator != "" {
		v.Set("idsOperator", string(q.IDsOperator))
	}
	if q.AfterID != "" {
		v.Set("afterId", q.AfterID)
	}
	if q.Page > 0 {
		v.Set("page", strconv.Itoa(q.Page))
	}
	if q.PerPage > 0 {
		v.Set("perPage", strconv.Itoa(q.PerPage))
	}
	if q.Search != "" {
		v.Set("search", q.Search)
	}
	if q.ExternalID != "" {
		v.Set("externalId", q.ExternalID)
	}
	if q.IntegrationID != "" {
		v.Set("integrationId", q.IntegrationID)
	}
	if q.ResourceID != "" {
		v.Set("resourceId", q.ResourceID)
	}
	return v
}

// ListResponse is the raw + parsed response of a list call. Result is set only on 200.
type ListResponse[T any] struct {
	Body         []byte
	HTTPResponse *http.Response
	Pagination   *PaginationResponseSchema
	Result       []T
}

// StatusCode returns HTTPResponse.StatusCode.
func (r ListResponse[T]) StatusCode() int {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.StatusCode
	}
	return 0
}

func list[T any](ctx context.Context, c *ClientWithResponses, path string, q ListQuery) (*ListResponse[T], error) {
	if encoded := q.Values().Encode(); encoded != "" {
		path += "?" + encoded
	}

	httpResp, body, err := c.doJSON(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	resp := &ListResponse[T]{Body: body, HTTPResponse: httpResp}
	if httpResp.StatusCode == http.StatusOK {
		var dest struct {
			Pagination *PaginationResponseSchema `json:"pagination"`
			Result     []T                       `json:"result"`
		}
		if err := json.Unmarshal(body, &dest); err != nil {
			return nil, fmt.Errorf("decoding list response from %s: %w", path, err)
		}
		resp.Pagination, resp.Result = dest.Pagination, dest.Result
	}
	return resp, nil
}

// ResourcesListWithResponse calls GET /public/v1/resources. Items carry every
// field of GET /public/v1/resources/{id}. integrationId is not needed when ids is set.
func (c *ClientWithResponses) ResourcesListWithResponse(ctx context.Context, q ListQuery) (*ListResponse[IntegrationResourceResultSchema], error) {
	return list[IntegrationResourceResultSchema](ctx, c, "/public/v1/resources", q)
}

// RolesListWithResponse calls GET /public/v1/roles.
func (c *ClientWithResponses) RolesListWithResponse(ctx context.Context, q ListQuery) (*ListResponse[IntegrationResourceRoleListItemResponseSchema], error) {
	return list[IntegrationResourceRoleListItemResponseSchema](ctx, c, "/public/v1/roles", q)
}

// BulkIntegrationItem is an integration list item with the fields the bulk
// settings resource manages. Owner and workflow are pointers so that a null
// value is distinguishable from an unset one.
type BulkIntegrationItem struct {
	Id          openapi_types.UUID                         `json:"id"`
	Name        string                                     `json:"name"`
	Requestable bool                                       `json:"requestable"`
	Workflow    *WorkflowResponseSchema                    `json:"workflow"`
	Owner       *EntityResponseSchema                      `json:"owner"`
	Maintainers []IntegrationResultSchema_Maintainers_Item `json:"maintainers"`
}

// IntegrationsBulkListWithResponse calls GET /public/v1/integrations and decodes
// the full items the bulk settings resource needs.
func (c *ClientWithResponses) IntegrationsBulkListWithResponse(ctx context.Context, q ListQuery) (*ListResponse[BulkIntegrationItem], error) {
	return list[BulkIntegrationItem](ctx, c, "/public/v1/integrations", q)
}

// IntegrationsListWithResponse calls GET /public/v1/integrations.
func (c *ClientWithResponses) IntegrationsListWithResponse(ctx context.Context, q ListQuery) (*ListResponse[IntegrationBaseResponseSchema], error) {
	return list[IntegrationBaseResponseSchema](ctx, c, "/public/v1/integrations", q)
}
