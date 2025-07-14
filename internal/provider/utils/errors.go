package utils

import (
	"encoding/json"
	"errors"
	"fmt"
)

var errUnauthorizedToken = errors.New("unauthorized token: update the entitle token and retry please")

type ErrorBody struct {
	ID      string `json:"errorId"`
	Message string `json:"message"`
}

func GetErrorBody(data []byte) (ErrorBody, error) {
	var result ErrorBody
	err := json.Unmarshal(data, &result)
	if err != nil {
		return result, err
	}

	if len(result.Message) == 0 {
		return result, errors.New("missing message in error body")
	}

	return result, nil
}

func (e ErrorBody) GetMessage() string {
	if e.Message == "" {
		return ""
	}

	return fmt.Sprintf(", getting error from entitle API error message: %s", e.Message)
}
