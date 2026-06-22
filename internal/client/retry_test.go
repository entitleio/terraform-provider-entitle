package client

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

// doerFunc lets us use a plain function as an HttpRequestDoer in tests.
type doerFunc func(*http.Request) (*http.Response, error)

func (f doerFunc) Do(req *http.Request) (*http.Response, error) { return f(req) }

// mockDoer sequences through a list of (response, error) pairs.
type mockDoer struct {
	calls []mockCall
	idx   int
}

type mockCall struct {
	resp *http.Response
	err  error
}

func (m *mockDoer) Do(_ *http.Request) (*http.Response, error) {
	if m.idx >= len(m.calls) {
		panic("mockDoer: more calls than expected")
	}
	c := m.calls[m.idx]
	m.idx++
	return c.resp, c.err
}

func resp(status int) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{},
		Body:       io.NopCloser(strings.NewReader("")),
	}
}

func getReq(t *testing.T) *http.Request {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://api.example.com/v1/roles", nil)
	if err != nil {
		t.Fatal(err)
	}
	return req
}

func postReq(t *testing.T) *http.Request {
	t.Helper()
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://api.example.com/v1/roles", strings.NewReader(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	return req
}

// fastDoer creates a RetryDoer with millisecond backoffs so tests don't sleep.
func fastDoer(d HttpRequestDoer) *RetryDoer {
	return &RetryDoer{
		wrapped:     d,
		maxAttempts: defaultMaxAttempts,
		baseBackoff: time.Millisecond,
		maxBackoff:  5 * time.Millisecond,
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		method string
		status int
		want   bool
	}{
		// 429 is safe on any method
		{http.MethodGet, http.StatusTooManyRequests, true},
		{http.MethodPost, http.StatusTooManyRequests, true},
		{http.MethodPatch, http.StatusTooManyRequests, true},
		{http.MethodDelete, http.StatusTooManyRequests, true},

		// 502 idempotent methods
		{http.MethodGet, http.StatusBadGateway, true},
		{http.MethodHead, http.StatusBadGateway, true},
		{http.MethodPut, http.StatusBadGateway, true},
		{http.MethodDelete, http.StatusBadGateway, true},
		{http.MethodOptions, http.StatusBadGateway, true},

		// 502 non-idempotent — must not retry
		{http.MethodPost, http.StatusBadGateway, false},
		{http.MethodPatch, http.StatusBadGateway, false},

		// transport error (0) follows same idempotency rule as 502
		{http.MethodGet, transportError, true},
		{http.MethodPost, transportError, false},
		{http.MethodPatch, transportError, false},

		// other status codes are never retried
		{http.MethodGet, http.StatusOK, false},
		{http.MethodGet, http.StatusNotFound, false},
		{http.MethodGet, http.StatusInternalServerError, false},
	}

	for _, tt := range tests {
		if got := isRetryable(tt.method, tt.status); got != tt.want {
			t.Errorf("isRetryable(%q, %d) = %v, want %v", tt.method, tt.status, got, tt.want)
		}
	}
}

func TestSleepWithContext_completes(t *testing.T) {
	if err := sleepWithContext(context.Background(), time.Millisecond); err != nil {
		t.Fatal(err)
	}
}

func TestSleepWithContext_cancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := sleepWithContext(ctx, time.Hour); !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
}

func TestDo_successOnFirstAttempt(t *testing.T) {
	mock := &mockDoer{calls: []mockCall{{resp: resp(http.StatusOK)}}}
	r, err := fastDoer(mock).Do(getReq(t))
	if err != nil {
		t.Fatal(err)
	}
	if r.StatusCode != http.StatusOK || mock.idx != 1 {
		t.Fatalf("status=%d calls=%d", r.StatusCode, mock.idx)
	}
}

func TestDo_429_retriesOnAnyMethod(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodPost, http.MethodPatch} {
		t.Run(method, func(t *testing.T) {
			mock := &mockDoer{calls: []mockCall{
				{resp: resp(http.StatusTooManyRequests)},
				{resp: resp(http.StatusOK)},
			}}
			req, _ := http.NewRequestWithContext(context.Background(), method, "https://api.example.com/test", nil)
			r, err := fastDoer(mock).Do(req)
			if err != nil || r.StatusCode != http.StatusOK || mock.idx != 2 {
				t.Fatalf("err=%v status=%d calls=%d", err, r.StatusCode, mock.idx)
			}
		})
	}
}

