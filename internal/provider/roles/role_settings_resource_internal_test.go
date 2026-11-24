package roles

// Direct tests of the role-specific parts of entitle_role_settings: reading
// roles into the common settings view, and submitting updates.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/entitleio/terraform-provider-entitle/internal/client"
)

func newRolesTestClient(t *testing.T, h http.HandlerFunc) *client.ClientWithResponses {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	c, err := client.NewClientWithResponses(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestFetchRoleSettings(t *testing.T) {
	c := newRolesTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/public/v1/roles" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
		  "pagination": {"page": 1, "perPage": 100, "totalResults": 2, "totalPages": 1},
		  "result": [
		    {
		      "id": "11111111-aaaa-4aaa-8aaa-111111111111", "name": "admin", "externalId": null,
		      "requestable": true,
		      "workflow": {"id": "3fa85f64-5717-4562-b3fc-2c963f66afa6", "name": "wf"},
		      "resource": {"id": "77777777-7777-7777-7777-777777777777", "name": "r", "type": "t", "externalId": "x",
		                   "integration": {"id": "88888888-8888-8888-8888-888888888888", "name": "i", "application": {"name": "a"}}}
		    },
		    {
		      "id": "22222222-bbbb-4bbb-8bbb-222222222222", "name": "read", "externalId": "ext",
		      "requestable": false, "workflow": null,
		      "resource": {"id": "77777777-7777-7777-7777-777777777777", "name": "r", "type": "t", "externalId": "x",
		                   "integration": {"id": "88888888-8888-8888-8888-888888888888", "name": "i", "application": {"name": "a"}}}
		    }
		  ]
		}`))
	})

	got, err := fetchRoleSettings(context.Background(), c, 2, []string{
		"11111111-aaaa-4aaa-8aaa-111111111111",
		"22222222-bbbb-4bbb-8bbb-222222222222",
		"33333333-cccc-4ccc-8ccc-333333333333", // does not exist
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d roles, want 2 (missing ones are absent)", len(got))
	}

	a := got["11111111-aaaa-4aaa-8aaa-111111111111"]
	if a == nil || !a.Requestable || a.WorkflowID != "3fa85f64-5717-4562-b3fc-2c963f66afa6" {
		t.Errorf("role a = %+v", a)
	}
	b := got["22222222-bbbb-4bbb-8bbb-222222222222"]
	if b == nil || b.Requestable || b.WorkflowID != "" || b.Owner != nil || len(b.Maintainers) != 0 {
		t.Errorf("role b = %+v", b)
	}
}

// IDs are sent at most 100 per request, without resourceId/integrationId, and looked up case-insensitively.
func TestFetchRoleSettingsBatches(t *testing.T) {
	var (
		mu      sync.Mutex
		batches []int
	)
	c := newRolesTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Has("integrationId") || q.Has("resourceId") {
			t.Errorf("scope parameters must not be sent with ids: %v", q)
		}
		ids := strings.Split(q.Get("ids"), ",")
		mu.Lock()
		batches = append(batches, len(ids))
		mu.Unlock()

		items := make([]map[string]any, len(ids))
		for i, id := range ids {
			items[i] = map[string]any{"id": strings.ToLower(id), "name": "n", "requestable": true}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"result": items})
	})

	ids := make([]string, 201)
	for i := range ids {
		ids[i] = fmt.Sprintf("%08X-0000-4000-8000-000000000000", i+1)
	}

	got, err := fetchRoleSettings(context.Background(), c, 3, ids)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 201 {
		t.Fatalf("got %d roles, want 201", len(got))
	}
	if _, ok := got[strings.ToLower(ids[200])]; !ok {
		t.Error("result must be keyed by lower-cased ID")
	}
	if len(batches) != 3 {
		t.Fatalf("got %d list requests, want 3", len(batches))
	}
	for _, n := range batches {
		if n > 100 {
			t.Errorf("list request with %d ids (max 100)", n)
		}
	}
}

func TestFetchRoleSettingsError(t *testing.T) {
	c := newRolesTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"errorId": "forbidden", "message": "not allowed"}`))
	})

	_, err := fetchRoleSettings(context.Background(), c, 1, []string{"11111111-aaaa-4aaa-8aaa-111111111111"})
	if err == nil || !strings.Contains(err.Error(), "listing roles") {
		t.Fatalf("expected a listing error, got %v", err)
	}
}

// The resource submits to PATCH /public/v1/roles/bulk with only requestable and
// workflow, even if the shared update carried owner or maintainers.
func TestRoleSettingsSubmit(t *testing.T) {
	var gotPath, gotMethod string
	var gotBody map[string]any
	c := newRolesTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.Path, r.Method
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"result": {"jobId": "00000000-0000-0000-0000-000000000009", "count": 1}}`))
	})

	var meta resource.MetadataResponse
	NewRoleSettingsResource().Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "entitle"}, &meta)
	if meta.TypeName != "entitle_role_settings" {
		t.Errorf("type name = %q", meta.TypeName)
	}

	kind := roleSettingsKind()
	if kind.WithOwner || kind.WithMaintainers {
		t.Error("roles have no owner or maintainers")
	}

	requestable := true
	submit := kind.Submit(c, client.BulkSettingsUpdate{
		Requestable: &requestable,
		Workflow:    &client.BulkEntityRef{Id: "3fa85f64-5717-4562-b3fc-2c963f66afa6"},
		Owner:       &client.BulkEntityRef{Id: "must-not-be-sent@example.com"},
	})
	resp, err := submit(context.Background(), []string{"11111111-aaaa-4aaa-8aaa-111111111111"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode() != http.StatusAccepted || resp.JSON202 == nil {
		t.Fatalf("status %d, job %+v", resp.StatusCode(), resp.JSON202)
	}
	if gotMethod != http.MethodPatch || gotPath != "/public/v1/roles/bulk" {
		t.Errorf("request = %s %s", gotMethod, gotPath)
	}

	update, _ := gotBody["update"].(map[string]any)
	if len(update) != 2 || update["requestable"] != true {
		t.Errorf("update body = %v, want only requestable and workflow", update)
	}
	if wf, _ := update["workflow"].(map[string]any); wf["id"] != "3fa85f64-5717-4562-b3fc-2c963f66afa6" {
		t.Errorf("workflow = %v", update["workflow"])
	}
}
