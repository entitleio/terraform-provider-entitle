package integrations

// Direct tests of the integration-specific parts of entitle_integration_settings:
// reading integrations into the common settings view, and submitting updates.

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

func newTestClient(t *testing.T, h http.HandlerFunc) *client.ClientWithResponses {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	c, err := client.NewClientWithResponses(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestFetchIntegrationSettings(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/public/v1/integrations" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
		  "pagination": {"page": 1, "perPage": 100, "totalResults": 2, "totalPages": 1},
		  "result": [
		    {
		      "id": "44444444-4444-4444-4444-444444444444",
		      "name": "full", "application": {"name": "AWS"},
		      "requestable": true,
		      "workflow": {"id": "3fa85f64-5717-4562-b3fc-2c963f66afa6", "name": "wf"},
		      "owner": {"id": "00000000-0000-0000-0000-000000000002", "email": "owner@org.domain"},
		      "maintainers": [
		        {"type": "group", "group": {"id": "00000000-0000-0000-0000-000000000003", "email": "team@org.domain"}},
		        {"type": "user", "user": {"id": "00000000-0000-0000-0000-000000000004", "email": "user@org.domain"}}
		      ]
		    },
		    {
		      "id": "55555555-5555-5555-5555-555555555555",
		      "name": "empty", "application": {"name": "Manual"},
		      "requestable": false, "workflow": null, "owner": null, "maintainers": null
		    }
		  ]
		}`))
	})

	got, err := fetchIntegrationSettings(context.Background(), c, 2, []string{
		"44444444-4444-4444-4444-444444444444",
		"55555555-5555-5555-5555-555555555555",
		"66666666-6666-6666-6666-666666666666", // does not exist
	})
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 2 {
		t.Fatalf("got %d integrations, want 2 (missing ones are absent)", len(got))
	}

	full := got["44444444-4444-4444-4444-444444444444"]
	switch {
	case full == nil:
		t.Fatal("full integration missing")
	case !full.Requestable:
		t.Error("requestable not mapped")
	case full.WorkflowID != "3fa85f64-5717-4562-b3fc-2c963f66afa6":
		t.Errorf("workflow = %q", full.WorkflowID)
	case full.Owner == nil || full.Owner.ID != "00000000-0000-0000-0000-000000000002" || full.Owner.Email != "owner@org.domain":
		t.Errorf("owner = %+v", full.Owner)
	case len(full.Maintainers) != 2:
		t.Fatalf("maintainers = %+v", full.Maintainers)
	}
	if m := full.Maintainers[0]; m.Type != "group" || m.ID != "00000000-0000-0000-0000-000000000003" || m.Email != "team@org.domain" {
		t.Errorf("group maintainer = %+v", m)
	}
	if m := full.Maintainers[1]; m.Type != "user" || m.ID != "00000000-0000-0000-0000-000000000004" {
		t.Errorf("user maintainer = %+v", m)
	}

	empty := got["55555555-5555-5555-5555-555555555555"]
	if empty == nil || empty.Requestable || empty.WorkflowID != "" || empty.Owner != nil || len(empty.Maintainers) != 0 {
		t.Errorf("null fields not mapped to empty values: %+v", empty)
	}
}

// IDs are sent at most 100 per request, without integrationId, and looked up case-insensitively.
func TestFetchIntegrationSettingsBatches(t *testing.T) {
	var (
		mu      sync.Mutex
		batches []int
	)
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Has("integrationId") {
			t.Errorf("integrationId must not be sent: %v", q)
		}
		ids := strings.Split(q.Get("ids"), ",")
		mu.Lock()
		batches = append(batches, len(ids))
		mu.Unlock()

		items := make([]map[string]any, len(ids))
		for i, id := range ids {
			items[i] = map[string]any{"id": strings.ToLower(id), "name": "n", "application": map[string]any{"name": "a"}, "requestable": true}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"result": items})
	})

	ids := make([]string, 250)
	for i := range ids {
		ids[i] = fmt.Sprintf("%08X-0000-4000-8000-000000000000", i+1) // upper case on purpose
	}

	got, err := fetchIntegrationSettings(context.Background(), c, 3, ids)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 250 {
		t.Fatalf("got %d integrations, want 250", len(got))
	}
	if _, ok := got[strings.ToLower(ids[0])]; !ok {
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

func TestFetchIntegrationSettingsError(t *testing.T) {
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"errorId": "forbidden", "message": "not allowed"}`))
	})

	_, err := fetchIntegrationSettings(context.Background(), c, 1, []string{"44444444-4444-4444-4444-444444444444"})
	if err == nil || !strings.Contains(err.Error(), "listing integrations") {
		t.Fatalf("expected a listing error, got %v", err)
	}
}

// The resource submits to PATCH /public/v1/integrations/bulk with only the given fields.
func TestIntegrationSettingsSubmit(t *testing.T) {
	var gotPath, gotMethod string
	var gotBody map[string]any
	c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.Path, r.Method
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"result": {"jobId": "00000000-0000-0000-0000-000000000009", "count": 1}}`))
	})

	var meta resource.MetadataResponse
	NewIntegrationSettingsResource().Metadata(context.Background(), resource.MetadataRequest{ProviderTypeName: "entitle"}, &meta)
	if meta.TypeName != "entitle_integration_settings" {
		t.Errorf("type name = %q", meta.TypeName)
	}

	requestable := false
	submit := integrationSettingsKind().Submit(c, client.BulkSettingsUpdate{Requestable: &requestable})
	resp, err := submit(context.Background(), []string{"44444444-4444-4444-4444-444444444444"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode() != http.StatusAccepted || resp.JSON202 == nil {
		t.Fatalf("status %d, job %+v", resp.StatusCode(), resp.JSON202)
	}
	if gotMethod != http.MethodPatch || gotPath != "/public/v1/integrations/bulk" {
		t.Errorf("request = %s %s", gotMethod, gotPath)
	}

	update, _ := gotBody["update"].(map[string]any)
	if len(update) != 1 || update["requestable"] != false {
		t.Errorf("update body = %v, want only requestable=false", update)
	}
	if ids, _ := gotBody["ids"].([]any); len(ids) != 1 {
		t.Errorf("ids = %v", gotBody["ids"])
	}
}
