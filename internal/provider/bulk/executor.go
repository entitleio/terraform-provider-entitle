// Package bulk contains the logic shared by all bulk-update resources:
// chunking IDs, submitting asynchronous bulk jobs with bounded parallelism,
// retrying while another bulk operation holds the tenant lock (409), and
// polling jobs until they complete.
package bulk

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	openapi_types "github.com/oapi-codegen/runtime/types"

	"github.com/entitleio/terraform-provider-entitle/internal/client"
	"github.com/entitleio/terraform-provider-entitle/internal/provider/utils"
)

const (
	// DefaultTimeout bounds a whole Run when the caller's context has no deadline.
	DefaultTimeout = 60 * time.Minute

	defaultPollInterval       = 1 * time.Second
	defaultMaxPollInterval    = 15 * time.Second
	defaultConflictBackoff    = 2 * time.Second
	defaultMaxConflictBackoff = 30 * time.Second
)

// SubmitFunc sends one bulk request for the given chunk of IDs.
type SubmitFunc func(ctx context.Context, ids []string) (*client.BulkSubmitResponse, error)

// Operation is a single logical bulk change applied to a set of IDs.
type Operation struct {
	// Name is used in logs and error messages, e.g. "add resource tags".
	Name string
	// IDs to apply the change to. Split into chunks of BulkConfig.ChunkSize.
	IDs []string
	// Submit sends the request for one chunk.
	Submit SubmitFunc
	// SkipMissingIDs drops IDs the API reports as not found (404, params.ids)
	// and resubmits the rest. Use for removals/cleanup, where an entity that
	// no longer exists needs no further change.
	SkipMissingIDs bool
}

// Result summarises a completed Run.
type Result struct {
	// Queued is the number of items accepted by the API across all chunks.
	Queued int
	// Skipped lists IDs dropped because SkipMissingIDs was set and the API reported them missing.
	Skipped []string
}

// Executor runs bulk operations according to the provider-wide BulkConfig.
type Executor struct {
	client *client.ClientWithResponses
	config utils.BulkConfig

	PollInterval       time.Duration
	MaxPollInterval    time.Duration
	ConflictBackoff    time.Duration
	MaxConflictBackoff time.Duration
}

// NewExecutor creates an Executor with default timings.
func NewExecutor(c *client.ClientWithResponses, cfg utils.BulkConfig) *Executor {
	return &Executor{
		client:             c,
		config:             cfg,
		PollInterval:       defaultPollInterval,
		MaxPollInterval:    defaultMaxPollInterval,
		ConflictBackoff:    defaultConflictBackoff,
		MaxConflictBackoff: defaultMaxConflictBackoff,
	}
}

// Config returns the bulk limits used by this executor.
func (e *Executor) Config() utils.BulkConfig { return e.config }

// RunAll runs operations sequentially, skipping ones with no IDs. It stops at the first error.
func (e *Executor) RunAll(ctx context.Context, ops ...Operation) (Result, error) {
	var total Result
	for _, op := range ops {
		res, err := e.Run(ctx, op)
		total.Queued += res.Queued
		total.Skipped = append(total.Skipped, res.Skipped...)
		if err != nil {
			return total, err
		}
	}
	return total, nil
}

// Run applies a single operation: it chunks the IDs, submits the chunks with at
// most BulkConfig.Parallelism in flight, and waits for every job to finish.
func (e *Executor) Run(ctx context.Context, op Operation) (Result, error) {
	ids := Unique(op.IDs)
	if len(ids) == 0 {
		return Result{}, nil
	}
	if op.Submit == nil {
		return Result{}, fmt.Errorf("%s: no submit function configured", op.Name)
	}

	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, DefaultTimeout)
		defer cancel()
	}

	chunks := Chunk(ids, int(e.config.ChunkSize))
	workers := min(int(e.config.Parallelism), len(chunks))

	tflog.Info(ctx, "Starting bulk operation", map[string]any{
		"operation":   op.Name,
		"items":       len(ids),
		"chunks":      len(chunks),
		"parallelism": workers,
	})

	var (
		mu     sync.Mutex
		result Result
	)

	err := ForEach(ctx, workers, chunks, func(ctx context.Context, chunk []string) error {
		res, err := e.runChunk(ctx, op, chunk)
		mu.Lock()
		result.Queued += res.Queued
		result.Skipped = append(result.Skipped, res.Skipped...)
		mu.Unlock()
		return err
	})
	if err != nil {
		return result, fmt.Errorf("%s: %w", op.Name, err)
	}

	tflog.Info(ctx, "Finished bulk operation", map[string]any{
		"operation": op.Name,
		"queued":    result.Queued,
		"skipped":   len(result.Skipped),
	})

	return result, nil
}

