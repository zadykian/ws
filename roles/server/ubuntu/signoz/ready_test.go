package signoz

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/server/ubuntu/signoz/internal/api"
)

// readyFixture is a booted machine and a SigNoz that answers 503 for the first fails requests.
func readyFixture(t *testing.T, fails int32) (Ready, *effecttest.Out) {
	var n atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/readyz" || n.Add(1) <= fails {
			http.Error(w, "starting", http.StatusServiceUnavailable)
		}
	}))
	t.Cleanup(srv.Close)
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	m.Dirs["/run/systemd/system"] = 0o755
	wait := api.Wait{Tries: 3, Delay: time.Second, Sleep: func(context.Context, time.Duration) error { return nil }}
	return Ready{M: m, Out: out, API: api.Client{URL: srv.URL}, Wait: wait}, out
}

func TestReadyHasNoChangeOnceSigNozAnswers(t *testing.T) {
	r, _ := readyFixture(t, 0)
	if p, err := r.Plan(t.Context()); err != nil || len(p.Changes) != 0 {
		t.Errorf("plan = %v, %v; want none", p.Changes, err)
	}
}

func TestReadyWaitsForTheAnswer(t *testing.T) {
	r, _ := readyFixture(t, 2)
	p, err := r.Plan(t.Context())
	if err != nil || len(p.Changes) != 1 {
		t.Fatalf("plan = %v, %v; want one change", p.Changes, err)
	}
	if err := r.Apply(t.Context(), p); err != nil {
		t.Errorf("apply: %v, want the third try to answer", err)
	}
}

func TestReadyGivesUp(t *testing.T) {
	r, _ := readyFixture(t, 100)
	p, _ := r.Plan(t.Context())
	if err := r.Apply(t.Context(), p); err == nil {
		t.Error("apply succeeded, want it to give up")
	}
}

func TestReadySkipsWithoutSystemd(t *testing.T) {
	r, out := readyFixture(t, 100)
	delete(r.M.(*effecttest.Machine).Dirs, "/run/systemd/system")
	if p, err := r.Plan(t.Context()); err != nil || len(p.Changes) != 0 || len(out.Lines) != 1 {
		t.Errorf("plan = %v, %v, said %q; want a skip said once", p.Changes, err, out.Lines)
	}
}
