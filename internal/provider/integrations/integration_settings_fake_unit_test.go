package integrations_test

// In-memory fake of the integrations endpoints used by entitle_integration_settings:
// PATCH /public/v1/integrations/bulk, GET /public/v1/integrations?ids=... and
// GET /public/v1/bulkOperations/{jobId}.

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/entitleio/terraform-provider-entitle/internal/client"
	"github.com/entitleio/terraform-provider-entitle/internal/provider"
)

var unitTestProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"entitle": providerserver.NewProtocol6WithError(provider.New("test")()),
}

// identity is a user or group; an email resolves to a stable UUID like the real API.
type identity struct {
	ID    string
	Email string
}

func resolve(ref string) identity {
	if strings.Contains(ref, "@") {
		email := strings.ToLower(ref)
		return identity{ID: uuid.NewSHA1(uuid.NameSpaceOID, []byte(email)).String(), Email: email}
	}
	return identity{ID: strings.ToLower(ref)}
}

func (i identity) json() map[string]any {
	out := map[string]any{"id": i.ID, "email": nil}
	if i.Email != "" {
		out["email"] = i.Email
	}
	return out
}

type maintainer struct {
	Type string
	identity
}

func (m maintainer) String() string { return m.Type + ":" + m.ID }

type fakeIntegration struct {
	Requestable bool
	WorkflowID  string
	Owner       *identity
	Maintainers []maintainer
}

type fakeAPI struct {
	t  *testing.T
	mu sync.Mutex

	integrations map[string]*fakeIntegration
	failNextJob  bool
	failedJobs   map[string]bool
	calls        []client.BulkUpdateIntegrationsRequest
	rawBodies    []map[string]any
	lists        []url.Values
	other        []string // any request the resource must not make
}

func setupFake(t *testing.T, ids ...string) *fakeAPI {
	t.Helper()
	f := &fakeAPI{t: t, integrations: map[string]*fakeIntegration{}, failedJobs: map[string]bool{}}
	for _, id := range ids {
		f.integrations[id] = &fakeIntegration{Requestable: true}
	}

	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)
	t.Setenv("ENTITLE_API_ENDPOINT", srv.URL)
	t.Setenv("ENTITLE_API_KEY", "test-token")
	return f
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func (f *fakeAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.Method == http.MethodPatch && r.URL.Path == "/public/v1/integrations/bulk":
		f.bulk(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/public/v1/integrations":
		f.list(w, r)
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/public/v1/bulkOperations/"):
		status := "completed"
		result := map[string]any{"processed": 1, "total": 1}
		if f.failedJobs[strings.TrimPrefix(r.URL.Path, "/public/v1/bulkOperations/")] {
			status, result["error"] = "failed", "simulated failure"
		}
		result["status"] = status
		writeJSON(w, http.StatusOK, map[string]any{"result": result})
	default:
		f.other = append(f.other, r.Method+" "+r.URL.String())
		writeJSON(w, http.StatusNotFound, map[string]any{"errorId": "notFound", "message": "unexpected request"})
	}
}

func (f *fakeAPI) list(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f.lists = append(f.lists, q)

	var ids []string
	for _, v := range q["ids"] {
		for _, id := range strings.Split(v, ",") {
			if id = strings.TrimSpace(id); id != "" {
				ids = append(ids, strings.ToLower(id))
			}
		}
	}
	if len(ids) > 100 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"errorId": "validation", "message": "ids: at most 100 ids"})
		return
	}

	keys := make([]string, 0, len(f.integrations))
	for id := range f.integrations {
		if len(ids) == 0 || slices.Contains(ids, id) {
			keys = append(keys, id)
		}
	}
	sort.Strings(keys)

	result := []any{}
	for _, id := range keys {
		in := f.integrations[id]
		maintainers := []any{}
		for _, m := range in.Maintainers {
			maintainers = append(maintainers, map[string]any{"type": m.Type, m.Type: m.identity.json()})
		}
		item := map[string]any{
			"id":          id,
			"name":        "integration " + id,
			"application": map[string]any{"name": "Manual"},
			"requestable": in.Requestable,
			"workflow":    nil,
			"owner":       nil,
			"maintainers": maintainers,
		}
		if in.WorkflowID != "" {
			item["workflow"] = map[string]any{"id": in.WorkflowID, "name": "workflow"}
		}
		if in.Owner != nil {
			item["owner"] = in.Owner.json()
		}
		result = append(result, item)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"pagination": map[string]any{"page": 1, "perPage": 100, "totalResults": len(result), "totalPages": 1},
		"result":     result,
	})
}

func toMaintainers(in []client.BulkMaintainer) []maintainer {
	out := make([]maintainer, 0, len(in))
	for _, m := range in {
		ref := m.User
		if m.Type == "group" {
			ref = m.Group
		}
		if ref != nil {
			out = append(out, maintainer{Type: m.Type, identity: resolve(ref.Id)})
		}
	}
	return out
}

