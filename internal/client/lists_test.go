package client

import (
	"context"
	"fmt"
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

func TestListQueryValuesResourceFilters(t *testing.T) {
	q := ListQuery{
		OwnerIDs:             []string{"o1", "o2"},
		OwnerIDOperator:      "isNot",
		MaintainerIDs:        []string{"m"},
		MaintainerIDOperator: "notContains",
		Types:                []string{"a,b"},
		TypeOperator:         "contains",
		Tags:                 []string{"env: prod"},
		TagsOperator:         "isNot",
		TagsRelation:         "and",
	}
	want := url.Values{
		"ownerId":              {"o1", "o2"},
		"ownerIdOperator":      {"isNot"},
		"maintainerId":         {"m"},
		"maintainerIdOperator": {"notContains"},
		"type":                 {"a,b"},
		"typeOperator":         {"contains"},
		"tags":                 {"env: prod"},
		"tagsOperator":         {"isNot"},
		"tagsRelation":         {"and"},
	}
	if got := q.Values(); got.Encode() != want.Encode() {
		t.Fatalf("got %s, want %s", got.Encode(), want.Encode())
	}
}

func TestListQueryValuesIntegrationFilters(t *testing.T) {
	q := ListQuery{
		Names:                 []string{"prod", "aws"},
		NameOperator:          "notContains",
		NameRelation:          "and",
		ApplicationIDs:        []string{"a"},
		ApplicationIDOperator: "isNot",
	}
	want := url.Values{
		"name":                  {"prod", "aws"},
		"nameOperator":          {"notContains"},
		"nameRelation":          {"and"},
		"applicationId":         {"a"},
		"applicationIdOperator": {"isNot"},
	}
	if got := q.Values(); got.Encode() != want.Encode() {
		t.Fatalf("got %s, want %s", got.Encode(), want.Encode())
	}
}

func TestListQueryValuesRoleFilters(t *testing.T) {
	requestable := false
	q := ListQuery{
		IntegrationID:        "i",
		Requestable:          &requestable,
		WorkflowIDs:          []string{"w1", "w2"},
		WorkflowIDOperator:   "isNot",
		ResourceTypes:        []string{"a,b", "c"},
		ResourceTypeOperator: "contains",
		ResourceTags:         []string{"env: prod"},
		ResourceTagsOperator: "is",
		ResourceTagsRelation: "and",
	}
	want := url.Values{
		"integrationId":        {"i"},
		"requestable":          {"false"},
		"workflowId":           {"w1", "w2"},
		"workflowIdOperator":   {"isNot"},
		"resourceType":         {"a,b", "c"},
		"resourceTypeOperator": {"contains"},
		"resourceTags":         {"env: prod"},
		"resourceTagsOperator": {"is"},
		"resourceTagsRelation": {"and"},
	}
	if got := q.Values(); got.Encode() != want.Encode() {
		t.Fatalf("got %s, want %s", got.Encode(), want.Encode())
	}
}

// Operators are sent even without their values.
func TestListQueryValuesOperatorsWithoutValues(t *testing.T) {
	q := ListQuery{
		IntegrationID:        "i",
		IDsOperator:          IDsOperatorIsNot,
		WorkflowIDOperator:   "isNot",
		ResourceTypeOperator: "contains",
		ResourceTagsOperator: "isNot",
		ResourceTagsRelation: "and",
	}
	want := url.Values{
		"integrationId":        {"i"},
		"idsOperator":          {"isNot"},
		"workflowIdOperator":   {"isNot"},
		"resourceTypeOperator": {"contains"},
		"resourceTagsOperator": {"isNot"},
		"resourceTagsRelation": {"and"},
	}.Encode()
	if got := q.Values().Encode(); got != want {
		t.Fatalf("got %s, want %s", got, want)
	}
}

type listItem struct{ ID string }

// fakePages serves items (sorted by ID) like the API with the afterId cursor.
func fakePages(items []string, calls *[]ListQuery) ListFunc[listItem] {
	return func(_ context.Context, q ListQuery) (*ListResponse[listItem], error) {
		*calls = append(*calls, q)

		start := 0
		for q.AfterID != "" && start < len(items) && items[start] <= q.AfterID {
			start++
		}
		end := min(start+q.PerPage, len(items))

		res := make([]listItem, 0, end-start)
		for _, id := range items[start:end] {
			res = append(res, listItem{ID: id})
		}
		return &ListResponse[listItem]{
			HTTPResponse: &http.Response{StatusCode: http.StatusOK},
			Result:       res,
		}, nil
	}
}

func ids(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = fmt.Sprintf("%08d", i)
	}
	return out
}

func TestListAll(t *testing.T) {
	idOf := func(it listItem) string { return it.ID }

	for _, tc := range []struct {
		name      string
		total     int
		wantCalls int
	}{
		{"empty", 0, 1},
		{"one partial page", 42, 1},
		{"exact page", 100, 2},
		{"several pages", 250, 3},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls []ListQuery
			all, resp, err := ListAll(context.Background(),
				ListQuery{Page: 7, PerPage: 3, AfterID: "ignored"},
				fakePages(ids(tc.total), &calls), idOf)
			if err != nil || resp == nil || resp.StatusCode() != http.StatusOK {
				t.Fatalf("err %v, resp %+v", err, resp)
			}
			if len(all) != tc.total {
				t.Fatalf("got %d items, want %d", len(all), tc.total)
			}
			if len(calls) != tc.wantCalls {
				t.Fatalf("got %d calls, want %d", len(calls), tc.wantCalls)
			}
			for i, c := range calls {
				if c.PerPage != MaxListPerPage || c.Page != 0 {
					t.Fatalf("call %d: %+v", i, c)
				}
			}
			if calls[0].AfterID != "" {
				t.Fatalf("first call sent afterId %q", calls[0].AfterID)
			}
		})
	}
}

