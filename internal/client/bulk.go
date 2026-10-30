package client

// Hand-written client for the bulk endpoints, which are not part of the
// OpenAPI specification used to generate entitle.gen.go. Once the spec
// contains them, these types and methods can be replaced by generated ones.

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	openapi_types "github.com/oapi-codegen/runtime/types"
)

// BulkOperationStatus is the lifecycle state of an asynchronous bulk operation.
type BulkOperationStatus string

const (
	BulkOperationStatusQueued    BulkOperationStatus = "queued"
	BulkOperationStatusRunning   BulkOperationStatus = "running"
	BulkOperationStatusCompleted BulkOperationStatus = "completed"
	BulkOperationStatusFailed    BulkOperationStatus = "failed"
)

// IsTerminal reports whether the operation has finished (successfully or not).
func (s BulkOperationStatus) IsTerminal() bool {
	return s == BulkOperationStatusCompleted || s == BulkOperationStatusFailed
}

// BulkEntityRef references an entity by identifier (UUID, email or unique identifier).
type BulkEntityRef struct {
	Id string `json:"id"`
}

// BulkMaintainer is a single maintainer entry in a bulk update.
type BulkMaintainer struct {
	Type  string         `json:"type"` // "user" or "group"
	User  *BulkEntityRef `json:"user,omitempty"`
	Group *BulkEntityRef `json:"group,omitempty"`
}

// BulkMaintainersUpdate describes maintainer changes in a bulk update.
type BulkMaintainersUpdate struct {
	Replace *[]BulkMaintainer `json:"replace,omitempty"`
	Add     *[]BulkMaintainer `json:"add,omitempty"`
	Remove  *[]BulkMaintainer `json:"remove,omitempty"`
}

// BulkTagsUpdate describes tag changes in a bulk update.
type BulkTagsUpdate struct {
	Add    *[]string `json:"add,omitempty"`
	Remove *[]string `json:"remove,omitempty"`
}

// BulkResourceUpdate is the set of changes applied to every resource in a bulk request.
// Nil fields are left untouched.
type BulkResourceUpdate struct {
	Requestable *bool                  `json:"requestable,omitempty"`
	Workflow    *BulkEntityRef         `json:"workflow,omitempty"`
	Owner       *BulkEntityRef         `json:"owner,omitempty"`
	Maintainers *BulkMaintainersUpdate `json:"maintainers,omitempty"`
	Tags        *BulkTagsUpdate        `json:"tags,omitempty"`
}

// BulkSettingsUpdate is the update body shared by the settings bulk endpoints
// (resources, integrations). Nil fields are left untouched.
type BulkSettingsUpdate struct {
	Requestable *bool                  `json:"requestable,omitempty"`
	Workflow    *BulkEntityRef         `json:"workflow,omitempty"`
	Owner       *BulkEntityRef         `json:"owner,omitempty"`
	Maintainers *BulkMaintainersUpdate `json:"maintainers,omitempty"`
}

// IsEmpty reports whether the update changes nothing.
func (u BulkSettingsUpdate) IsEmpty() bool {
	return u.Requestable == nil && u.Workflow == nil && u.Owner == nil && u.Maintainers == nil
}

// AsResourceUpdate converts the settings update into a resources bulk update.
func (u BulkSettingsUpdate) AsResourceUpdate() BulkResourceUpdate {
	return BulkResourceUpdate{Requestable: u.Requestable, Workflow: u.Workflow, Owner: u.Owner, Maintainers: u.Maintainers}
}

// BulkUpdateIntegrationsRequest is the body of PATCH /public/v1/integrations/bulk.
type BulkUpdateIntegrationsRequest struct {
	Ids    []string           `json:"ids"`
	Update BulkSettingsUpdate `json:"update"`
}

// BulkUpdateResourcesRequest is the body of PATCH /public/v1/resources/bulk.
type BulkUpdateResourcesRequest struct {
	Ids    []string           `json:"ids"`
	Update BulkResourceUpdate `json:"update"`
}

// BulkJobResult is returned when a bulk operation is accepted.
type BulkJobResult struct {
	JobId openapi_types.UUID `json:"jobId"`
	Count int                `json:"count"`
}

// BulkJobResponseSchema wraps BulkJobResult.
type BulkJobResponseSchema struct {
	Result BulkJobResult `json:"result"`
}

// BulkOperationResult describes the progress of a bulk operation.
type BulkOperationResult struct {
	Status    BulkOperationStatus `json:"status"`
	Processed int                 `json:"processed"`
	Total     int                 `json:"total"`
	Error     *string             `json:"error,omitempty"`
}

