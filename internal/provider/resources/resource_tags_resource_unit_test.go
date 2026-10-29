package resources_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"slices"
	"sort"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/entitleio/terraform-provider-entitle/internal/client"
	"github.com/entitleio/terraform-provider-entitle/internal/provider"
)

// Unit tests for entitle_resource_tags. They run the real provider through
// Terraform against an in-memory fake of the Entitle API, so no credentials
// are needed (the Terraform CLI must be available or downloadable).

const (
	tagsR1 = "11111111-1111-1111-1111-111111111111"
	tagsR2 = "22222222-2222-2222-2222-222222222222"
	tagsR3 = "33333333-3333-3333-3333-333333333333"
)

// fakeEntitle is an in-memory stand-in for the resources and bulk endpoints.
type fakeEntitle struct {
	t  *testing.T
	mu sync.Mutex

	tags          map[string][]string // resource id -> userDefinedTags
	conflictsLeft int                 // 409 responses to return before accepting bulk requests
	bulkNotFound  map[string]bool     // readable resources the bulk endpoint rejects with 404 (e.g. internal integrations)
	failNextJob   bool                // next accepted job applies its changes but reports "failed"
	failedJobs    map[string]bool
	calls         []client.BulkUpdateResourcesRequest
}

func newFakeEntitle(t *testing.T, tags map[string][]string) *fakeEntitle {
	return &fakeEntitle{t: t, tags: tags, bulkNotFound: map[string]bool{}, failedJobs: map[string]bool{}}
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

	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/public/v1/resources/"):
		id := strings.TrimPrefix(r.URL.Path, "/public/v1/resources/")
		tags, ok := f.tags[id]
		if !ok {
			writeJSON(w, http.StatusNotFound, map[string]any{"errorId": "resource.notFound", "message": "resource not found"})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"result": map[string]any{
				"id":               id,
				"name":             "resource " + id,
				"requestable":      true,
				"allowedDurations": []any{},
				"maintainers":      []any{},
				"tags":             []string{},
				"userDefinedTags":  tags,
			},
		})

	default:
		f.t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
	}
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
		if _, ok := f.tags[id]; !ok || f.bulkNotFound[id] {
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

	for _, id := range req.Ids {
		current := f.tags[id]
		if req.Update.Tags != nil && req.Update.Tags.Remove != nil {
			current = slices.DeleteFunc(current, func(tag string) bool {
				return slices.Contains(*req.Update.Tags.Remove, tag)
			})
		}
		if req.Update.Tags != nil && req.Update.Tags.Add != nil {
			for _, tag := range *req.Update.Tags.Add {
				if !slices.Contains(current, tag) {
					current = append(current, tag)
				}
			}
		}
		f.tags[id] = current
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

func (f *fakeEntitle) setTags(id string, tags ...string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.tags[id] = tags
}

func (f *fakeEntitle) deleteResource(id string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.tags, id)
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

// checkTags verifies the exact (unordered) set of tags on a resource in the fake API.
func (f *fakeEntitle) checkTags(id string, want ...string) resource.TestCheckFunc {
	return func(*terraform.State) error {
		f.mu.Lock()
		defer f.mu.Unlock()

		got := slices.Clone(f.tags[id])
		sort.Strings(got)
		want = slices.Clone(want)
		sort.Strings(want)
		if !slices.Equal(got, want) {
			return fmt.Errorf("resource %s: tags = %v, want %v", id, got, want)
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

var unitTestProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"entitle": providerserver.NewProtocol6WithError(provider.New("test")()),
}

// chunk_size = 1 forces one bulk request per resource, exercising chunking.
const tagsProviderConfig = `
provider "entitle" {
  bulk {
    chunk_size  = 1
    parallelism = 1
  }
}
`

func tagsConfig(ids []string, tags []string) string {
	return tagsProviderConfig + fmt.Sprintf(`
resource "entitle_resource_tags" "test" {
  resource_ids = %s
  tags         = %s
}
`, hclList(ids), hclList(tags))
}

func hclList(items []string) string {
	quoted := make([]string, len(items))
	for i, it := range items {
		quoted[i] = fmt.Sprintf("%q", it)
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}

func TestUnitResourceTagsResource_Lifecycle(t *testing.T) {
	fake := setupFakeEntitle(t, map[string][]string{
		tagsR1: {"keep"},
		tagsR2: {"keep"},
		tagsR3: {"keep"},
	})
	fake.conflictsLeft = 1 // first submission hits "another bulk operation is running"

	var mark int

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: unitTestProviderFactories,
		Steps: []resource.TestStep{
			// Create: tags added to every resource, existing tags preserved,
			// one request per chunk, 409 retried transparently.
			{
				Config: tagsConfig([]string{tagsR1, tagsR2}, []string{"pci", "sox"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("entitle_resource_tags.test", "resource_ids.#", "2"),
					resource.TestCheckTypeSetElemAttr("entitle_resource_tags.test", "resource_ids.*", tagsR1),
					resource.TestCheckTypeSetElemAttr("entitle_resource_tags.test", "resource_ids.*", tagsR2),
					resource.TestCheckResourceAttr("entitle_resource_tags.test", "tags.#", "2"),
					resource.TestCheckNoResourceAttr("entitle_resource_tags.test", "id"),
					fake.checkTags(tagsR1, "keep", "pci", "sox"),
					fake.checkTags(tagsR2, "keep", "pci", "sox"),
					fake.checkTags(tagsR3, "keep"),
					func(*terraform.State) error {
						if n := fake.callCount(); n != 2 {
							return fmt.Errorf("expected 2 accepted bulk requests (chunk_size = 1), got %d", n)
						}
						if left := fake.conflicts(); left != 0 {
							return fmt.Errorf("expected conflicts to be retried, %d left", left)
						}
						return nil
					},
				),
			},
			// Update: r1 leaves (tags removed), r2 stays (only tag diff applied),
			// r3 joins (all tags added); "sox" replaced by "hipaa".
			{
				PreConfig: func() { mark = fake.callCount() },
				Config:    tagsConfig([]string{tagsR2, tagsR3}, []string{"pci", "hipaa"}),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("entitle_resource_tags.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					fake.checkTags(tagsR1, "keep"),
					fake.checkTags(tagsR2, "keep", "pci", "hipaa"),
					fake.checkTags(tagsR3, "keep", "pci", "hipaa"),
					func(*terraform.State) error {
						for _, call := range fake.callsSince(mark) {
							if slices.Contains(call.Ids, tagsR2) && call.Update.Tags.Add != nil &&
								slices.Contains(*call.Update.Tags.Add, "pci") {
								return fmt.Errorf("unchanged tag %q was re-sent for kept resource %s", "pci", tagsR2)
							}
						}
						return nil
					},
				),
			},
			// Drift: a tag removed outside Terraform is detected on refresh and re-added
			// with a request for the drifted resource only.
			{
				PreConfig: func() {
					fake.setTags(tagsR3, "keep", "hipaa")
					mark = fake.callCount()
				},
				Config: tagsConfig([]string{tagsR2, tagsR3}, []string{"pci", "hipaa"}),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectResourceAction("entitle_resource_tags.test", plancheck.ResourceActionUpdate),
					},
				},
				Check: resource.ComposeAggregateTestCheckFunc(
					fake.checkTags(tagsR3, "keep", "pci", "hipaa"),
					func(*terraform.State) error {
						calls := fake.callsSince(mark)
						if len(calls) != 1 || !slices.Equal(calls[0].Ids, []string{tagsR3}) {
							return fmt.Errorf("expected a single request for %s, got %+v", tagsR3, calls)
						}
						return nil
					},
				),
			},
			// A resource deleted outside Terraform but still in config: the API rejects
			// the request and the error names the offending ID.
			{
				PreConfig:   func() { fake.deleteResource(tagsR2) },
				Config:      tagsConfig([]string{tagsR2, tagsR3}, []string{"pci", "hipaa"}),
				ExpectError: regexp.MustCompile(tagsR2),
			},
			// Dropping the deleted resource from config converges without changes to r3.
			{
				Config: tagsConfig([]string{tagsR3}, []string{"pci", "hipaa"}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("entitle_resource_tags.test", "resource_ids.#", "1"),
					fake.checkTags(tagsR3, "keep", "pci", "hipaa"),
				),
			},
		},
		// Destroy removes only the managed tags.
		CheckDestroy: resource.ComposeAggregateTestCheckFunc(
			fake.checkTags(tagsR1, "keep"),
			fake.checkTags(tagsR3, "keep"),
		),
	})
}

func TestUnitResourceTagsResource_ToleratesPreexistingAndForeignTags(t *testing.T) {
	fake := setupFakeEntitle(t, map[string][]string{
		tagsR1: {"pci", "other"},
	})

	resource.UnitTest(t, resource.TestCase{
		ProtoV6ProviderFactories: unitTestProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: tagsConfig([]string{tagsR1}, []string{"pci", "sox"}),
				Check:  fake.checkTags(tagsR1, "pci", "other", "sox"),
			},
			// Tags added outside Terraform do not cause drift.
			{
				PreConfig: func() { fake.setTags(tagsR1, "pci", "other", "sox", "new") },
				Config:    tagsConfig([]string{tagsR1}, []string{"pci", "sox"}),
				ConfigPlanChecks: resource.ConfigPlanChecks{
					PreApply: []plancheck.PlanCheck{
						plancheck.ExpectEmptyPlan(),
					},
				},
			},
		},
		// "pci" existed before Terraform managed it, so destroy keeps it.
		CheckDestroy: fake.checkTags(tagsR1, "pci", "other", "new"),
	})
}

