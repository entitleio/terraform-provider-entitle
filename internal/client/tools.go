package client

import (
	"context"
	"net/http"
)

func SetBearerToken(bearer string) RequestEditorFn {
	return func(ctx context.Context, req *http.Request) error {
		req.Header.Set("Authorization", "Bearer "+bearer)
		return nil
	}
}
