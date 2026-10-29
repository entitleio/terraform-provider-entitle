package resources_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/entitleio/terraform-provider-entitle/internal/client"
)

// fakeEntitle is an in-memory stand-in for the resources and bulk endpoints,
// shared by the unit tests of the bulk resources.

// fakeEntity is a user or group. Identifiers sent to the fake (UUID or email)
// are resolved the way the real API does: an email maps to a stable UUID.
type fakeEntity struct {
	ID    string
	Email string
}

func resolveEntity(ref string) fakeEntity {
	if strings.Contains(ref, "@") {
		return fakeEntity{ID: uuid.NewSHA1(uuid.NameSpaceOID, []byte(strings.ToLower(ref))).String(), Email: strings.ToLower(ref)}
	}
	return fakeEntity{ID: strings.ToLower(ref)}
}

type fakeMaintainer struct {
	Type string // "user" or "group"
	fakeEntity
}

func (m fakeMaintainer) String() string { return m.Type + ":" + m.ID }

// fakeResource holds the attributes the bulk resources manage.
type fakeResource struct {
	Tags        []string
	Requestable bool
	WorkflowID  string
	Owner       *fakeEntity
	Maintainers []fakeMaintainer
}

type fakeEntitle struct {
	t  *testing.T
	mu sync.Mutex

	resources     map[string]*fakeResource
	conflictsLeft int             // 409 responses to return before accepting bulk requests
	bulkNotFound  map[string]bool // readable resources the bulk endpoint rejects with 404 (e.g. internal integrations)
	failNextJob   bool            // next accepted job applies its changes but reports "failed"
	failedJobs    map[string]bool
	calls         []client.BulkUpdateResourcesRequest
	listCalls     []url.Values // queries of GET /public/v1/resources
	singleGets    int          // GET /public/v1/resources/{id} requests
}

func newFakeEntitle(t *testing.T, tags map[string][]string) *fakeEntitle {
	f := &fakeEntitle{t: t, resources: map[string]*fakeResource{}, bulkNotFound: map[string]bool{}, failedJobs: map[string]bool{}}
	for id, tt := range tags {
		f.resources[id] = &fakeResource{Tags: tt, Requestable: true}
	}
	return f
}

func (f *fakeEntitle) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")

	switch {
	case r.Method == http.MethodPatch && r.URL.Path == "/public/v1/resources/bulk":
		f.bulkUpdate(w, r)

	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/public/v1/bulkOperations/"):
		jobID := strings.TrimPrefix(r.URL.Path, "/public/v1/bulkOperations/")
		result := map[string]any{"status": "completed", "processed": 1, "total": 1}
		if f.failedJobs[jobID] {
			result["status"] = "failed"
			result["error"] = "simulated failure"
		}
		writeJSON(w, http.StatusOK, map[string]any{"result": result})

	case r.Method == http.MethodGet && r.URL.Path == "/public/v1/resources":
		f.listResources(w, r)

	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/public/v1/resources/"):
		f.singleGets++
		id := strings.TrimPrefix(r.URL.Path, "/public/v1/resources/")
		res, ok := f.resources[id]
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]any{"errorId": "resource.notFound", "message": "resource not found"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"result": res.toJSON(id)})

	default:
		f.t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func entityJSON(e fakeEntity) map[string]any {
	out := map[string]any{"id": e.ID, "email": nil}
	if e.Email != "" {
		out["email"] = e.Email
	}
	return out
}

func (res *fakeResource) toJSON(id string) map[string]any {
	maintainers := make([]any, 0, len(res.Maintainers))
	for _, m := range res.Maintainers {
		maintainers = append(maintainers, map[string]any{"type": m.Type, m.Type: entityJSON(m.fakeEntity)})
	}

	out := map[string]any{
		"id":               id,
		"name":             "resource " + id,
		"requestable":      res.Requestable,
		"allowedDurations": []any{},
		"maintainers":      maintainers,
		"tags":             []string{},
		"userDefinedTags":  res.Tags,
		"workflow":         nil,
		"owner":            nil,
	}
	if res.WorkflowID != "" {
		out["workflow"] = map[string]any{"id": res.WorkflowID, "name": "workflow"}
	}
	if res.Owner != nil {
		out["owner"] = entityJSON(*res.Owner)
	}
	return out
}

