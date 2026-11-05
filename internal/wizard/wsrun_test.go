package wizard

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

// extra is what mac.conf needs besides its target.
var extra = map[string]string{"cloud": "WS_REMOTE=ws.example.org\nWS_REMOTE_PORT=22022\n"}

// newRun returns the stage over a Machine whose mac.conf holds the target, and whose remote script
// answers status with the text.
func newRun(target, status string, update, check bool) (*WsRun, *effecttest.Machine) {
	m := effecttest.NewMachine()
	m.Files["/h/mac.conf"] = []byte("WS_TARGET=" + target + "\nWS_USER=alice\n" + extra[target])
	h := &HostKey{Conf: "/h/mac.conf", Lock: "/l", Check: check, M: m, Out: &effecttest.Out{},
		SSH: SSH{M: m, Lock: "/l"}}
	r := &WsRun{Host: h, Version: "v1.2.3", Update: update, Default: "alice", Remote: &Remote{SSH: h.SSH, Script: []byte("x")},
		Files: &Remote{SSH: h.SSH, Script: []byte("y")}}
	m.Results[remoteLine(r.Remote, "status")] = effecttest.Result{Out: []byte(status)}
	m.Results[remoteLine(r.Remote, "secrets", "drop")] = effecttest.Result{}
	return r, m
}

func TestWsRunPlan(t *testing.T) {
	for name, tc := range map[string]struct {
		target, status, want string
		update, check        bool
	}{
		"done":       {"lan", "run v1.2.3 0 on -\nactive -\n", "", false, false},
		"older ws":   {"lan", "run v1.2.2 0 on -\nactive -\n", "run ws server apply on ws", false, false},
		"first run":  {"lan", "run - -\nactive -\n", "run ws server apply on ws", false, false},
		"update":     {"lan", "run v1.2.3 0 on -\nactive -\n", "run ws server apply on ws", true, false},
		"cloud done": {"cloud", "run v1.2.3 0 off -\nactive -\noutbound off\n", "", false, false},
		"cloud host_vars": {"cloud", "run v1.2.3 0 off -\nactive -\noutbound on\n",
			"name security_outbound: false in ws's host_vars", false, false},
		"cloud run": {"cloud", "run v1.2.3 0 on -\nactive -\n", "after naming security_outbound: false in ws's host_vars", false, false},
		"follows":   {"lan", "run - -\nactive ws-wizard-1\n", "follow ws-wizard-1, which runs on ws", false, true},
		"failed":    {"lan", "run v1.2.3 1 on -\nactive -\n", "its last full run exited 1", false, true},
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

func TestWsRunSeesWsUser(t *testing.T) {
	done := "run v1.2.3 0 on -\nactive -\n"
	for name, tc := range map[string]struct{ status, server, want string }{
		"the default":    {done + "user -\n", "alice", ""},
		"no line":        {done + "user -\n", "zadykian", "run ws server apply on ws"},
		"named":          {done + "user alice\n", "zadykian", ""},
		"another":        {done + "user other\n", "alice", "run ws server apply on ws"},
		"an older ws":    {"run v1.2.2 0 on -\nactive -\nuser alice\n", "zadykian", "run ws server apply on ws"},
		"status with no": {done, "zadykian", "run ws server apply on ws"},
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

func TestWsRunUpdateWaitsForTheRun(t *testing.T) {
	r, _ := newRun("lan", "run v1.2.3 0 on -\nactive -\n", true, false)
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

func TestWsRunApplyCloudAgain(t *testing.T) {
	r, m := newRun("cloud", "run v1.2.3 0 on -\nactive -\noutbound on\n", false, false)
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
		t.Errorf("calls = %q; want two runs, as the status says on, and outbound", m.Calls)
	}
}