func TestUnitResourceTagsResource_Validation(t *testing.T) {
	setupFakeEntitle(t, map[string][]string{})

	cases := map[string]struct {
		config string
		err    string
	}{
		"invalid uuid": {
			config: tagsConfig([]string{"not-a-uuid"}, []string{"pci"}),
			err:    `UUID Validate failed`,
		},
		"empty resource_ids": {
			config: tagsConfig([]string{}, []string{"pci"}),
			err:    `resource_ids set must contain at least 1 elements`,
		},
		"empty tags": {
			config: tagsConfig([]string{tagsR1}, []string{}),
			err:    `tags set must contain at least 1 elements`,
		},
		"empty tag value": {
			config: tagsConfig([]string{tagsR1}, []string{""}),
			err:    `string length must be at least 1`,
		},
		"missing tags": {
			config: tagsProviderConfig + fmt.Sprintf(`
resource "entitle_resource_tags" "test" {
  resource_ids = [%q]
}
`, tagsR1),
			err: `The argument "tags" is required`,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			resource.UnitTest(t, resource.TestCase{
				ProtoV6ProviderFactories: unitTestProviderFactories,
				Steps: []resource.TestStep{
					{
						Config:      tc.config,
						PlanOnly:    true,
						ExpectError: regexp.MustCompile(strings.ReplaceAll(tc.err, " ", `\s+`)),
					},
				},
			})
		})
	}
}
