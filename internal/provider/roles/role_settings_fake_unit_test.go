package roles_test

// In-memory fake of the roles endpoints used by entitle_role_settings:
// PATCH /public/v1/roles/bulk, GET /public/v1/roles?ids=... and
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

var roleSettingsProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"entitle": providerserver.NewProtocol6WithError(provider.New("test")()),
}

type fakeRole struct {
	Requestable bool
	WorkflowID  string
}

type fakeRolesAPI struct {
	t  *testing.T
	mu sync.Mutex

	roles       map[string]*fakeRole
	failNextJob bool
	failedJobs  map[string]bool
	calls       []client.BulkUpdateRolesRequest
	rawUpdates  []map[string]any
	lists       []url.Values
	other       []string // any request the resource must not make
}

func setupFakeRoles(t *testing.T, ids ...string) *fakeRolesAPI {
	t.Helper()
	f := &fakeRolesAPI{t: t, roles: map[string]*fakeRole{}, failedJobs: map[string]bool{}}
	for _, id := range ids {
		f.roles[id] = &fakeRole{Requestable: true}
	}

	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)
	t.Setenv("ENTITLE_API_ENDPOINT", srv.URL)
	t.Setenv("ENTITLE_API_KEY", "test-token")
	return f
}

func writeRolesJSON(w http.ResponseWriter, status int, body any) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func (f *fakeRolesAPI) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.Method == http.MethodPatch && r.URL.Path == "/public/v1/roles/bulk":
		f.bulk(w, r)
	case r.Method == http.MethodGet && r.URL.Path == "/public/v1/roles":
		f.list(w, r)
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/public/v1/bulkOperations/"):
		result := map[string]any{"status": "completed", "processed": 1, "total": 1}
		if f.failedJobs[strings.TrimPrefix(r.URL.Path, "/public/v1/bulkOperations/")] {
			result["status"], result["error"] = "failed", "simulated failure"
		}
		writeRolesJSON(w, http.StatusOK, map[string]any{"result": result})
	default:
		f.other = append(f.other, r.Method+" "+r.URL.String())
		writeRolesJSON(w, http.StatusNotFound, map[string]any{"errorId": "notFound", "message": "unexpected request"})
	}
}

func (f *fakeRolesAPI) list(w http.ResponseWriter, r *http.Request) {
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
		writeRolesJSON(w, http.StatusBadRequest, map[string]any{"errorId": "validation", "message": "ids: at most 100 ids"})
		return
	}

	keys := make([]string, 0, len(f.roles))
	for id := range f.roles {
		if slices.Contains(ids, id) {
			keys = append(keys, id)
		}
	}
	sort.Strings(keys)

	result := []any{}
	for _, id := range keys {
		role := f.roles[id]
		item := map[string]any{
			"id":          id,
			"name":        "role " + id,
			"externalId":  nil,
			"requestable": role.Requestable,
			"workflow":    nil,
			"resource": map[string]any{
				"id": "77777777-7777-7777-7777-777777777777", "name": "resource", "type": "t", "externalId": "x",
				"integration": map[string]any{"id": "88888888-8888-8888-8888-888888888888", "name": "integration", "application": map[string]any{"name": "Manual"}},
			},
		}
		if role.WorkflowID != "" {
			item["workflow"] = map[string]any{"id": role.WorkflowID, "name": "workflow"}
		}
		result = append(result, item)
	}
	writeRolesJSON(w, http.StatusOK, map[string]any{
		"pagination": map[string]any{"page": 1, "perPage": 100, "totalResults": len(result), "totalPages": 1},
		"result":     result,
	})
}

