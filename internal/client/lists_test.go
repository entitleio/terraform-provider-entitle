package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestListQueryValues(t *testing.T) {
	q := ListQuery{
		IDs:         []string{"a", "b"},
		IDsOperator: IDsOperatorIsNot,
		AfterID:     "c",
		PerPage:     50,
	}
	got := q.Values()
	want := url.Values{"ids": {"a,b"}, "idsOperator": {"isNot"}, "afterId": {"c"}, "perPage": {"50"}}
	if got.Encode() != want.Encode() {
		t.Fatalf("got %s, want %s", got.Encode(), want.Encode())
	}

	if enc := (ListQuery{}).Values().Encode(); enc != "" {
		t.Fatalf("empty query should encode to nothing, got %q", enc)
	}
}

func TestResourcesListWithResponse(t *testing.T) {
	var gotQuery url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/public/v1/resources" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		gotQuery = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
		  "pagination": {"perPage": 100, "page": 1, "totalResults": 1, "totalPages": 1},
		  "result": [{
		    "requestable": true,
		    "id": "0006d765-fcff-4e7c-b2df-19a805e0f0b5",
		    "integration": {"application": {"name": "AWS"}, "id": "ccea6b23-b73b-47f9-a061-7a1ccedf75be", "adapterName": "aws", "name": "aws"},
		    "maintainers": null, "name": "Policy", "owner": null,
		    "tags": ["account: Lab"], "type": "Policy", "userDefinedTags": null, "workflow": null
		  }]
		}`))
	}))
	defer srv.Close()

	c, err := NewClientWithResponses(srv.URL)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := c.ResourcesListWithResponse(context.Background(), ListQuery{IDs: []string{"0006d765-fcff-4e7c-b2df-19a805e0f0b5"}})
	if err != nil {
		t.Fatal(err)
	}
	if gotQuery.Get("ids") != "0006d765-fcff-4e7c-b2df-19a805e0f0b5" || gotQuery.Has("integrationId") {
		t.Fatalf("unexpected query %v", gotQuery)
	}
	if resp.StatusCode() != http.StatusOK || len(resp.Result) != 1 {
		t.Fatalf("status %d, result %+v", resp.StatusCode(), resp.Result)
	}
	res := resp.Result[0]
	if !res.Requestable || res.Owner != nil || res.Workflow != nil || res.Maintainers != nil {
		t.Fatalf("unexpected item %+v", res)
	}
	if res.Tags == nil || (*res.Tags)[0] != "account: Lab" {
		t.Fatalf("tags = %v", res.Tags)
	}
	if resp.Pagination == nil || resp.Pagination.TotalResults != 1 {
		t.Fatalf("pagination = %+v", resp.Pagination)
	}
}

// The integrations list response (as returned by the API) decodes into the
// fields entitle_integration_settings compares for drift.
func TestIntegrationsBulkListWithResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/public/v1/integrations" || r.URL.Query().Get("ids") == "" {
			t.Errorf("unexpected request %s?%s", r.URL.Path, r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
		  "pagination": {"page": 0, "perPage": 0, "totalResults": 0, "totalPages": 0},
		  "result": [{
		    "id": "00000000-0000-0000-0000-000000000001",
		    "name": "string",
		    "application": {"name": "string"},
		    "requestable": true,
		    "workflow": {"id": "3fa85f64-5717-4562-b3fc-2c963f66afa6", "name": "string"},
		    "owner": {"id": "00000000-0000-0000-0000-000000000002", "email": "user@org.domain"},
		    "maintainers": [
		      {"type": "group", "group": {"id": "00000000-0000-0000-0000-000000000003", "email": "user@org.domain"}},
		      {"type": "user", "user": {"id": "00000000-0000-0000-0000-000000000004", "email": "user@org.domain"}}
		    ]
		  }]
		}`))
	}))
	defer srv.Close()

	c, err := NewClientWithResponses(srv.URL)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := c.IntegrationsBulkListWithResponse(context.Background(), ListQuery{IDs: []string{"00000000-0000-0000-0000-000000000001"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(resp.Result) != 1 {
		t.Fatalf("result = %+v", resp.Result)
	}

	it := resp.Result[0]
	if !it.Requestable {
		t.Error("requestable not decoded")
	}
	if it.Workflow == nil || it.Workflow.Id.String() != "3fa85f64-5717-4562-b3fc-2c963f66afa6" {
		t.Errorf("workflow = %+v", it.Workflow)
	}
	if it.Owner == nil || it.Owner.Id.String() != "00000000-0000-0000-0000-000000000002" ||
		it.Owner.Email == nil || string(*it.Owner.Email) != "user@org.domain" {
		t.Errorf("owner = %+v", it.Owner)
	}
	if len(it.Maintainers) != 2 {
		t.Fatalf("maintainers = %d, want 2", len(it.Maintainers))
	}
	group, err := it.Maintainers[0].AsMaintainerGroupResponseSchema()
	if err != nil || group.Group.Id.String() != "00000000-0000-0000-0000-000000000003" {
		t.Errorf("group maintainer = %+v, %v", group, err)
	}
	user, err := it.Maintainers[1].AsMaintainerUserResponseSchema()
	if err != nil || user.User.Id.String() != "00000000-0000-0000-0000-000000000004" {
		t.Errorf("user maintainer = %+v, %v", user, err)
	}
}
