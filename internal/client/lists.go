package client

// Hand-written list calls for resources, roles and integrations. They support
// the newer query parameters (ids, idsOperator, afterId, role filters) that are not yet in
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

// Defaults sent for unset filter operators (idsOperator defaults to
// IDsOperatorIs), so results do not depend on the API's own defaults.
const (
	DefaultOperator           = "is"
	DefaultMaintainerOperator = "contains"
	DefaultNameOperator       = "contains"
	DefaultTagsRelation       = "or"
	DefaultNameRelation       = "or"
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

	// Role, resource and integration filters.
	Requestable        *bool
	WorkflowIDs        []string
	WorkflowIDOperator string // is, isNot

	// Resource and integration filters.
	OwnerIDs             []string
	OwnerIDOperator      string // is, isNot
	MaintainerIDs        []string
	MaintainerIDOperator string // contains, notContains

	// Integration filters (GET /public/v1/integrations).
	Names                 []string // must not contain commas
	NameOperator          string   // contains, notContains
	NameRelation          string   // and, or
	ApplicationIDs        []string
	ApplicationIDOperator string // is, isNot

	// Resource filters (GET /public/v1/resources).
	Types        []string
	TypeOperator string // is, isNot, contains, notContains
	Tags         []string
	TagsOperator string // is, isNot
	TagsRelation string // and, or

	// Role filters (GET /public/v1/roles).
	ResourceTypes        []string
	ResourceTypeOperator string // is, isNot, contains, notContains
	ResourceTags         []string
	ResourceTagsOperator string // is, isNot
	ResourceTagsRelation string // and, or
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
	if q.Requestable != nil {
		v.Set("requestable", strconv.FormatBool(*q.Requestable))
	}
	setList(v, "workflowId", q.WorkflowIDs)
	setNonEmpty(v, "workflowIdOperator", q.WorkflowIDOperator)
	setList(v, "resourceType", q.ResourceTypes)
	setNonEmpty(v, "resourceTypeOperator", q.ResourceTypeOperator)
	setList(v, "resourceTags", q.ResourceTags)
	setNonEmpty(v, "resourceTagsOperator", q.ResourceTagsOperator)
	setNonEmpty(v, "resourceTagsRelation", q.ResourceTagsRelation)
	setList(v, "ownerId", q.OwnerIDs)
	setNonEmpty(v, "ownerIdOperator", q.OwnerIDOperator)
	setList(v, "maintainerId", q.MaintainerIDs)
	setNonEmpty(v, "maintainerIdOperator", q.MaintainerIDOperator)
	setList(v, "type", q.Types)
	setNonEmpty(v, "typeOperator", q.TypeOperator)
	setList(v, "tags", q.Tags)
	setNonEmpty(v, "tagsOperator", q.TagsOperator)
	setNonEmpty(v, "tagsRelation", q.TagsRelation)
	setList(v, "name", q.Names)
	setNonEmpty(v, "nameOperator", q.NameOperator)
	setNonEmpty(v, "nameRelation", q.NameRelation)
	setList(v, "applicationId", q.ApplicationIDs)
	setNonEmpty(v, "applicationIdOperator", q.ApplicationIDOperator)
	return v
}

// setList sends values as a repeated parameter, so values containing commas
// (resource types and tags are free text) are not split by the API.
func setList(v url.Values, key string, values []string) {
	if len(values) > 0 {
		v[key] = append([]string(nil), values...)
	}
}

func setNonEmpty(v url.Values, key, value string) {
	if value != "" {
		v.Set(key, value)
	}
}

// ListFunc performs one list call.
type ListFunc[T any] func(ctx context.Context, q ListQuery) (*ListResponse[T], error)

// maxListPages bounds ListAll so a misbehaving API cannot loop forever.
const maxListPages = 100_000