// listResources emulates GET /public/v1/resources with the ids filter: up to
// 100 ids (comma-separated or repeated), idsOperator is/isNot, full items
// sorted by id. integrationId is not required when ids is given.
func (f *fakeEntitle) listResources(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f.listCalls = append(f.listCalls, q)

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
	if len(ids) == 0 && q.Get("integrationId") == "" {
		writeJSON(w, http.StatusBadRequest, map[string]any{"errorId": "validation", "message": "integrationId is required"})
		return
	}

	selected := map[string]bool{}
	for _, id := range ids {
		selected[id] = true
	}
	isNot := q.Get("idsOperator") == "isNot"

	keys := make([]string, 0, len(f.resources))
	for id := range f.resources {
		keys = append(keys, id)
	}
	sort.Strings(keys)

	result := []any{}
	for _, id := range keys {
		if len(ids) > 0 && selected[strings.ToLower(id)] == isNot {
			continue
		}
		result = append(result, f.resources[id].toJSON(id))
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"pagination": map[string]any{"page": 1, "perPage": 100, "totalResults": len(result), "totalPages": 1},
		"result":     result,
	})
}

// requests returns how many list calls and single-resource GETs were made.
func (f *fakeEntitle) requests() (lists []url.Values, singles int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.listCalls), f.singleGets
}

func toFakeMaintainers(in []client.BulkMaintainer) []fakeMaintainer {
	out := make([]fakeMaintainer, 0, len(in))
	for _, m := range in {
		ref := m.User
		if m.Type == "group" {
			ref = m.Group
		}
		if ref == nil {
			continue
		}
		out = append(out, fakeMaintainer{Type: m.Type, fakeEntity: resolveEntity(ref.Id)})
	}
	return out
}

