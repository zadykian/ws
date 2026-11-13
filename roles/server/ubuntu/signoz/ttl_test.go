package signoz

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/zadykian/ws/roles/server/ubuntu/signoz/internal/api"
)

// fakeTTL is SigNoz's v1 retention API for traces and metrics. A POST sets the hours, and the
// status stays "in_progress" for the first reads after it.
type fakeTTL struct {
	mu      sync.Mutex
	hours   map[string]int
	status  map[string]string
	pending map[string]int
	posts   []string
}

func (f *fakeTTL) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	signal := r.URL.Query().Get("type")
	if r.Method == http.MethodPost {
		f.posts = append(f.posts, r.URL.RawQuery)
		var h int
		_, _ = fmt.Sscanf(r.URL.Query().Get("duration"), "%dh", &h)
		f.hours[signal], f.status[signal], f.pending[signal] = h, "in_progress", 2
		return
	}
	if f.pending[signal]--; f.pending[signal] == 0 {
		f.status[signal] = "success"
	}
	_ = json.NewEncoder(w).Encode(map[string]any{signal + "_ttl_duration_hrs": f.hours[signal], "status": f.status[signal]})
}

func newRetention(t *testing.T, f *fakeTTL) Retention {
	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)
	w := api.Wait{Tries: 5, Delay: time.Second, Sleep: func(context.Context, time.Duration) error { return nil }}
	return Retention{API: api.Client{URL: srv.URL}, Days: 90, Altered: w}
}

func TestSetSignalsSetsThenWaitsForTheTables(t *testing.T) {
	f := &fakeTTL{hours: map[string]int{"traces": 360, "metrics": 720}, status: map[string]string{},
		pending: map[string]int{}}
	r := newRetention(t, f)
	c := setSignals{r, []ttl{{signal: "traces"}, {signal: "metrics"}}}
	if err := c.Apply(t.Context()); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(f.posts, " "); got != "type=traces&duration=2160h type=metrics&duration=2160h" {
		t.Errorf("posts = %q", got)
	}
	if f.status["traces"] != "success" || f.status["metrics"] != "success" {
		t.Errorf("status = %v, want both done", f.status)
	}
}

func TestSignalsDueWhereTheHoursDifferOrTheLastChangeFailed(t *testing.T) {
	for _, tc := range []struct {
		t    ttl
		want bool
	}{
		{ttl{hours: 2160, status: "success"}, false},
		{ttl{hours: 360, status: "success"}, true},
		{ttl{hours: 2160, status: "failed"}, true},
		{ttl{hours: 2160, status: "in_progress"}, false},
	} {
		if got := tc.t.due(2160); got != tc.want {
			t.Errorf("%v due = %v, want %v", tc.t, got, tc.want)
		}
	}
}