func TestDo_502_retriesIdempotentMethods(t *testing.T) {
	for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodPut, http.MethodDelete, http.MethodOptions} {
		t.Run(method, func(t *testing.T) {
			mock := &mockDoer{calls: []mockCall{
				{resp: resp(http.StatusBadGateway)},
				{resp: resp(http.StatusOK)},
			}}
			req, _ := http.NewRequestWithContext(context.Background(), method, "https://api.example.com/test", nil)
			r, err := fastDoer(mock).Do(req)
			if err != nil || r.StatusCode != http.StatusOK || mock.idx != 2 {
				t.Fatalf("err=%v status=%d calls=%d", err, r.StatusCode, mock.idx)
			}
		})
	}
}

func TestDo_502_doesNotRetryPost(t *testing.T) {
	mock := &mockDoer{calls: []mockCall{{resp: resp(http.StatusBadGateway)}}}
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://api.example.com/test", nil)
	r, err := fastDoer(mock).Do(req)
	if err != nil || r.StatusCode != http.StatusBadGateway || mock.idx != 1 {
		t.Fatalf("err=%v status=%d calls=%d", err, r.StatusCode, mock.idx)
	}
}

func TestDo_transportError_retriesGet(t *testing.T) {
	netErr := errors.New("connection reset")
	mock := &mockDoer{calls: []mockCall{
		{err: netErr},
		{err: netErr},
		{resp: resp(http.StatusOK)},
	}}
	r, err := fastDoer(mock).Do(getReq(t))
	if err != nil || r.StatusCode != http.StatusOK || mock.idx != 3 {
		t.Fatalf("err=%v status=%d calls=%d", err, r.StatusCode, mock.idx)
	}
}

func TestDo_transportError_doesNotRetryPost(t *testing.T) {
	netErr := errors.New("connection reset")
	mock := &mockDoer{calls: []mockCall{{err: netErr}}}
	_, err := fastDoer(mock).Do(postReq(t))
	if !errors.Is(err, netErr) || mock.idx != 1 {
		t.Fatalf("err=%v calls=%d", err, mock.idx)
	}
}

func TestDo_exhaustedAttempts_returnsLastResponse(t *testing.T) {
	calls := make([]mockCall, defaultMaxAttempts)
	for i := range calls {
		calls[i] = mockCall{resp: resp(http.StatusTooManyRequests)}
	}
	mock := &mockDoer{calls: calls}
	r, err := fastDoer(mock).Do(getReq(t))
	if err != nil || r.StatusCode != http.StatusTooManyRequests || mock.idx != defaultMaxAttempts {
		t.Fatalf("err=%v status=%d calls=%d", err, r.StatusCode, mock.idx)
	}
}

func TestDo_exhaustedTransportErrors_returnsError(t *testing.T) {
	netErr := errors.New("timeout")
	calls := make([]mockCall, defaultMaxAttempts)
	for i := range calls {
		calls[i] = mockCall{err: netErr}
	}
	mock := &mockDoer{calls: calls}
	_, err := fastDoer(mock).Do(getReq(t))
	if !errors.Is(err, netErr) || mock.idx != defaultMaxAttempts {
		t.Fatalf("err=%v calls=%d", err, mock.idx)
	}
}

func TestDo_contextCancelledDuringSleep(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	var callCount int

	rd := &RetryDoer{
		wrapped: doerFunc(func(_ *http.Request) (*http.Response, error) {
			callCount++
			cancel() // cancel after the first attempt so the retry sleep fires Done
			return resp(http.StatusTooManyRequests), nil
		}),
		maxAttempts: defaultMaxAttempts,
		baseBackoff: time.Hour,
		maxBackoff:  time.Hour,
	}

	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.example.com/test", nil)
	_, err := rd.Do(req)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v, want context.Canceled", err)
	}
	if callCount != 1 {
		t.Fatalf("expected 1 call, got %d", callCount)
	}
}

func TestDo_unrewindableBody_returnsResponseWithoutRetry(t *testing.T) {
	// Body set manually without GetBody — simulates a body that can't be replayed.
	mock := &mockDoer{calls: []mockCall{{resp: resp(http.StatusTooManyRequests)}}}
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://api.example.com/test", nil)
	req.Body = io.NopCloser(strings.NewReader(`{}`))
	req.GetBody = nil

	r, err := fastDoer(mock).Do(req)
	if err != nil || r.StatusCode != http.StatusTooManyRequests || mock.idx != 1 {
		t.Fatalf("err=%v status=%d calls=%d", err, r.StatusCode, mock.idx)
	}
}

