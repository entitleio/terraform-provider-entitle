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
	// DefaultRequestTimeout is the per-request HTTP timeout. Any individual API call
	// that does not complete within this window is cancelled and returns an error,
	// preventing data-source reads from hanging indefinitely.
	DefaultRequestTimeout = 30 * time.Second

	// defaultMaxRetries is the number of times a request is retried on 429 / 502.
	defaultMaxRetries = 5

	// defaultBaseBackoff is the initial back-off interval used when no Retry-After
	// header is present.
	defaultBaseBackoff = 2 * time.Second

	// defaultMaxBackoff caps the computed back-off so retries never wait longer
	// than this duration regardless of what the server advertises.
	defaultMaxBackoff = 64 * time.Second

	// defaultRetryBudget is the total wall-clock budget for a single Do call,
	// including all attempts and sleep intervals. http.Client.Timeout only bounds
	// each individual attempt; without this cap, a call honouring maxRetries=5
	// Retry-After values of maxBackoff=64s each could block for several minutes.
	// If the caller's context already has a shorter deadline, that takes precedence.
	defaultRetryBudget = 3 * time.Minute
)

// isRetryable reports whether a failed request should be retried given the
// HTTP method and status code (or zero for a transport error).
//
// 429 (Too Many Requests) is safe on any method: the server rejected the
// request before processing it, so no side effect has occurred.
//
// 502 (Bad Gateway) and transport errors (status == 0) are only safe on
// idempotent methods. The gateway may have forwarded the request before
// failing, so retrying POST or PATCH risks duplicate resource creation or
// inconsistent state — unacceptable in a Terraform provider.
func isRetryable(method string, status int) bool {
	switch status {
	case http.StatusTooManyRequests: // 429 — rejected before processing, safe on any method
		return true
	case http.StatusBadGateway, 0: // 502 or transport error — idempotent methods only
		switch method {
		case http.MethodGet, http.MethodHead, http.MethodPut, http.MethodDelete, http.MethodOptions:
			return true
		}
	}
	return false
}

// RetryDoer wraps any HttpRequestDoer and adds:
//   - Retry logic for 429 (Too Many Requests) and 502 (Bad Gateway) responses.
//   - Respect for the Retry-After response header (both delta-seconds and
//     HTTP-date forms are supported).
//   - Exponential back-off when no Retry-After header is present.
type RetryDoer struct {
	wrapped     HttpRequestDoer
	maxRetries  int
	baseBackoff time.Duration
	maxBackoff  time.Duration
}

// NewRetryDoer creates a RetryDoer that delegates to wrapped.
func NewRetryDoer(wrapped HttpRequestDoer) *RetryDoer {
	return &RetryDoer{
		wrapped:     wrapped,
		maxRetries:  defaultMaxRetries,
		baseBackoff: defaultBaseBackoff,
		maxBackoff:  defaultMaxBackoff,
	}
}

// Do executes the request, retrying on 429 / 502 up to maxRetries times.
// The request body is re-created for each retry attempt via req.GetBody when
// available (the stdlib sets this automatically for bytes.Reader / strings.Reader
// bodies, which is what the generated client uses). Non-GET requests without a
// rewindable body are not retried.
//
// Note: Do applies an internal context deadline and calls req.WithContext,
// which creates a shallow copy of the request. req.Body is mutated on the
// copy, not the caller's original. It is not safe to call Do twice with the
// same *http.Request. This is fine in practice because the generated client
// constructs a fresh request on every call.
func (r *RetryDoer) Do(req *http.Request) (*http.Response, error) {
	// Apply an overall budget for the entire retry loop (attempts + sleep).
	// If the caller's context already has a shorter deadline, context.WithTimeout
	// is a no-op in practice — the shorter deadline wins.
	ctx, cancel := context.WithTimeout(req.Context(), defaultRetryBudget)
	defer cancel()
	req = req.WithContext(ctx)

	backoff := r.baseBackoff

	for attempt := 0; attempt <= r.maxRetries; attempt++ {
		// Re-wind the request body for retries.
		if attempt > 0 && req.GetBody != nil {
			body, err := req.GetBody()
			if err != nil {
				return nil, fmt.Errorf("entitle retry: resetting request body for attempt %d: %w", attempt, err)
			}
			req.Body = body
		}

		resp, err := r.wrapped.Do(req)
		if err != nil {
			// Retry transport errors (timeout, connection reset) only for idempotent
			// methods — non-idempotent requests (POST, PATCH) must not be re-sent
			// blindly as they may have already produced side effects.
			if attempt < r.maxRetries && isRetryable(req.Method, 0) {
				select {
				case <-req.Context().Done():
					return nil, req.Context().Err()
				case <-time.After(backoff):
				}
				backoff = min(backoff*2, r.maxBackoff)
				continue
			}
			return nil, err
		}

		// Not a retryable status/method combination — return immediately.
		if !isRetryable(req.Method, resp.StatusCode) {
			return resp, nil
		}

		// Exhausted retries — return the last response (body still open).
		if attempt == r.maxRetries {
			return resp, nil
		}

		// Can't replay the body — return this response (body still open) rather
		// than closing it and firing a broken request with an empty body.
		if req.Body != nil && req.GetBody == nil {
			return resp, nil
		}

		sleep := r.retryAfterDuration(req.Context(), resp, backoff)
		_, _ = io.Copy(io.Discard, resp.Body)
		_ = resp.Body.Close()

		select {
		case <-req.Context().Done():
			return nil, req.Context().Err()
		case <-time.After(sleep):
		}

		backoff = min(backoff*2, r.maxBackoff)
	}

	// Unreachable: every loop iteration returns or continues.
	panic("entitle retry: Do: fell through retry loop — this is a bug")
}

// retryAfterDuration returns the duration to wait before the next attempt.
// It prefers the server-supplied Retry-After value; if absent or unparseable
// it falls back to the caller-supplied exponential backoff value.
func (r *RetryDoer) retryAfterDuration(ctx context.Context, resp *http.Response, backoff time.Duration) time.Duration {
	ra := resp.Header.Get("Retry-After")
	if ra == "" {
		return backoff
	}

	// delta-seconds form: "Retry-After: 30"
	if secs, err := strconv.Atoi(ra); err == nil && secs > 0 {
		d := time.Duration(secs) * time.Second
		return min(d, r.maxBackoff)
	}

	// HTTP-date form: "Retry-After: Wed, 21 Oct 2015 07:28:00 GMT"
	if t, err := http.ParseTime(ra); err == nil {
		if d := time.Until(t); d > 0 {
			return min(d, r.maxBackoff)
		}
		// The Retry-After timestamp is in the past. This can indicate clock skew
		// between client and server. Fall back to exponential backoff.
		tflog.Debug(ctx, "entitle retry: Retry-After HTTP-date is in the past, possible clock skew; using backoff",
			map[string]any{"retry_after": ra, "backoff": backoff.String()})
	}

	return backoff
}
