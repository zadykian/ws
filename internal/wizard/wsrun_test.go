package wizard

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

// extra is what mac.conf needs besides its target.
var extra = map[string]string{"cloud": "WS_REMOTE=ws.example.org\nWS_REMOTE_PORT=22022\n"}

// newRun returns the stage over a Machine whose mac.conf holds the target, and whose remote script
// answers status with the text, and with no secrets file or host_vars on ws where the text, if any,
// names none.
func newRun(target, status string, update, check bool) (*WsRun, *effecttest.Machine) {
	m := effecttest.NewMachine()
	m.Files["/h/mac.conf"] = []byte("WS_TARGET=" + target + "\nWS_USER=alice\n" + extra[target])
	h := &HostKey{Conf: "/h/mac.conf", Lock: "/l", Check: check, M: m, Out: &effecttest.Out{},
		SSH: SSH{M: m, Lock: "/l"}}
	r := &WsRun{Host: h, Want: h.ReadConf, Check: check, Out: h.Out, Version: "v1.2.3", Update: update, Default: "alice", Remote: &Remote{Shell: SSHShell{SSH: h.SSH}, Script: []byte("x")},
		Files: &Remote{Shell: SSHShell{SSH: h.SSH}, Script: []byte("y")}}
	if status != "" {
		status += "secrets -\nvars -\n"
	}
	m.Results[remoteLine(r.Remote, "status")] = effecttest.Result{Out: []byte(status)}
	return r, m
}

func TestWsRunPlan(t *testing.T) {
	for name, tc := range map[string]struct {
		target, status, want string
		update, check        bool
	}{
		"done":       {"lan", "run v1.2.3 0 on - -\nactive -\n", "", false, false},
		"older ws":   {"lan", "run v1.2.2 0 on - -\nactive -\n", "run ws server apply on ws", false, false},
		"first run":  {"lan", "run - -\nactive -\n", "run ws server apply on ws", false, false},
		"update":     {"lan", "run v1.2.3 0 on - -\nactive -\n", "run ws server apply on ws", true, false},
		"cloud done": {"cloud", "run v1.2.3 0 off - -\nactive -\noutbound off\n", "", false, false},
		"cloud host_vars": {"cloud", "run v1.2.3 0 off - -\nactive -\noutbound on\n",
			"name security_outbound: false in ws's host_vars", false, false},
		"cloud run": {"cloud", "run v1.2.3 0 on - -\nactive -\n", "after naming security_outbound: false in ws's host_vars", false, false},
		"follows":   {"lan", "run - -\nactive ws-wizard-1\n", "follow ws-wizard-1, which runs on ws", false, true},
		"failed":    {"lan", "run v1.2.3 1 on - -\nactive -\n", "its last full run exited 1", false, true},
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
		"changed file": {"aa11", "bb22", "run ws server apply on ws in a unit of its own, and follow its log, with ws's secrets file"},
		"a new file":   {"-", "bb22", ", with ws's secrets file"},
		"file gone":    {"aa11", "-", "and follow its log"},
		"no file":      {"-", "-", ""},
		"no line":      {"-", "", "and follow its log"},
	} {
		r, _ := newRun("lan", "run v1.2.3 0 on "+tc.stamp+" -\nactive -\nsecrets "+tc.now+"\n", false, false)
		p, err := r.Plan(context.Background())
		got := ""
		if len(p.Changes) > 0 {
			got = p.Changes[0].String()
		}
		if err != nil || (tc.want == "") != (got == "") || !strings.Contains(got, tc.want) ||
			(tc.now == "-" || tc.now == "") && strings.Contains(got, "secrets file") {
			t.Errorf("%s: Plan = %q, %v; want %q", name, got, err, tc.want)
		}
	}
}

