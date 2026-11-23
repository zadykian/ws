package kind

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/zadykian/ws/engine/core/resource"
)

// healthTries and healthWait bound an http state's wait: 30 tries, 2 seconds apart.
const (
	healthTries = 30
	healthWait  = 2 * time.Second
)

// healthClient asks the URL. Each try ends after its timeout, so that a service that takes the
// connection and never answers costs one try.
var healthClient = &http.Client{Timeout: 5 * time.Second}

// httpState is a URL that answers 200 once its service is up, such as a health check. systemctl
// returns once a service runs, which can be before it listens, or before it fails.
type httpState string

func (s httpState) resource(_ Env) (resource.Resource, error) {
	u, err := url.Parse(string(s))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, fmt.Errorf("%q is no http or https URL", string(s))
	}
	return health{url: string(s), tries: healthTries, wait: healthWait}, nil
}

// health waits for the URL to answer 200, tries times, wait apart.
type health struct {
	url   string
	tries int
	wait  time.Duration
}

// await is the wait for an answer.
type await struct{ h health }

func (c await) String() string { return "wait for an answer at " + c.h.url }

// Apply asks until the URL answers 200, or the tries run out.
func (c await) Apply(ctx context.Context) error {
	var err error
	for try := 1; try <= c.h.tries; try++ {
		if err = c.h.get(ctx); err == nil {
			return nil
		}
		if try == c.h.tries {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(c.h.wait):
		}
	}
	return fmt.Errorf("no answer at %s after %d tries: %w", c.h.url, c.h.tries, err)
}

// get asks the URL once, and fails unless it answers 200.
func (h health) get(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, h.url, nil)
	if err != nil {
		return err
	}
	resp, err := healthClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s answers %s", h.url, resp.Status)
	}
	return nil
}

// Plan plans the wait where the URL does not answer 200 now.
func (h health) Plan(ctx context.Context) ([]resource.Change, error) {
	if h.get(ctx) == nil {
		return nil, nil
	}
	return []resource.Change{await{h}}, nil
}
