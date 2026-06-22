package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	// DefaultRequestTimeout is the per-request HTTP timeout.
	DefaultRequestTimeout = 30 * time.Second

	defaultMaxAttempts = 5
	defaultBaseBackoff = 2 * time.Second
	defaultMaxBackoff  = 64 * time.Second

	// defaultRetryBudget caps the total wall-clock time for a Do call including
	// all attempts and sleeps. http.Client.Timeout only bounds each attempt.
	defaultRetryBudget = 3 * time.Minute

	// transportError is used as a status sentinel when no HTTP response was
	// received (network/transport failure).
	transportError = 0

	// maxBodyDrainBytes limits how much of a retryable response body is read
	// before closing, to avoid consuming unbounded memory on large error pages.
	maxBodyDrainBytes = 1 << 20
)

// isRetryable reports whether the request should be retried.
//
// 429 is safe on any method — the server rejected before processing.
// 502 and transport errors are only safe on idempotent methods, since the
// backend may have already processed the request.
func isRetryable(method string, status int) bool {
	switch status {
	case http.StatusTooManyRequests:
		return true
	case http.StatusBadGateway, transportError:
		switch method {
		case http.MethodGet, http.MethodHead, http.MethodPut, http.MethodDelete, http.MethodOptions:
			return true
		}
	}
	return false
}

// sleepWithContext waits for d, returning early with ctx.Err() if the context
// is canceled. Uses time.NewTimer to avoid leaking the timer on early return.
func sleepWithContext(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

// RetryDoer wraps an HttpRequestDoer with retry logic for 429 and 502
// responses, honouring the Retry-After header and falling back to exponential
// backoff. Transport errors are retried on idempotent methods.
type RetryDoer struct {
	wrapped     HttpRequestDoer
	maxAttempts int
	baseBackoff time.Duration
	maxBackoff  time.Duration
}

func NewRetryDoer(wrapped HttpRequestDoer) *RetryDoer {
	return &RetryDoer{
		wrapped:     wrapped,
		maxAttempts: defaultMaxAttempts,
		baseBackoff: defaultBaseBackoff,
		maxBackoff:  defaultMaxBackoff,
	}
}

// Do executes the request with retry logic. It calls req.WithContext internally,
// creating a shallow copy — req.Body is mutated on the copy, not the caller's
// original. Do is not safe to call twice with the same *http.Request; the
// generated client always builds a fresh one, so this is fine in practice.
func (r *RetryDoer) Do(req *http.Request) (*http.Response, error) {
	ctx, cancel := context.WithTimeout(req.Context(), defaultRetryBudget)
	defer cancel()
	req = req.WithContext(ctx)

	backoff := r.baseBackoff

	for attempt := 0; attempt < r.maxAttempts; attempt++ {
		if attempt > 0 && req.GetBody != nil {
			body, err := req.GetBody()
			if err != nil {
				return nil, fmt.Errorf("entitle retry: resetting request body for attempt %d: %w", attempt, err)
			}
			req.Body = body
		}

		resp, err := r.wrapped.Do(req)
		if err != nil {
			if attempt < r.maxAttempts-1 && isRetryable(req.Method, transportError) {
				tflog.Debug(ctx, "entitle retry: transport error, will retry",
					map[string]any{"attempt": attempt + 1, "method": req.Method, "error": err.Error(), "backoff": backoff.String()})
				if sleepErr := sleepWithContext(ctx, backoff); sleepErr != nil {
					return nil, sleepErr
				}
				backoff = min(backoff*2, r.maxBackoff)
				continue
			}
			return nil, err
		}

		if !isRetryable(req.Method, resp.StatusCode) {
			return resp, nil
		}

		if attempt == r.maxAttempts-1 {
			return resp, nil
		}

		// An unrewindable body means we can't safely retry — return the response
		// as-is with the body still open for the caller to read.
		if req.Body != nil && req.GetBody == nil {
			return resp, nil
		}

		sleep := r.retryAfterDuration(ctx, resp, backoff)
		tflog.Debug(ctx, "entitle retry: retryable response, will retry",
			map[string]any{"attempt": attempt + 1, "method": req.Method, "url": req.URL.String(), "status": resp.StatusCode, "sleep": sleep.String()})

		_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxBodyDrainBytes))
		_ = resp.Body.Close()

		if sleepErr := sleepWithContext(ctx, sleep); sleepErr != nil {
			return nil, sleepErr
		}

		backoff = min(backoff*2, r.maxBackoff)
	}

	return nil, fmt.Errorf("entitle retry: Do: fell through retry loop — this is a bug")
}

// retryAfterDuration parses the Retry-After header and returns the sleep
// duration, capped at maxBackoff. Falls back to backoff if the header is
// absent or unparseable.
func (r *RetryDoer) retryAfterDuration(ctx context.Context, resp *http.Response, backoff time.Duration) time.Duration {
	ra := resp.Header.Get("Retry-After")
	if ra == "" {
		return backoff
	}

	// delta-seconds form
	if secs, err := strconv.Atoi(ra); err == nil && secs > 0 {
		return min(time.Duration(secs)*time.Second, r.maxBackoff)
	}

	// HTTP-date form
	if t, err := http.ParseTime(ra); err == nil {
		if d := time.Until(t); d > 0 {
			return min(d, r.maxBackoff)
		}
		tflog.Debug(ctx, "entitle retry: Retry-After date is in the past, possible clock skew",
			map[string]any{"retry_after": ra, "backoff": backoff.String()})
	}

	return backoff
}