// The run is done where its stamp names the sum of ws's host_vars that the status reads now, so a
// change by either wizard, Configure or by hand makes it due.
func TestWsRunIsDoneAtHostVarsSum(t *testing.T) {
	for name, tc := range map[string]struct{ stamp, now, want string }{
		"same file":      {" aa11", "aa11", ""},
		"changed file":   {" aa11", "bb22", "run ws server apply on ws"},
		"a new file":     {" -", "bb22", "run ws server apply on ws"},
		"no file":        {" -", "-", ""},
		"an older stamp": {"", "-", "run ws server apply on ws"},
	} {
		r, _ := newRun("lan", "run v1.2.3 0 on -"+tc.stamp+"\nactive -\nvars "+tc.now+"\n", false, false)
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

func TestWsRunNamesWsUser(t *testing.T) {
	done := "run v1.2.3 0 on - -\nactive -\n"
	for name, tc := range map[string]struct{ status, server, want string }{
		"the default":    {done + "user -\n", "alice", ""},
		"no line":        {done + "user -\n", "zadykian", "after naming ws_user: alice in ws's host_vars"},
		"named":          {done + "user alice\n", "zadykian", ""},
		"another":        {done + "user other\n", "alice", "after naming ws_user: alice in ws's host_vars"},
		"an older ws":    {"run v1.2.2 0 on - -\nactive -\nuser alice\n", "zadykian", "run ws server apply on ws"},
		"status with no": {done, "zadykian", "after naming ws_user: alice"},
	} {
		r, _ := newRun("lan", tc.status, false, false)
		r.Default = tc.server
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

func TestWsRunNamesWsUserBeforeTheRun(t *testing.T) {
	r, m := newRun("lan", "run v1.2.3 0 on - -\nactive -\nuser -\n", false, false)
	r.Default = "zadykian"
	p, _ := r.Plan(context.Background())
	err := r.Apply(context.Background(), p)
	if err == nil || !strings.Contains(err.Error(), "ws's host_vars lacks ws_user: alice") {
		t.Fatalf("Apply where the files script fails = %v; want the stop", err)
	}
	m.Results[remoteLine(r.Files, "user", "alice")] = effecttest.Result{}
	m.Results[remoteLine(r.Remote, "run")] = effecttest.Result{}
	m.Calls = nil
	if err := r.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	all := strings.Join(m.Calls, "\n")
	if i := strings.Index(all, "-- user alice"); i < 0 || i > strings.Index(all, "-- run") {
		t.Errorf("calls = %q; want the name before the run", m.Calls)
	}
}

func TestWsRunUpdateWaitsForTheRun(t *testing.T) {
	r, _ := newRun("lan", "run v1.2.3 0 on - -\nactive -\n", true, false)
	r.ran = true
	if p, err := r.Plan(context.Background()); err != nil || len(p.Changes) != 0 {
		t.Errorf("Plan after the run = %v, %v; want none", p, err)
	}
}

func TestWsRunApply(t *testing.T) {
	r, m := newRun("lan", "run - -\nactive -\n", false, false)
	p, _ := r.Plan(context.Background())
	if err := r.Apply(context.Background(), p); err == nil || !strings.Contains(err.Error(), "ws server apply stopped") {
		t.Errorf("Apply where ssh has no result = %v; want the stop", err)
	}
	m.Results[remoteLine(r.Remote, "run")] = effecttest.Result{}
	if err := r.Apply(context.Background(), p); err != nil || !r.ran {
		t.Errorf("Apply = %v, ran %v; want a run", err, r.ran)
	}
}

func TestWsRunTakesItsPart(t *testing.T) {
	for name, status := range map[string]string{
		"a clean stage": "run v1.2.3 0 on - -\nactive -\n",
		"a stage due":   "run - -\nactive -\n",
		"a cloud VM":    "run v1.2.3 0 off - -\nactive -\noutbound off\n",
	} {
		target := "lan"
		if name == "a cloud VM" {
			target = "cloud"
		}
		r, m := newRun(target, status, false, false)
		r.Only = []string{"signoz/signoz up", "fans"}
		line := remoteLine(r.Remote, "run", "--only=signoz/signoz up", "--only=fans")
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
	r, _ := newRun("lan", "run - -\nactive -\n", false, false)
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
	r, _ := newRun("lan", "run - -\nactive -\n", false, false)
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
	r, m := newRun("lan", "run v1.2.3 0 on - aa11\nactive -\nvars aa11\n", true, false)
	p, _ := r.Plan(context.Background())
	m.Results[remoteLine(r.Remote, "run")] = effecttest.Result{}
	if err := r.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	if runs := strings.Count(strings.Join(m.Calls, "\n"), "-- run"); runs != 1 {
		t.Errorf("calls = %q; want one run", m.Calls)
	}
}

func TestWsRunApplyCloudAgain(t *testing.T) {
	r, m := newRun("cloud", "run v1.2.3 0 on - aa11\nactive -\noutbound on\nvars bb22\n", false, false)
	p, _ := r.Plan(context.Background())
	m.Results[remoteLine(r.Remote, "run")] = effecttest.Result{}
	m.Results[remoteLine(r.Remote, "outbound")] = effecttest.Result{}
	if err := r.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	runs := 0
	for _, c := range m.Calls {
		if strings.HasSuffix(c, "-- run") {
			runs++
		}
	}
	if runs != 2 || !strings.Contains(strings.Join(m.Calls, "\n"), "-- outbound") {
		t.Errorf("calls = %q; want two runs, as the stamp names another host_vars, and outbound", m.Calls)
	}
}

// On ws with no booted systemd, as in a container, the stage reads nothing and skips with a line.
func TestWsRunSkipsUnbooted(t *testing.T) {
	r, m := newRun("lan", "", false, true)
	r.Unbooted, r.Await, m.Calls = true, "ws's values", nil
	p, err := r.Plan(context.Background())
	if err != nil || len(p.Changes) != 0 || len(m.Calls) != 0 || r.WaitsOn() != "ws's values" {
		t.Fatalf("Plan = %v, %v; calls %q; waits on %q", p, err, m.Calls, r.WaitsOn())
	}
	if out := r.Out.(*effecttest.Out); len(out.Lines) != 1 || !strings.Contains(out.Lines[0], "isn't booted") {
		t.Errorf("said %q", out.Lines)
	}
}