func TestListAllRejectsSearch(t *testing.T) {
	var calls []ListQuery
	_, _, err := ListAll(context.Background(), ListQuery{Search: "x"}, fakePages(nil, &calls), func(it listItem) string { return it.ID })
	if err == nil || len(calls) != 0 {
		t.Fatalf("err %v, calls %d", err, len(calls))
	}
}

func TestListAllErrorStatus(t *testing.T) {
	n := 0
	call := func(_ context.Context, q ListQuery) (*ListResponse[listItem], error) {
		n++
		if n == 1 {
			res := make([]listItem, MaxListPerPage)
			for i := range res {
				res[i] = listItem{ID: fmt.Sprintf("%08d", i)}
			}
			return &ListResponse[listItem]{HTTPResponse: &http.Response{StatusCode: http.StatusOK}, Result: res}, nil
		}
		return &ListResponse[listItem]{HTTPResponse: &http.Response{StatusCode: http.StatusNotFound}, Body: []byte(`{}`)}, nil
	}

	all, resp, err := ListAll(context.Background(), ListQuery{}, call, func(it listItem) string { return it.ID })
	if err != nil || all != nil || resp.StatusCode() != http.StatusNotFound {
		t.Fatalf("all %d, resp %+v, err %v", len(all), resp, err)
	}
}

func TestListAllStuckCursor(t *testing.T) {
	call := func(_ context.Context, q ListQuery) (*ListResponse[listItem], error) {
		res := make([]listItem, MaxListPerPage)
		for i := range res {
			res[i] = listItem{ID: "same"}
		}
		return &ListResponse[listItem]{HTTPResponse: &http.Response{StatusCode: http.StatusOK}, Result: res}, nil
	}
	if _, _, err := ListAll(context.Background(), ListQuery{}, call, func(it listItem) string { return it.ID }); err == nil {
		t.Fatal("expected an error when the cursor does not advance")
	}
}

// queryOf serves an empty list and records the query of the last request.
func queryOf(t *testing.T, got *url.Values) *ClientWithResponses {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		*got = r.URL.Query()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"pagination": {}, "result": []}`))
	}))
	t.Cleanup(srv.Close)

	c, err := NewClientWithResponses(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// Unset operators are sent with the defaults, each endpoint only its own.
func TestListDefaultOperators(t *testing.T) {
	var got url.Values
	c := queryOf(t, &got)

	if _, err := c.RolesListWithResponse(context.Background(), ListQuery{IntegrationID: "i"}); err != nil {
		t.Fatal(err)
	}
	want := url.Values{
		"integrationId":        {"i"},
		"idsOperator":          {"is"},
		"workflowIdOperator":   {"is"},
		"resourceTypeOperator": {"is"},
		"resourceTagsOperator": {"is"},
		"resourceTagsRelation": {"or"},
	}
	if got.Encode() != want.Encode() {
		t.Fatalf("roles: got %s, want %s", got.Encode(), want.Encode())
	}

	if _, err := c.RolesListWithResponse(context.Background(), ListQuery{
		IDsOperator: IDsOperatorIsNot, WorkflowIDOperator: "isNot", ResourceTypeOperator: "contains",
		ResourceTagsOperator: "isNot", ResourceTagsRelation: "and",
	}); err != nil {
		t.Fatal(err)
	}
	if got.Get("idsOperator") != "isNot" || got.Get("workflowIdOperator") != "isNot" || got.Get("resourceTypeOperator") != "contains" ||
		got.Get("resourceTagsOperator") != "isNot" || got.Get("resourceTagsRelation") != "and" {
		t.Fatalf("roles: configured operators not kept: %s", got.Encode())
	}

	if _, err := c.ResourcesListWithResponse(context.Background(), ListQuery{IntegrationID: "i"}); err != nil {
		t.Fatal(err)
	}
	want = url.Values{
		"integrationId":        {"i"},
		"idsOperator":          {"is"},
		"workflowIdOperator":   {"is"},
		"ownerIdOperator":      {"is"},
		"maintainerIdOperator": {"contains"},
		"typeOperator":         {"is"},
		"tagsOperator":         {"is"},
		"tagsRelation":         {"or"},
	}
	if got.Encode() != want.Encode() {
		t.Fatalf("resources: got %s, want %s", got.Encode(), want.Encode())
	}

	if _, err := c.IntegrationsListWithResponse(context.Background(), ListQuery{}); err != nil {
		t.Fatal(err)
	}
	want = url.Values{
		"idsOperator":           {"is"},
		"workflowIdOperator":    {"is"},
		"ownerIdOperator":       {"is"},
		"maintainerIdOperator":  {"contains"},
		"nameOperator":          {"contains"},
		"nameRelation":          {"or"},
		"applicationIdOperator": {"is"},
	}
	if got.Encode() != want.Encode() {
		t.Fatalf("integrations: got %s, want %s", got.Encode(), want.Encode())
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