// BulkOperationResponseSchema wraps BulkOperationResult.
type BulkOperationResponseSchema struct {
	Result BulkOperationResult `json:"result"`
}

// BulkSubmitResponse is the raw + parsed response of a bulk submit call.
type BulkSubmitResponse struct {
	Body         []byte
	HTTPResponse *http.Response
	JSON202      *BulkJobResponseSchema
}

// StatusCode returns HTTPResponse.StatusCode.
func (r BulkSubmitResponse) StatusCode() int {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.StatusCode
	}
	return 0
}

// BulkOperationShowResponse is the raw + parsed response of GET /public/v1/bulkOperations/{jobId}.
type BulkOperationShowResponse struct {
	Body         []byte
	HTTPResponse *http.Response
	JSON200      *BulkOperationResponseSchema
}

// StatusCode returns HTTPResponse.StatusCode.
func (r BulkOperationShowResponse) StatusCode() int {
	if r.HTTPResponse != nil {
		return r.HTTPResponse.StatusCode
	}
	return 0
}

// ResourcesBulkUpdateWithResponse calls PATCH /public/v1/resources/bulk.
func (c *ClientWithResponses) ResourcesBulkUpdateWithResponse(ctx context.Context, body BulkUpdateResourcesRequest) (*BulkSubmitResponse, error) {
	return c.SubmitBulkWithResponse(ctx, http.MethodPatch, "/public/v1/resources/bulk", body)
}

// IntegrationsBulkUpdateWithResponse calls PATCH /public/v1/integrations/bulk.
func (c *ClientWithResponses) IntegrationsBulkUpdateWithResponse(ctx context.Context, body BulkUpdateIntegrationsRequest) (*BulkSubmitResponse, error) {
	return c.SubmitBulkWithResponse(ctx, http.MethodPatch, "/public/v1/integrations/bulk", body)
}

// SubmitBulkWithResponse sends a bulk request to the given path and parses the
// 200/202 job response. It is shared by all bulk endpoints.
func (c *ClientWithResponses) SubmitBulkWithResponse(ctx context.Context, method, path string, body any) (*BulkSubmitResponse, error) {
	httpResp, respBody, err := c.doJSON(ctx, method, path, body)
	if err != nil {
		return nil, err
	}

	resp := &BulkSubmitResponse{Body: respBody, HTTPResponse: httpResp}
	if httpResp.StatusCode == http.StatusAccepted || httpResp.StatusCode == http.StatusOK {
		var dest BulkJobResponseSchema
		if err := json.Unmarshal(respBody, &dest); err != nil {
			return nil, fmt.Errorf("decoding bulk job response: %w", err)
		}
		resp.JSON202 = &dest
	}

	return resp, nil
}

// BulkOperationsShowWithResponse calls GET /public/v1/bulkOperations/{jobId}.
func (c *ClientWithResponses) BulkOperationsShowWithResponse(ctx context.Context, jobID openapi_types.UUID) (*BulkOperationShowResponse, error) {
	httpResp, respBody, err := c.doJSON(ctx, http.MethodGet, "/public/v1/bulkOperations/"+url.PathEscape(jobID.String()), nil)
	if err != nil {
		return nil, err
	}

	resp := &BulkOperationShowResponse{Body: respBody, HTTPResponse: httpResp}
	if httpResp.StatusCode == http.StatusOK {
		var dest BulkOperationResponseSchema
		if err := json.Unmarshal(respBody, &dest); err != nil {
			return nil, fmt.Errorf("decoding bulk operation response: %w", err)
		}
		resp.JSON200 = &dest
	}

	return resp, nil
}

// doJSON performs a JSON request using the underlying generated client's server,
// HTTP doer (including retry logic) and request editors (including auth).
func (c *ClientWithResponses) doJSON(ctx context.Context, method, path string, body any) (*http.Response, []byte, error) {
	base, ok := c.ClientInterface.(*Client)
	if !ok {
		return nil, nil, fmt.Errorf("unsupported client implementation %T", c.ClientInterface)
	}

	var reader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			return nil, nil, fmt.Errorf("encoding request body: %w", err)
		}
		// bytes.Reader lets http.NewRequest set GetBody, so RetryDoer can rewind.
		reader = bytes.NewReader(buf)
	}

	req, err := http.NewRequestWithContext(ctx, method, strings.TrimSuffix(base.Server, "/")+path, reader)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if err := base.applyEditors(ctx, req, nil); err != nil {
		return nil, nil, err
	}

	httpResp, err := base.Client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = httpResp.Body.Close() }()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, nil, err
	}

	return httpResp, respBody, nil
}
