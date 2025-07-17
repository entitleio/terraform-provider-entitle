package utils

import (
	"fmt"
	"net/http"
	"strings"
)

func HTTPResponseToError(statusCode int, body []byte) error {
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

		return fmt.Errorf("request failed, status code: %d, err: %s", statusCode, errBody.GetMessage())
	}
}
