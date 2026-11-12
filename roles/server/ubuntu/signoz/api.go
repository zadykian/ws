package signoz

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// API calls SigNoz's HTTP API. SigNoz serves every request as its root user, an admin, so the
// calls need no login. A nil Client is an http.Client with a timeout.
type API struct {
	URL    string
	Client *http.Client
}

func (a API) client() *http.Client {
	if a.Client != nil {
		return a.Client
	}
	return &http.Client{Timeout: 30 * time.Second}
}

// call sends the request and decodes the JSON answer into out, if given. Any status but want is an
// error that holds the start of the body.
func (a API) call(ctx context.Context, method, path string, body []byte, want int, out any) error {
	req, err := http.NewRequestWithContext(ctx, method, a.URL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := a.client().Do(req)
	if err != nil {
		return fmt.Errorf("%s %s: %w", method, path, err)
	}
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("%s %s: %w", method, path, err)
	}
	if resp.StatusCode != want {
		return fmt.Errorf("%s %s: status %d, want %d: %.200s", method, path, resp.StatusCode, want, data)
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("%s %s: %w", method, path, err)
	}
	return nil
}

func (a API) get(ctx context.Context, path string, out any) error {
	return a.call(ctx, http.MethodGet, path, nil, http.StatusOK, out)
}