// ListAll reads every page of a list endpoint with the afterId cursor, which
// does not skip or repeat items while the list changes, and returns all items.
// Page, PerPage and AfterID of q are ignored. The API does not accept afterId
// together with search, so a query with Search is rejected; filter by name locally.
//
// When a call returns a non-200 status, ListAll stops and returns that
// response with a nil error; the caller reports it (Body holds the API error).
// On success the returned response is the last page's.
func ListAll[T any](ctx context.Context, q ListQuery, call ListFunc[T], idOf func(T) string) ([]T, *ListResponse[T], error) {
	if q.Search != "" {
		return nil, nil, fmt.Errorf("ListAll does not support search: the API does not accept it with the afterId cursor")
	}
	q.PerPage = MaxListPerPage
	q.Page = 0
	q.AfterID = ""

	var all []T
	for range maxListPages {
		resp, err := call(ctx, q)
		if err != nil {
			return nil, nil, err
		}
		if resp.StatusCode() != http.StatusOK {
			return nil, resp, nil
		}

		all = append(all, resp.Result...)
		if len(resp.Result) < q.PerPage {
			return all, resp, nil
		}

		next := idOf(resp.Result[len(resp.Result)-1])
		if next == "" || strings.EqualFold(next, q.AfterID) {
			return nil, nil, fmt.Errorf("list cursor did not advance past %q", q.AfterID)
		}
		q.AfterID = next
	}
	return nil, nil, fmt.Errorf("list did not finish after %d pages", maxListPages)
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
	if q.IDsOperator == "" {
		q.IDsOperator = IDsOperatorIs
	}
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
// field of GET /public/v1/resources/{id}. Without integrationId it lists the
// resources of every integration.
func (c *ClientWithResponses) ResourcesListWithResponse(ctx context.Context, q ListQuery) (*ListResponse[IntegrationResourceResultSchema], error) {
	return list[IntegrationResourceResultSchema](ctx, c, "/public/v1/resources", q.withResourceDefaults())
}

// withResourceDefaults fills the unset resource filter operators with
// DefaultOperator, DefaultMaintainerOperator and DefaultTagsRelation.
func (q ListQuery) withResourceDefaults() ListQuery {
	for _, op := range []*string{&q.WorkflowIDOperator, &q.OwnerIDOperator, &q.TypeOperator, &q.TagsOperator} {
		if *op == "" {
			*op = DefaultOperator
		}
	}
	if q.MaintainerIDOperator == "" {
		q.MaintainerIDOperator = DefaultMaintainerOperator
	}
	if q.TagsRelation == "" {
		q.TagsRelation = DefaultTagsRelation
	}
	return q
}

// RolesListWithResponse calls GET /public/v1/roles.
func (c *ClientWithResponses) RolesListWithResponse(ctx context.Context, q ListQuery) (*ListResponse[IntegrationResourceRoleListItemResponseSchema], error) {
	return list[IntegrationResourceRoleListItemResponseSchema](ctx, c, "/public/v1/roles", q.withRoleDefaults())
}

// withRoleDefaults fills the unset role filter operators with DefaultOperator
// and DefaultTagsRelation.
func (q ListQuery) withRoleDefaults() ListQuery {
	for _, op := range []*string{&q.WorkflowIDOperator, &q.ResourceTypeOperator, &q.ResourceTagsOperator} {
		if *op == "" {
			*op = DefaultOperator
		}
	}
	if q.ResourceTagsRelation == "" {
		q.ResourceTagsRelation = DefaultTagsRelation
	}
	return q
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
	return list[BulkIntegrationItem](ctx, c, "/public/v1/integrations", q.withIntegrationDefaults())
}

// IntegrationsListWithResponse calls GET /public/v1/integrations.
func (c *ClientWithResponses) IntegrationsListWithResponse(ctx context.Context, q ListQuery) (*ListResponse[IntegrationBaseResponseSchema], error) {
	return list[IntegrationBaseResponseSchema](ctx, c, "/public/v1/integrations", q.withIntegrationDefaults())
}

// withIntegrationDefaults fills the unset integration filter operators with
// DefaultOperator, DefaultMaintainerOperator, DefaultNameOperator and DefaultNameRelation.
func (q ListQuery) withIntegrationDefaults() ListQuery {
	for _, op := range []*string{&q.WorkflowIDOperator, &q.OwnerIDOperator, &q.ApplicationIDOperator} {
		if *op == "" {
			*op = DefaultOperator
		}
	}
	if q.MaintainerIDOperator == "" {
		q.MaintainerIDOperator = DefaultMaintainerOperator
	}
	if q.NameOperator == "" {
		q.NameOperator = DefaultNameOperator
	}
	if q.NameRelation == "" {
		q.NameRelation = DefaultNameRelation
	}
	return q
}