func (e *Executor) runChunk(ctx context.Context, op Operation, ids []string) (Result, error) {
	var res Result

	for len(ids) > 0 {
		job, err := e.submit(ctx, op, ids)
		if err != nil {
			apiErr, ok := AsAPIError(err)
			if !ok || !op.SkipMissingIDs || !apiErr.IsNotFound() || len(apiErr.IDs) == 0 {
				return res, err
			}

			missing := Intersection(ids, apiErr.IDs)
			if len(missing) == 0 {
				// The 404 refers to something other than the target IDs (e.g. owner/workflow).
				return res, err
			}

			tflog.Warn(ctx, "Skipping IDs reported as not found", map[string]any{
				"operation": op.Name,
				"ids":       missing,
			})
			res.Skipped = append(res.Skipped, missing...)
			ids = Difference(ids, missing)
			continue
		}

		res.Queued += job.Count
		return res, e.wait(ctx, op, job.JobId)
	}

	return res, nil
}

// submit sends a chunk, retrying while another bulk operation is running (409).
func (e *Executor) submit(ctx context.Context, op Operation, ids []string) (client.BulkJobResult, error) {
	backoff := e.ConflictBackoff

	for {
		resp, err := op.Submit(ctx, ids)
		if err != nil {
			return client.BulkJobResult{}, err
		}

		switch code := resp.StatusCode(); code {
		case http.StatusOK, http.StatusAccepted:
			if resp.JSON202 == nil {
				return client.BulkJobResult{}, fmt.Errorf("bulk request returned status %d without a job", code)
			}
			return resp.JSON202.Result, nil

		case http.StatusConflict:
			tflog.Debug(ctx, "Another bulk operation is in progress, retrying", map[string]any{
				"operation": op.Name,
				"backoff":   backoff.String(),
			})
			if err := sleep(ctx, backoff); err != nil {
				return client.BulkJobResult{}, fmt.Errorf("waiting for another bulk operation to finish: %w", err)
			}
			backoff = min(backoff*2, e.MaxConflictBackoff)

		default:
			return client.BulkJobResult{}, NewAPIError(code, resp.Body)
		}
	}
}

// wait polls a job until it reaches a terminal status.
func (e *Executor) wait(ctx context.Context, op Operation, jobID openapi_types.UUID) error {
	interval := e.PollInterval

	for {
		if err := sleep(ctx, interval); err != nil {
			return fmt.Errorf("waiting for bulk operation %s: %w", jobID, err)
		}

		resp, err := e.client.BulkOperationsShowWithResponse(ctx, jobID)
		if err != nil {
			return fmt.Errorf("polling bulk operation %s: %w", jobID, err)
		}
		if resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
			return fmt.Errorf("polling bulk operation %s: %w", jobID, NewAPIError(resp.StatusCode(), resp.Body))
		}

		status := resp.JSON200.Result
		tflog.Debug(ctx, "Bulk operation progress", map[string]any{
			"operation": op.Name,
			"job_id":    jobID.String(),
			"status":    string(status.Status),
			"processed": status.Processed,
			"total":     status.Total,
		})

		switch status.Status {
		case client.BulkOperationStatusCompleted:
			return nil
		case client.BulkOperationStatusFailed:
			return &JobFailedError{
				JobID:     jobID.String(),
				Message:   utils.StringValue(status.Error),
				Processed: status.Processed,
				Total:     status.Total,
			}
		}

		interval = min(interval*3/2, e.MaxPollInterval)
	}
}

func sleep(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return fmt.Errorf("timed out: %w", ctx.Err())
		}
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
