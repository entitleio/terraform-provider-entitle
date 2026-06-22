package client

import (
	"fmt"
	"net/http"
	"strconv"
	"time"
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
)

// retryableStatuses lists the HTTP status codes that trigger a retry.
var retryableStatuses = map[int]bool{
	http.StatusTooManyRequests: true, // 429
	http.StatusBadGateway:      true, // 502
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
// Note: Do mutates req.Body in place on retries. It is not safe to call Do
// twice with the same *http.Request. This is fine in practice because the
// generated client constructs a fresh request on every call.
func (r *RetryDoer) Do(req *http.Request) (*http.Response, error) {
	backoff := r.baseBackoff

	for attempt := 0; attempt <= r.maxRetries; attempt++ {
		// Re-wind the request body for retries.
		if attempt > 0 {
			if req.GetBody == nil && req.Method != http.MethodGet {
				// Body already consumed and cannot be replayed — give up retrying.
				break
			}
			if req.GetBody != nil {
				body, err := req.GetBody()
				if err != nil {
					return nil, fmt.Errorf("entitle retry: resetting request body for attempt %d: %w", attempt, err)
				}
				req.Body = body
			}
		}

		resp, err := r.wrapped.Do(req)
		if err != nil {
			return nil, err
		}

		if !retryableStatuses[resp.StatusCode] {
			return resp, nil
		}

		if attempt == r.maxRetries {
			// Return the final error response as-is so callers can inspect the body.
			return resp, nil
		}

		sleep := r.retryAfterDuration(resp, backoff)
		_ = resp.Body.Close()

		select {
		case <-req.Context().Done():
			return nil, req.Context().Err()
		case <-time.After(sleep):
		}

		backoff = min(backoff*2, r.maxBackoff)
	}

	// Unreachable, but satisfies the compiler.
	return r.wrapped.Do(req)
}

// retryAfterDuration returns the duration to wait before the next attempt.
// It prefers the server-supplied Retry-After value; if absent or unparseable
// it falls back to the caller-supplied exponential backoff value.
func (r *RetryDoer) retryAfterDuration(resp *http.Response, backoff time.Duration) time.Duration {
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
	}

	return backoff
}
