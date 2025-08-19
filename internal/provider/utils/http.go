package utils

import (
	"fmt"
	"net/http"
	"strings"
)

type httpErrorOptions struct {
	ignoreNotFound bool
}

type HTTPErrorOption func(*httpErrorOptions)

func WithIgnoreNotFound() HTTPErrorOption {
	return func(opt *httpErrorOptions) {
		opt.ignoreNotFound = true
	}
}

func HTTPResponseToError(statusCode int, body []byte, opts ...HTTPErrorOption) error {
	// Apply options
	options := &httpErrorOptions{}
	for _, opt := range opts {
		opt(options)
	}

	switch statusCode {
	case http.StatusOK, http.StatusCreated, http.StatusAccepted:
		return nil
	case http.StatusUnauthorized:
		return errUnauthorizedToken
	default:
		errBody, _ := GetErrorBody(body)
		if strings.Contains(errBody.GetMessage(), "is not a valid uuid") {
			return errUnauthorizedToken
		}

		if options.ignoreNotFound && errBody.ID == "resource.notFound" {
			return nil
		}

		return fmt.Errorf("request failed, status code: %d, err: %s", statusCode, errBody.GetMessage())
	}
}