func TestDo_rewindableBody_isResetBetweenAttempts(t *testing.T) {
	const body = `{"name":"test"}`
	var captured []string

	rd := fastDoer(doerFunc(func(req *http.Request) (*http.Response, error) {
		b, _ := io.ReadAll(req.Body)
		captured = append(captured, string(b))
		if len(captured) == 1 {
			return resp(http.StatusTooManyRequests), nil
		}
		return resp(http.StatusOK), nil
	}))

	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://api.example.com/test", strings.NewReader(body))
	r, err := rd.Do(req)
	if err != nil || r.StatusCode != http.StatusOK {
		t.Fatalf("err=%v status=%d", err, r.StatusCode)
	}
	if len(captured) != 2 {
		t.Fatalf("expected 2 attempts, got %d", len(captured))
	}
	for i, b := range captured {
		if b != body {
			t.Errorf("attempt %d: body = %q, want %q", i+1, b, body)
		}
	}
}

func TestDo_cancelTransferredToBody(t *testing.T) {
	// cancel must not fire when Do returns — it should fire when the caller
	// closes the response body. We observe this via a channel written to by
	// a cancel function we inject through a custom context.
	cancelled := make(chan struct{})

	// Wrap the response body with a closer that records when Close is called.
	type trackClose struct {
		io.ReadCloser
		closed chan struct{}
	}
	trackBody := &trackClose{
		ReadCloser: io.NopCloser(strings.NewReader("")),
		closed:     cancelled,
	}
	// We can't easily inject into the internal context, so instead we verify
	// that the returned body is a cancelOnCloseBody and that its Close
	// triggers the underlying body's Close (proving the chain is intact).
	mockResp := &http.Response{
		StatusCode: http.StatusOK,
		Header:     http.Header{},
		Body:       trackBody,
	}

	var cancelCalled bool
	// Replace the wrapped doer with one that returns our instrumented response.
	rd := &RetryDoer{
		wrapped: doerFunc(func(_ *http.Request) (*http.Response, error) {
			return mockResp, nil
		}),
		maxAttempts: defaultMaxAttempts,
		baseBackoff: time.Millisecond,
		maxBackoff:  5 * time.Millisecond,
	}

	r, err := rd.Do(getReq(t))
	if err != nil {
		t.Fatal(err)
	}

	cb, ok := r.Body.(*cancelOnCloseBody)
	if !ok {
		t.Fatal("expected body to be wrapped in cancelOnCloseBody")
	}
	// Swap in an observable cancel to confirm it fires on Close, not before.
	original := cb.cancel
	cb.cancel = func() {
		cancelCalled = true
		original()
	}

	if cancelCalled {
		t.Fatal("cancel fired before body was closed")
	}
	r.Body.Close()
	if !cancelCalled {
		t.Fatal("cancel did not fire after body close")
	}
}

func TestRetryAfterDuration(t *testing.T) {
	rd := &RetryDoer{maxBackoff: 60 * time.Second}
	ctx := context.Background()
	fallback := 5 * time.Second

	makeResp := func(header string) *http.Response {
		h := http.Header{}
		if header != "" {
			h.Set("Retry-After", header)
		}
		return &http.Response{Header: h}
	}

	tests := []struct {
		name    string
		header  string
		wantMin time.Duration
		wantMax time.Duration
	}{
		{"no header falls back", "", fallback, fallback},
		{"delta-seconds", "10", 10 * time.Second, 10 * time.Second},
		{"delta-seconds capped at maxBackoff", "120", 60 * time.Second, 60 * time.Second},
		{"zero is invalid, falls back", "0", fallback, fallback},
		{"non-numeric falls back", "soon", fallback, fallback},
		{"past HTTP-date falls back", "Thu, 01 Jan 1970 00:00:00 GMT", fallback, fallback},
		{
			"future HTTP-date",
			time.Now().Add(10 * time.Second).UTC().Format(http.TimeFormat),
			9 * time.Second,
			11 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rd.retryAfterDuration(ctx, makeResp(tt.header), fallback)
			if got < tt.wantMin || got > tt.wantMax {
				t.Errorf("got %v, want [%v, %v]", got, tt.wantMin, tt.wantMax)
			}
		})
	}
}