func (f *fakeEntitle) bulkUpdate(w http.ResponseWriter, r *http.Request) {
	var req client.BulkUpdateResourcesRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		f.t.Errorf("decoding bulk request: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if f.conflictsLeft > 0 {
		f.conflictsLeft--
		writeJSON(w, http.StatusConflict, map[string]any{"errorId": "bulkOperation.inProgress", "message": "another bulk operation is running"})
		return
	}

	// All-or-nothing validation, like the real API.
	var missing []string
	for _, id := range req.Ids {
		if _, ok := f.resources[id]; !ok || f.bulkNotFound[id] {
			missing = append(missing, id)
		}
	}
	if len(missing) > 0 {
		writeJSON(w, http.StatusNotFound, map[string]any{
			"errorId": "resource.notFound",
			"message": "some resources were not found",
			"params":  map[string]any{"ids": missing},
		})
		return
	}

	u := req.Update
	if u.Maintainers != nil && u.Maintainers.Replace != nil && (u.Maintainers.Add != nil || u.Maintainers.Remove != nil) {
		f.t.Errorf("maintainers.replace must not be combined with add/remove: %+v", u.Maintainers)
	}

	for _, id := range req.Ids {
		res := f.resources[id]

		if u.Tags != nil && u.Tags.Remove != nil {
			res.Tags = slices.DeleteFunc(res.Tags, func(tag string) bool { return slices.Contains(*u.Tags.Remove, tag) })
		}
		if u.Tags != nil && u.Tags.Add != nil {
			for _, tag := range *u.Tags.Add {
				if !slices.Contains(res.Tags, tag) {
					res.Tags = append(res.Tags, tag)
				}
			}
		}
		if u.Requestable != nil {
			res.Requestable = *u.Requestable
		}
		if u.Workflow != nil {
			res.WorkflowID = strings.ToLower(u.Workflow.Id)
		}
		if u.Owner != nil {
			owner := resolveEntity(u.Owner.Id)
			res.Owner = &owner
		}
		if u.Maintainers != nil {
			if u.Maintainers.Replace != nil {
				res.Maintainers = toFakeMaintainers(*u.Maintainers.Replace)
			}
			if u.Maintainers.Remove != nil {
				for _, rm := range toFakeMaintainers(*u.Maintainers.Remove) {
					res.Maintainers = slices.DeleteFunc(res.Maintainers, func(m fakeMaintainer) bool {
						return m.Type == rm.Type && m.ID == rm.ID
					})
				}
			}
			if u.Maintainers.Add != nil {
				for _, add := range toFakeMaintainers(*u.Maintainers.Add) {
					if !slices.ContainsFunc(res.Maintainers, func(m fakeMaintainer) bool { return m.Type == add.Type && m.ID == add.ID }) {
						res.Maintainers = append(res.Maintainers, add)
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
	writeJSON(w, http.StatusAccepted, map[string]any{
		"result": map[string]any{"jobId": jobID, "count": len(req.Ids)},
	})
}

// --- test controls ---------------------------------------------------------

func (f *fakeEntitle) setTags(id string, tags ...string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if res, ok := f.resources[id]; ok {
		res.Tags = tags
		return
	}
	f.resources[id] = &fakeResource{Tags: tags, Requestable: true}
}

// update changes a resource outside Terraform.
func (f *fakeEntitle) update(id string, fn func(r *fakeResource)) {
	f.mu.Lock()
	defer f.mu.Unlock()
	fn(f.resources[id])
}

func (f *fakeEntitle) deleteResource(id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.resources, id)
}

// hideFromBulk makes the bulk endpoint reject id with 404 while it stays readable.
func (f *fakeEntitle) hideFromBulk(id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.bulkNotFound[id] = true
}

// failNext makes the next accepted bulk job apply its changes and then report "failed".
func (f *fakeEntitle) failNext() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.failNextJob = true
}

func (f *fakeEntitle) conflicts() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.conflictsLeft
}

func (f *fakeEntitle) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

func (f *fakeEntitle) callsSince(n int) []client.BulkUpdateResourcesRequest {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.calls[n:])
}

// --- checks ------------------------------------------------------------------

// checkTags verifies the exact (unordered) set of tags on a resource in the fake API.
func (f *fakeEntitle) checkTags(id string, want ...string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		f.mu.Lock()
		defer f.mu.Unlock()

		var got []string
		if res, ok := f.resources[id]; ok {
			got = slices.Clone(res.Tags)
		}
		sort.Strings(got)
		want = slices.Clone(want)
		sort.Strings(want)
		if !slices.Equal(got, want) {
			return fmt.Errorf("resource %s: tags = %v, want %v", id, got, want)
		}
		return nil
	}
}

// checkResource runs fn against a snapshot of the resource in the fake API.
func (f *fakeEntitle) checkResource(id string, fn func(r fakeResource) error) resource.TestCheckFunc {
	return func(*terraform.State) error {
		f.mu.Lock()
		defer f.mu.Unlock()

		res, ok := f.resources[id]
		if !ok {
			return fmt.Errorf("resource %s not found in fake API", id)
		}
		if err := fn(*res); err != nil {
			return fmt.Errorf("resource %s: %w", id, err)
		}
		return nil
	}
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func setupFakeEntitle(t *testing.T, tags map[string][]string) *fakeEntitle {
	t.Helper()

	fake := newFakeEntitle(t, tags)
	srv := httptest.NewServer(fake)
	t.Cleanup(srv.Close)

	// The endpoint attribute only accepts real Entitle URLs; the env var is not validated.
	t.Setenv("ENTITLE_API_ENDPOINT", srv.URL)
	t.Setenv("ENTITLE_API_KEY", "test-token")

	return fake
}
