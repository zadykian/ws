package signoz

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
)

// signozAPI serves the retention of traces and metrics from a fakeTTL, and of logs from days.
func signozAPI(t *testing.T, f *fakeTTL, logsDays *int) Retention {
	mux := http.NewServeMux()
	mux.Handle("/api/v1/settings/ttl", f)
	mux.HandleFunc("/api/v2/settings/ttl", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			*logsDays = 90
			return
		}
		_, _ = fmt.Fprintf(w, `{"default_ttl_days": %d, "status": "success", "ttl_conditions": []}`, *logsDays)
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	m := effecttest.NewMachine()
	m.Dirs["/run/systemd/system"] = 0o755
	m.Results[lsLine] = effecttest.Result{Out: []byte("Exited (0) 1 hour ago\n")}
	w := Wait{Tries: 5, Delay: time.Second, Sleep: func(context.Context, time.Duration) error { return nil }}
	return Retention{M: m, Out: &effecttest.Out{}, API: API{URL: srv.URL}, Days: 90, Migrator: w, Altered: w}
}

func newTTL(traces, metrics int) *fakeTTL {
	return &fakeTTL{hours: map[string]int{"traces": traces, "metrics": metrics},
		status: map[string]string{"traces": "success", "metrics": "success"}, pending: map[string]int{}}
}

func TestRetentionPlansWhatDiffers(t *testing.T) {
	days := 15
	r := signozAPI(t, newTTL(360, 2160), &days)
	p, err := r.Plan(t.Context())
	if err != nil || len(p.Changes) != 2 {
		t.Fatalf("plan = %v, %v; want traces and logs", p.Changes, err)
	}
}

func TestRetentionSetsWhatDiffersAndThenHasNoChange(t *testing.T) {
	days := 15
	r := signozAPI(t, newTTL(360, 2160), &days)
	p, err := r.Plan(t.Context())
	if err != nil || len(p.Changes) != 2 {
		t.Fatalf("plan = %v, %v; want traces and logs", p.Changes, err)
	}
	if err := r.Apply(t.Context(), p); err != nil {
		t.Fatal(err)
	}
	if again, err := r.Plan(t.Context()); err != nil || len(again.Changes) != 0 {
		t.Errorf("second plan = %v, %v; want none", again.Changes, err)
	}
}

func TestRetentionHasNoChangeWhereSigNozKeepsTheDays(t *testing.T) {
	days := 90
	r := signozAPI(t, newTTL(2160, 2160), &days)
	if changed, err := step.Run(t.Context(), []step.Step{noWaitRetention{r}}, r.Out, true); err != nil || changed {
		t.Errorf("check = %v, %v; want no change", changed, err)
	}
}

// noWaitRetention is Retention without its wait on a step the test does not run.
type noWaitRetention struct{ Retention }

func (noWaitRetention) WaitsOn() string { return "" }

func TestRetentionNamesTheSignalsItWouldSet(t *testing.T) {
	days := 90
	r := signozAPI(t, newTTL(2160, 720), &days)
	p, _ := r.Plan(t.Context())
	got := fmt.Sprint(p.Changes)
	if want := "[set the retention of metrics (now metrics: 720 h, success) to 2160 h]"; got != want {
		t.Errorf("changes = %s, want %s", got, want)
	}
}

func TestRetentionSkipsWithoutSystemd(t *testing.T) {
	days := 15
	r := signozAPI(t, newTTL(360, 360), &days)
	delete(r.M.(*effecttest.Machine).Dirs, "/run/systemd/system")
	if p, err := r.Plan(t.Context()); err != nil || len(p.Changes) != 0 {
		t.Errorf("plan = %v, %v; want a skip", p.Changes, err)
	}
}