func (f *fakeRolesAPI) bulk(w http.ResponseWriter, r *http.Request) {
	raw, _ := io.ReadAll(r.Body)
	var body struct {
		Update map[string]any `json:"update"`
	}
	var req client.BulkUpdateRolesRequest
	if err := json.Unmarshal(raw, &body); err != nil {
		f.t.Errorf("decoding bulk request: %v", err)
	}
	if err := json.Unmarshal(raw, &req); err != nil {
		f.t.Errorf("decoding bulk request: %v", err)
	}

	var missing []string
	for _, id := range req.Ids {
		if _, ok := f.roles[id]; !ok {
			missing = append(missing, id)
		}
	}
	if len(missing) > 0 {
		writeRolesJSON(w, http.StatusNotFound, map[string]any{
			"errorId": "role.notFound", "message": "some roles were not found",
			"params": map[string]any{"ids": missing},
		})
		return
	}

	for _, id := range req.Ids {
		role := f.roles[id]
		if req.Update.Requestable != nil {
			role.Requestable = *req.Update.Requestable
		}
		if req.Update.Workflow != nil {
			role.WorkflowID = strings.ToLower(req.Update.Workflow.Id)
		}
	}

	jobID := uuid.NewString()
	if f.failNextJob {
		f.failNextJob = false
		f.failedJobs[jobID] = true
	}
	f.calls = append(f.calls, req)
	f.rawUpdates = append(f.rawUpdates, body.Update)
	writeRolesJSON(w, http.StatusAccepted, map[string]any{"result": map[string]any{"jobId": jobID, "count": len(req.Ids)}})
}

// --- test controls -----------------------------------------------------------

func (f *fakeRolesAPI) update(id string, fn func(*fakeRole)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	fn(f.roles[id])
}

func (f *fakeRolesAPI) failNext() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.failNextJob = true
}

func (f *fakeRolesAPI) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

// --- checks ------------------------------------------------------------------

func (f *fakeRolesAPI) check(id string, requestable bool, workflowID string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		f.mu.Lock()
		defer f.mu.Unlock()
		role, ok := f.roles[id]
		if !ok {
			return fmt.Errorf("role %s not found in fake API", id)
		}
		if role.Requestable != requestable || role.WorkflowID != workflowID {
			return fmt.Errorf("role %s: requestable=%v workflow=%q, want requestable=%v workflow=%q",
				id, role.Requestable, role.WorkflowID, requestable, workflowID)
		}
		return nil
	}
}

// checkRequests verifies the requests made so far: only the roles endpoints
// are used, reads go through the list with ids, and every bulk update contains
// nothing but requestable and workflow.
func (f *fakeRolesAPI) checkRequests() resource.TestCheckFunc {
	return func(*terraform.State) error {
		f.mu.Lock()
		defer f.mu.Unlock()
		if len(f.other) > 0 {
			return fmt.Errorf("unexpected requests: %v", f.other)
		}
		for _, q := range f.lists {
			if q.Get("ids") == "" || q.Has("integrationId") || q.Has("resourceId") {
				return fmt.Errorf("unexpected list query %v", q)
			}
		}
		for _, u := range f.rawUpdates {
			for k := range u {
				if k != "requestable" && k != "workflow" {
					return fmt.Errorf("unexpected field %q sent to the roles bulk endpoint: %v", k, u)
				}
			}
		}
		return nil
	}
}

// updatesSince returns, per role, the raw update bodies received since mark.
func (f *fakeRolesAPI) updatesSince(mark int) map[string][]map[string]any {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := map[string][]map[string]any{}
	for i, call := range f.calls[mark:] {
		for _, id := range call.Ids {
			out[id] = append(out[id], f.rawUpdates[mark+i])
		}
	}
	return out
}

// expectUpdates checks that exactly the given roles were updated since *mark,
// each once, with exactly the given top-level fields.
func (f *fakeRolesAPI) expectUpdates(mark *int, want map[string][]string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		got := f.updatesSince(*mark)
		if len(got) != len(want) {
			return fmt.Errorf("updated roles = %v, want %v", got, want)
		}
		for id, fields := range want {
			if len(got[id]) != 1 {
				return fmt.Errorf("role %s: got %d updates, want 1", id, len(got[id]))
			}
			var keys []string
			for k := range got[id][0] {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			exp := slices.Clone(fields)
			sort.Strings(exp)
			if !slices.Equal(keys, exp) {
				return fmt.Errorf("role %s: sent fields %v, want %v", id, keys, exp)
			}
		}
		return nil
	}
}
