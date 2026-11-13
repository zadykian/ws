package stages

import (
	"context"
	"github.com/zadykian/ws/roles/server/ubuntu/session"
	"github.com/zadykian/ws/roles/server/ubuntu/session/sessiontest"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"

	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

func TestWsRunPlan(t *testing.T) {
	off := session.Stamp{Version: "v1.2.3", Exit: "0", Off: true}
	for name, tc := range map[string]struct {
		target        string
		status        session.Status
		want          string
		update, check bool
	}{
		"done":       {"lan", idleRun, "", false, false},
		"older ws":   {"lan", session.Status{Run: session.Stamp{Version: "v1.2.2", Exit: "0"}}, "run ws server ubuntu apply on ws", false, false},
		"first run":  {"lan", session.Status{}, "run ws server ubuntu apply on ws", false, false},
		"update":     {"lan", idleRun, "run ws server ubuntu apply on ws", true, false},
		"cloud done": {"cloud", session.Status{Run: off, OutboundOff: true}, "", false, false},
		"cloud host_vars": {"cloud", session.Status{Run: session.Stamp{Version: "v1.2.3", Exit: "0", Off: true, Vars: "aa11"}, Vars: "bb22"},
			"after naming security_outbound: false in ws's host_vars", false, false},
		"cloud run": {"cloud", idleRun, "after naming security_outbound: false in ws's host_vars", false, false},
		"follows":   {"lan", session.Status{Active: "ws-wizard-1"}, "follow ws-wizard-1, which runs on ws", false, true},
		"failed":    {"lan", session.Status{Run: session.Stamp{Version: "v1.2.3", Exit: "1"}}, "its last full run exited 1", false, true},
	} {
		r, _ := newRun(tc.target, tc.status, tc.update, tc.check)
		p, err := r.Plan(context.Background())
		got := ""
		if len(p.Changes) > 0 {
			got = p.Changes[0].String()
		}
		if err != nil || (tc.want == "") != (got == "") || !strings.Contains(got, tc.want) {
			t.Errorf("%s: Plan = %q, %v; want %q", name, got, err, tc.want)
		}
	}
}

// The run is done where its stamp names the sum of ws's secrets file that the status reads now.
func TestWsRunIsDoneAtWsFilesSum(t *testing.T) {
	for name, tc := range map[string]struct{ stamp, now, want string }{
		"same file":    {"aa11", "aa11", ""},
		"changed file": {"aa11", "bb22", "run ws server ubuntu apply on ws in a unit of its own, and follow its log, with ws's secrets file"},
		"a new file":   {"", "bb22", ", with ws's secrets file"},
		"file gone":    {"aa11", "", "and follow its log"},
		"no file":      {"", "", ""},
	} {
		r, _ := newRun("lan", session.Status{Run: session.Stamp{Version: "v1.2.3", Exit: "0", Secrets: tc.stamp}, Secrets: tc.now}, false, false)
		p, err := r.Plan(context.Background())
		got := ""
		if len(p.Changes) > 0 {
			got = p.Changes[0].String()
		}
		if err != nil || (tc.want == "") != (got == "") || !strings.Contains(got, tc.want) ||
			tc.now == "" && strings.Contains(got, "secrets file") {
			t.Errorf("%s: Plan = %q, %v; want %q", name, got, err, tc.want)
		}
	}
}

// The run is done where its stamp names the sum of ws's host_vars that the status reads now. So a
// change by either wizard, Configure or by hand makes it due.
func TestWsRunIsDoneAtHostVarsSum(t *testing.T) {
	at := func(vars string) session.Stamp { return session.Stamp{Version: "v1.2.3", Exit: "0", Vars: vars} }
	for name, tc := range map[string]struct {
		run       session.Stamp
		now, want string
	}{
		"same file":    {at("aa11"), "aa11", ""},
		"changed file": {at("aa11"), "bb22", "run ws server ubuntu apply on ws"},
		"a new file":   {at(""), "bb22", "run ws server ubuntu apply on ws"},
		"no file":      {at(""), "", ""},
		"no stamp":     {session.Stamp{}, "", "run ws server ubuntu apply on ws"},
	} {
		r, _ := newRun("lan", session.Status{Run: tc.run, Vars: tc.now}, false, false)
		p, err := r.Plan(context.Background())
		got := ""
		if len(p.Changes) > 0 {
			got = p.Changes[0].String()
		}
		if err != nil || (tc.want == "") != (got == "") || !strings.Contains(got, tc.want) {
			t.Errorf("%s: Plan = %q, %v; want %q", name, got, err, tc.want)
		}
	}
}

// asks sets the user that mac.conf wants ws's run to leave.
func asks(m *effecttest.Machine, user string) {
	m.Files["/h/mac.conf"] = []byte("WS_TARGET=lan\nWS_USER=" + user + "\n")
}

func TestWsRunNamesWsUser(t *testing.T) {
	for name, tc := range map[string]struct{ named, want, plan string }{
		"the default": {"", values.Default().User, ""},
		"none named":  {"", "alice", "after naming ws_user: alice in ws's host_vars"},
		"named":       {"alice", "alice", ""},
		"another":     {"other", "alice", "after naming ws_user: alice in ws's host_vars"},
	} {
		s := idleRun
		s.User = tc.named
		r, m := newRun("lan", s, false, false)
		asks(m, tc.want)
		p, err := r.Plan(context.Background())
		got := ""
		if len(p.Changes) > 0 {
			got = p.Changes[0].String()
		}
		if err != nil || (tc.plan == "") != (got == "") || !strings.Contains(got, tc.plan) {
			t.Errorf("%s: Plan = %q, %v; want %q", name, got, err, tc.plan)
		}
	}
}

func TestWsRunNamesWsUserBeforeTheRun(t *testing.T) {
	r, m := newRun("lan", idleRun, false, false)
	asks(m, "alice")
	p, _ := r.Plan(context.Background())
	err := r.Apply(context.Background(), p)
	if err == nil || !strings.Contains(err.Error(), "ws's host_vars lacks ws_user: alice") {
		t.Fatalf("Apply where the files script fails = %v; want the stop", err)
	}
	m.Results[sessiontest.Op("user", "alice")] = effecttest.Result{}
	m.Results[sessiontest.Op("run")] = effecttest.Result{}
	m.Calls = nil
	if err := r.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	all := strings.Join(m.Calls, "\n")
	if i := strings.Index(all, "op user alice"); i < 0 || i > strings.Index(all, "op run") {
		t.Errorf("calls = %q; want the name before the run", m.Calls)
	}
}

func TestWsRunUpdateWaitsForTheRun(t *testing.T) {
	r, _ := newRun("lan", idleRun, true, false)
	r.ran = true
	if p, err := r.Plan(context.Background()); err != nil || len(p.Changes) != 0 {
		t.Errorf("Plan after the run = %v, %v; want none", p, err)
	}
}

func TestWsRunApply(t *testing.T) {
	r, m := newRun("lan", session.Status{}, false, false)
	p, _ := r.Plan(context.Background())
	if err := r.Apply(context.Background(), p); err == nil || !strings.Contains(err.Error(), "ws server ubuntu apply stopped") {
		t.Errorf("Apply where ssh has no result = %v; want the stop", err)
	}
	m.Results[sessiontest.Op("run")] = effecttest.Result{}
	if err := r.Apply(context.Background(), p); err != nil || !r.ran {
		t.Errorf("Apply = %v, ran %v; want a run", err, r.ran)
	}
}

func TestWsRunTakesItsPart(t *testing.T) {
	for name, status := range map[string]session.Status{
		"a clean stage": idleRun,
		"a stage due":   {},
		"a cloud VM":    {Run: session.Stamp{Version: "v1.2.3", Exit: "0", Off: true}, OutboundOff: true},
	} {
		target := "lan"
		if name == "a cloud VM" {
			target = "cloud"
		}
		r, m := newRun(target, status, false, false)
		r.Only = []string{"signoz/signoz up", "fans"}
		line := sessiontest.Op("run", "--only=signoz/signoz up", "--only=fans")
		m.Results[line] = effecttest.Result{}
		p, err := r.Plan(context.Background())
		if err != nil || len(p.Changes) != 1 || !strings.Contains(p.Changes[0].String(), "the steps chosen") {
			t.Fatalf("%s: Plan = %v, %v; want the part", name, p, err)
		}
		if err := r.Apply(context.Background(), p); err != nil {
			t.Fatalf("%s: Apply = %v", name, err)
		}
		if !slices.Contains(m.Calls, "run "+line) {
			t.Errorf("%s: calls = %q; want %q", name, m.Calls, line)
		}
		// The runner's second plan: the part ran, though no stamp says so.
		if p, err := r.Plan(context.Background()); err != nil || len(p.Changes) != 0 {
			t.Errorf("%s: Plan after the part = %v, %v; want none", name, p, err)
		}
	}
}

func TestWsRunCutLeavesTheUnitAndSaysSo(t *testing.T) {
	r, _ := newRun("lan", session.Status{}, false, false)
	out := r.Out.(*effecttest.Out)
	p, _ := r.Plan(context.Background())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := r.Apply(ctx, p); err == nil {
		t.Fatal("Apply took a cut run for a whole one")
	}
	if !slices.ContainsFunc(out.Lines, func(l string) bool { return strings.Contains(l, "the unit goes on on ws") }) {
		t.Errorf("lines = %q; want the unit named", out.Lines)
	}
}

func TestWsRunFailureNamesNoUnit(t *testing.T) {
	r, _ := newRun("lan", session.Status{}, false, false)
	out := r.Out.(*effecttest.Out)
	p, _ := r.Plan(context.Background())
	if err := r.Apply(context.Background(), p); err == nil {
		t.Fatal("Apply succeeded with no result for ssh")
	}
	if slices.ContainsFunc(out.Lines, func(l string) bool { return strings.Contains(l, "the unit goes on") }) {
		t.Errorf("lines = %q; a run that failed by itself has no unit left", out.Lines)
	}
}

func TestOnlyArgsQuoteForWsShell(t *testing.T) {
	got := onlyArgs([]string{"a b", "it's"})
	if want := []string{"--only=a b", "--only=it's"}; !slices.Equal(got, want) {
		t.Errorf("onlyArgs = %q, want %q", got, want)
	}
}

// A run whose stamp names host_vars as the status reads it after the run runs once.
func TestWsRunApplyOnceWhereItsStampNamesHostVars(t *testing.T) {
	r, m := newRun("lan", session.Status{Run: session.Stamp{Version: "v1.2.3", Exit: "0", Vars: "aa11"}, Vars: "aa11"}, true, false)
	p, _ := r.Plan(context.Background())
	m.Results[sessiontest.Op("run")] = effecttest.Result{}
	if err := r.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	if runs := strings.Count(strings.Join(m.Calls, "\n"), "op run"); runs != 1 {
		t.Errorf("calls = %q; want one run", m.Calls)
	}
}

func TestWsRunApplyCloudAgain(t *testing.T) {
	r, m := newRun("cloud", session.Status{Run: session.Stamp{Version: "v1.2.3", Exit: "0", Vars: "aa11"}, Vars: "bb22"}, false, false)
	p, _ := r.Plan(context.Background())
	m.Results[sessiontest.Op("run")] = effecttest.Result{}
	m.Results[sessiontest.Op("outbound")] = effecttest.Result{}
	if err := r.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	runs := 0
	for _, c := range m.Calls {
		if strings.HasSuffix(c, "op run") {
			runs++
		}
	}
	if runs != 2 || !strings.Contains(strings.Join(m.Calls, "\n"), "op outbound") {
		t.Errorf("calls = %q; want two runs, as the stamp names another host_vars, and outbound", m.Calls)
	}
}

// On ws with no booted systemd, as in a container, the stage reads nothing and skips with a line.
func TestWsRunSkipsUnbooted(t *testing.T) {
	r, m := newRun("lan", session.Status{}, false, true)
	r.Unbooted, r.Await, m.Calls = true, "ws's values", nil
	p, err := r.Plan(context.Background())
	if err != nil || len(p.Changes) != 0 || len(m.Calls) != 0 || r.WaitsOn() != "ws's values" {
		t.Fatalf("Plan = %v, %v; calls %q; waits on %q", p, err, m.Calls, r.WaitsOn())
	}
	if out := r.Out.(*effecttest.Out); len(out.Lines) != 1 || !strings.Contains(out.Lines[0], "isn't booted") {
		t.Errorf("said %q", out.Lines)
	}
}
