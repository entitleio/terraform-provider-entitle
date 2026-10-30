package bulk

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/entitleio/terraform-provider-entitle/internal/client"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/utils"
)

// fakeAPI emulates the bulk resources endpoint and the bulk operations endpoint.
type fakeAPI struct {
	mu sync.Mutex

	conflictsLeft int             // number of 409 responses to return before accepting
	missing       map[string]bool // IDs reported as not found
	jobStatus     client.BulkOperationStatus
	jobError      string
	submitted     [][]string // IDs of accepted submissions
	bodies        []client.BulkUpdateResourcesRequest
	polls         map[string]int // jobId -> number of polls
}

func (f *fakeAPI) handler(t *testing.T) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.mu.Lock()
		defer f.mu.Unlock()

		switch {
		case r.Method == http.MethodPatch && r.URL.Path == "/public/v1/resources/bulk":
			if r.Header.Get("Authorization") != "Bearer token" {
				t.Errorf("missing auth header")
			}
			var body client.BulkUpdateResourcesRequest
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decode: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			if f.conflictsLeft > 0 {
				f.conflictsLeft--
				w.WriteHeader(http.StatusConflict)
				_, _ = w.Write([]byte(`{"errorId":"bulk.conflict","message":"busy"}`))
				return
			}

			var missing []string
			for _, id := range body.Ids {
				if f.missing[id] {
					missing = append(missing, id)
				}
			}
			if len(missing) > 0 {
				w.WriteHeader(http.StatusNotFound)
				_ = json.NewEncoder(w).Encode(map[string]any{
					"errorId": "resource.notFound",
					"message": "not found",
					"params":  map[string]any{"ids": missing},
				})
				return
			}

			f.submitted = append(f.submitted, body.Ids)
			f.bodies = append(f.bodies, body)
			w.WriteHeader(http.StatusAccepted)
			_ = json.NewEncoder(w).Encode(client.BulkJobResponseSchema{
				Result: client.BulkJobResult{JobId: uuid.New(), Count: len(body.Ids)},
			})

		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/public/v1/bulkOperations/"):
			jobID := strings.TrimPrefix(r.URL.Path, "/public/v1/bulkOperations/")
			f.polls[jobID]++

			status := client.BulkOperationStatusRunning
			if f.polls[jobID] > 1 {
				status = f.jobStatus
			}
			res := client.BulkOperationResult{Status: status, Processed: 1, Total: 1}
			if status == client.BulkOperationStatusFailed {
				res.Error = &f.jobError
			}
			_ = json.NewEncoder(w).Encode(client.BulkOperationResponseSchema{Result: res})

		default:
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusTeapot)
		}
	})
}

func newTestExecutor(t *testing.T, api *fakeAPI, cfg utils.BulkConfig) *Executor {
	t.Helper()
	if api.polls == nil {
		api.polls = map[string]int{}
	}
	if api.jobStatus == "" {
		api.jobStatus = client.BulkOperationStatusCompleted
	}

	srv := httptest.NewServer(api.handler(t))
	t.Cleanup(srv.Close)

	c, err := client.NewClientWithResponses(srv.URL, client.WithRequestEditorFn(client.SetBearerToken("token")))
	if err != nil {
		t.Fatal(err)
	}

	e := NewExecutor(c, cfg)
	e.PollInterval = time.Millisecond
	e.MaxPollInterval = 2 * time.Millisecond
	e.ConflictBackoff = time.Millisecond
	e.MaxConflictBackoff = 2 * time.Millisecond
	return e
}

func tagsOp(e *Executor, ids []string, skipMissing bool) Operation {
	add := []string{"pci"}
	return Operation{
		Name:           "add tags",
		IDs:            ids,
		SkipMissingIDs: skipMissing,
		Submit:         ResourcesUpdate(e.client, client.BulkResourceUpdate{Tags: &client.BulkTagsUpdate{Add: &add}}),
	}
}