func (f *fakeAPI) bulk(w http.ResponseWriter, r *http.Request) {
	raw, _ := io.ReadAll(r.Body)
	var body map[string]any
	var req client.BulkUpdateIntegrationsRequest
	if err := json.Unmarshal(raw, &body); err != nil {
		f.t.Errorf("decoding bulk request: %v", err)
	}
	if err := json.Unmarshal(raw, &req); err != nil {
		f.t.Errorf("decoding bulk request: %v", err)
	}

	var missing []string
	for _, id := range req.Ids {
		if _, ok := f.integrations[id]; !ok {
			missing = append(missing, id)
		}
	}
	if len(missing) > 0 {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"errorId": "integration.notFound", "message": "some integrations were not found",
			"params": map[string]any{"ids": missing},
		})
		return
	}

	u := req.Update
	if u.Maintainers != nil && u.Maintainers.Replace != nil && (u.Maintainers.Add != nil || u.Maintainers.Remove != nil) {
		f.t.Errorf("maintainers.replace must not be combined with add/remove")
	}
	for _, id := range req.Ids {
		in := f.integrations[id]
		if u.Requestable != nil {
			in.Requestable = *u.Requestable
		}
		if u.Workflow != nil {
			in.WorkflowID = strings.ToLower(u.Workflow.Id)
		}
		if u.Owner != nil {
			o := resolve(u.Owner.Id)
			in.Owner = &o
		}
		if m := u.Maintainers; m != nil {
			if m.Replace != nil {
				in.Maintainers = toMaintainers(*m.Replace)
			}
			if m.Remove != nil {
				for _, rm := range toMaintainers(*m.Remove) {
					in.Maintainers = slices.DeleteFunc(in.Maintainers, func(x maintainer) bool { return x.Type == rm.Type && x.ID == rm.ID })
				}
			}
			if m.Add != nil {
				for _, add := range toMaintainers(*m.Add) {
					if !slices.ContainsFunc(in.Maintainers, func(x maintainer) bool { return x.Type == add.Type && x.ID == add.ID }) {
						in.Maintainers = append(in.Maintainers, add)
					}
				}
			}
		}
	}

	jobID := uuid.NewString()
	if f.failNextJob {
		f.failNextJob = false
		f.failedJobs[jobID] = true
	}
	f.calls = append(f.calls, req)
	f.rawBodies = append(f.rawBodies, body)
	writeJSON(w, http.StatusAccepted, map[string]any{"result": map[string]any{"jobId": jobID, "count": len(req.Ids)}})
}

// --- test controls -----------------------------------------------------------

func (f *fakeAPI) update(id string, fn func(*fakeIntegration)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	fn(f.integrations[id])
}

func (f *fakeAPI) failNext() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.failNextJob = true
}

func (f *fakeAPI) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

// --- checks ------------------------------------------------------------------

func (f *fakeAPI) check(id string, fn func(fakeIntegration) error) resource.TestCheckFunc {
	return func(*terraform.State) error {
		f.mu.Lock()
		defer f.mu.Unlock()
		in, ok := f.integrations[id]
		if !ok {
			return fmt.Errorf("integration %s not found in fake API", id)
		}
		if err := fn(*in); err != nil {
			return fmt.Errorf("integration %s: %w", id, err)
		}
		return nil
	}
}

// checkRequests verifies the requests made so far: only the integrations
// endpoints are used, reads go through the list with ids (no integrationId),
// and no bulk body ever contains tags.
func (f *fakeAPI) checkRequests() resource.TestCheckFunc {
	return func(*terraform.State) error {
		f.mu.Lock()
		defer f.mu.Unlock()
		if len(f.other) > 0 {
			return fmt.Errorf("unexpected requests: %v", f.other)
		}
		for _, q := range f.lists {
			if q.Get("ids") == "" || q.Has("integrationId") {
				return fmt.Errorf("unexpected list query %v", q)
			}
		}
		for _, b := range f.rawBodies {
			if update, _ := b["update"].(map[string]any); update != nil {
				if _, ok := update["tags"]; ok {
					return fmt.Errorf("tags sent to the integrations bulk endpoint: %v", b)
				}
			}
		}
		return nil
	}
}

// expectUpdates checks the bulk requests since *mark: each listed integration
// received exactly one update, accepted by its check; no other integration was updated.
func (f *fakeAPI) expectUpdates(mark *int, want map[string]func(client.BulkSettingsUpdate) error) resource.TestCheckFunc {
	return func(*terraform.State) error {
		f.mu.Lock()
		defer f.mu.Unlock()

		got := map[string][]client.BulkSettingsUpdate{}
		for _, call := range f.calls[*mark:] {
			for _, id := range call.Ids {
				got[id] = append(got[id], call.Update)
			}
		}
		for id := range got {
			if _, ok := want[id]; !ok {
				return fmt.Errorf("unexpected update for %s: %+v", id, got[id])
			}
		}
		for id, fn := range want {
			if len(got[id]) != 1 {
				return fmt.Errorf("integration %s: got %d updates, want 1", id, len(got[id]))
			}
			if err := fn(got[id][0]); err != nil {
				return fmt.Errorf("integration %s: %w", id, err)
			}
		}
		return nil
	}
}

func (f *fakeAPI) expectNoUpdates(mark *int) resource.TestCheckFunc {
	return func(*terraform.State) error {
		if n := f.callCount(); n != *mark {
			return fmt.Errorf("expected no bulk requests, got %d", n-*mark)
		}
		return nil
	}
}
