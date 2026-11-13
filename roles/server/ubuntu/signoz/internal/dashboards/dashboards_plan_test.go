package dashboards

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/server/ubuntu/signoz/internal/api"
)

// fakeDashboards is SigNoz's v2 dashboards API: one list, a read by ID, a create and a replace.
type fakeDashboards struct {
	mu    sync.Mutex
	have  map[string]dashboard
	calls []string
}

func (f *fakeDashboards) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	switch {
	case r.Method == http.MethodGet && r.URL.Path == "/api/v2/dashboards":
		var list []dashboard
		for _, d := range f.have {
			list = append(list, dashboard{ID: d.ID, Name: d.Name})
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"dashboards": list, "total": len(list)}})
	case r.Method == http.MethodGet:
		id := strings.TrimPrefix(r.URL.Path, "/api/v2/dashboards/")
		_ = json.NewEncoder(w).Encode(map[string]any{"data": f.have[id]})
	default:
		body, _ := io.ReadAll(r.Body)
		f.calls = append(f.calls, r.Method+" "+r.URL.Path+" "+string(body[:10]))
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusCreated)
		}
	}
}

func dashboardsFixture(t *testing.T, f *fakeDashboards, names ...string) (Dashboards, *effecttest.Out) {
	srv := httptest.NewServer(f)
	t.Cleanup(srv.Close)
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	m.Dirs["/run/systemd/system"] = 0o755
	return Dashboards{M: m, Out: out, API: api.Client{URL: srv.URL}, Names: names}, out
}

// fromFile is the dashboard that SigNoz holds once the file of name is applied.
func fromFile(t *testing.T, name, id string) dashboard {
	d, _, err := load(name)
	if err != nil {
		t.Fatal(err)
	}
	d.ID = id
	return d
}

func TestDashboardsCreateWhatSigNozLacks(t *testing.T) {
	f := &fakeDashboards{have: map[string]dashboard{"1": fromFile(t, "host", "1")}}
	s, _ := dashboardsFixture(t, f, "host", "claude")
	p, err := s.Plan(t.Context())
	if err != nil || len(p.Changes) != 1 || p.Changes[0].String() != "create the dashboard claude" {
		t.Fatalf("plan = %v, %v; want to create claude", p.Changes, err)
	}
	if err := s.Apply(t.Context(), p); err != nil {
		t.Fatal(err)
	}
	if len(f.calls) != 1 || !strings.HasPrefix(f.calls[0], "POST /api/v2/dashboards ") {
		t.Errorf("calls = %q", f.calls)
	}
}

func TestDashboardsReplaceOneThatDiffers(t *testing.T) {
	d := fromFile(t, "host", "7")
	d.Tags = json.RawMessage(`[]`)
	s, _ := dashboardsFixture(t, &fakeDashboards{have: map[string]dashboard{"7": d}}, "host")
	p, err := s.Plan(t.Context())
	if err != nil || len(p.Changes) != 1 || p.Changes[0].String() != "replace the dashboard host" {
		t.Fatalf("plan = %v, %v; want to replace host", p.Changes, err)
	}
}

func TestDashboardsLeaveOneLockedAndSayso(t *testing.T) {
	d := fromFile(t, "host", "7")
	d.Tags, d.Locked = json.RawMessage(`[]`), true
	s, out := dashboardsFixture(t, &fakeDashboards{have: map[string]dashboard{"7": d}}, "host")
	p, err := s.Plan(t.Context())
	if err != nil || len(p.Changes) != 0 || !strings.Contains(strings.Join(out.Lines, "\n"), "locked") {
		t.Errorf("plan = %v, %v, said %q; want none and a word on the lock", p.Changes, err, strings.Join(out.Lines, "\n"))
	}
}

func TestDashboardsHaveNoChangeWhereTheyMatch(t *testing.T) {
	f := &fakeDashboards{have: map[string]dashboard{"1": fromFile(t, "host", "1"), "2": fromFile(t, "claude", "2")}}
	s, _ := dashboardsFixture(t, f, "host", "claude")
	if p, err := s.Plan(t.Context()); err != nil || len(p.Changes) != 0 {
		t.Errorf("plan = %v, %v; want none", p.Changes, err)
	}
}

func TestDashboardsSkipWithoutSystemd(t *testing.T) {
	s, out := dashboardsFixture(t, &fakeDashboards{}, "host")
	delete(s.M.(*effecttest.Machine).Dirs, "/run/systemd/system")
	if p, err := s.Plan(t.Context()); err != nil || len(p.Changes) != 0 || !strings.Contains(strings.Join(out.Lines, "\n"), "skipped") {
		t.Errorf("plan = %v, %v, said %q", p.Changes, err, strings.Join(out.Lines, "\n"))
	}
}