func TestExecutorChunksAndWaits(t *testing.T) {
	api := &fakeAPI{}
	e := newTestExecutor(t, api, utils.BulkConfig{ChunkSize: 2, Parallelism: 2})

	res, err := e.Run(context.Background(), tagsOp(e, []string{"a", "b", "c", "d", "e", "a"}, false))
	if err != nil {
		t.Fatal(err)
	}
	if res.Queued != 5 {
		t.Fatalf("queued = %d, want 5", res.Queued)
	}
	if len(api.submitted) != 3 {
		t.Fatalf("submissions = %d, want 3", len(api.submitted))
	}

	var all []string
	for _, s := range api.submitted {
		if len(s) > 2 {
			t.Fatalf("chunk larger than chunk_size: %v", s)
		}
		all = append(all, s...)
	}
	sort.Strings(all)
	if strings.Join(all, ",") != "a,b,c,d,e" {
		t.Fatalf("submitted ids = %v", all)
	}
	if got := *api.bodies[0].Update.Tags.Add; len(got) != 1 || got[0] != "pci" {
		t.Fatalf("unexpected tags body: %v", got)
	}
}

func TestExecutorRetriesConflict(t *testing.T) {
	api := &fakeAPI{conflictsLeft: 3}
	e := newTestExecutor(t, api, utils.DefaultBulkConfig())

	if _, err := e.Run(context.Background(), tagsOp(e, []string{"a"}, false)); err != nil {
		t.Fatal(err)
	}
	if api.conflictsLeft != 0 || len(api.submitted) != 1 {
		t.Fatalf("conflicts left = %d, submissions = %d", api.conflictsLeft, len(api.submitted))
	}
}

func TestExecutorNotFound(t *testing.T) {
	api := &fakeAPI{missing: map[string]bool{"b": true}}
	e := newTestExecutor(t, api, utils.DefaultBulkConfig())

	_, err := e.Run(context.Background(), tagsOp(e, []string{"a", "b"}, false))
	apiErr, ok := AsAPIError(err)
	if !ok || !apiErr.IsNotFound() || len(apiErr.IDs) != 1 || apiErr.IDs[0] != "b" {
		t.Fatalf("expected not found error for b, got %v", err)
	}
	if len(api.submitted) != 0 {
		t.Fatal("nothing should be applied")
	}
}

func TestExecutorSkipMissing(t *testing.T) {
	api := &fakeAPI{missing: map[string]bool{"b": true}}
	e := newTestExecutor(t, api, utils.DefaultBulkConfig())

	res, err := e.Run(context.Background(), tagsOp(e, []string{"a", "b", "c"}, true))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Skipped) != 1 || res.Skipped[0] != "b" {
		t.Fatalf("skipped = %v", res.Skipped)
	}
	if len(api.submitted) != 1 || strings.Join(api.submitted[0], ",") != "a,c" {
		t.Fatalf("submitted = %v", api.submitted)
	}
}

func TestExecutorJobFailed(t *testing.T) {
	api := &fakeAPI{jobStatus: client.BulkOperationStatusFailed, jobError: "boom"}
	e := newTestExecutor(t, api, utils.DefaultBulkConfig())

	_, err := e.Run(context.Background(), tagsOp(e, []string{"a"}, false))
	var jobErr *JobFailedError
	if !errors.As(err, &jobErr) || jobErr.Message != "boom" {
		t.Fatalf("expected job failed error, got %v", err)
	}
}

func TestExecutorEmptyIsNoop(t *testing.T) {
	api := &fakeAPI{}
	e := newTestExecutor(t, api, utils.DefaultBulkConfig())

	if _, err := e.RunAll(context.Background(), tagsOp(e, nil, false), tagsOp(e, []string{}, true)); err != nil {
		t.Fatal(err)
	}
	if len(api.submitted) != 0 {
		t.Fatal("expected no requests")
	}
}

func TestFilterIDs(t *testing.T) {
	kept, err := FilterIDs(context.Background(), 3, []string{"a", "b", "c", "d"}, func(_ context.Context, id string) (bool, error) {
		return id != "b", nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(kept, ",") != "a,c,d" {
		t.Fatalf("kept = %v", kept)
	}

	_, err = FilterIDs(context.Background(), 2, []string{"a", "b"}, func(_ context.Context, id string) (bool, error) {
		return false, errors.New("fail")
	})
	if err == nil {
		t.Fatal("expected error")
	}
}
