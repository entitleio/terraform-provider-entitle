package bulk

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// APIError is a non-2xx response from a bulk endpoint.
type APIError struct {
	StatusCode int
	ErrorID    string
	Message    string
	// IDs lists the offending entity IDs reported by the API in params.ids
	// (e.g. resources that do not exist or cannot be edited).
	IDs []string
}

func (e *APIError) Error() string {
	var b strings.Builder
	fmt.Fprintf(&b, "bulk request failed, status code: %d", e.StatusCode)
	if e.ErrorID != "" {
		fmt.Fprintf(&b, ", error id: %s", e.ErrorID)
	}
	if e.Message != "" {
		fmt.Fprintf(&b, ", message: %s", e.Message)
	}
	if len(e.IDs) > 0 {
		fmt.Fprintf(&b, ", offending ids: %s", strings.Join(e.IDs, ", "))
	}
	return b.String()
}

// IsConflict reports whether another bulk operation is already queued or running for the tenant.
func (e *APIError) IsConflict() bool { return e.StatusCode == http.StatusConflict }

// IsNotFound reports whether some referenced entity could not be found.
func (e *APIError) IsNotFound() bool { return e.StatusCode == http.StatusNotFound }

type errorBody struct {
	ErrorID string `json:"errorId"`
	Message string `json:"message"`
	Params  struct {
		IDs []string `json:"ids"`
	} `json:"params"`
}

// NewAPIError builds an APIError from a raw response.
func NewAPIError(statusCode int, body []byte) *APIError {
	apiErr := &APIError{StatusCode: statusCode}

	var parsed errorBody
	if err := json.Unmarshal(body, &parsed); err == nil {
		apiErr.ErrorID = parsed.ErrorID
		apiErr.Message = parsed.Message
		apiErr.IDs = parsed.Params.IDs
	} else if len(body) > 0 {
		apiErr.Message = truncate(string(body), 512)
	}

	return apiErr
}

// AsAPIError unwraps err into an *APIError if possible.
func AsAPIError(err error) (*APIError, bool) {
	var apiErr *APIError
	ok := errors.As(err, &apiErr)
	return apiErr, ok
}

// JobFailedError is returned when a bulk job finishes with status "failed".
type JobFailedError struct {
	JobID     string
	Message   string
	Processed int
	Total     int
}

func (e *JobFailedError) Error() string {
	msg := e.Message
	if msg == "" {
		msg = "no error message returned"
	}
	return fmt.Sprintf("bulk operation %s failed after processing %d/%d items: %s", e.JobID, e.Processed, e.Total, msg)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
