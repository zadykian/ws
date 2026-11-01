package wizard

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/mac/internal/effect/effecttest"
	"github.com/zadykian/ws/mac/internal/step"
)

func newHookWs(conf, status string) (*HookWs, *effecttest.Machine) {
	r, m := newRun("lan", "run abc 0 on\nactive -\n"+status, false, false)
	m.Files["/h/mac.conf"] = append(m.Files["/h/mac.conf"], []byte(conf)...)
	mac := &HookMac{Home: "/h", Gh: &Gh{LookPath: func(string) (string, error) { return "/bin/gh", nil }, M: m}}
	files := &Remote{SSH: r.Host.SSH, Script: []byte("y")}
	return &HookWs{Host: r.Host, Run: r, Mac: mac, Files: files, User: "me", Tty: true, M: m, Out: r.Host.Out}, m
}

func TestHookWsPlan(t *testing.T) {
	for name, tc := range map[string]struct{ conf, status, want string }{
		"no hook":  {"", "", ""},
		"stamped":  {"WS_HOOK=me/tools\n", "hook other/x me/tools\n", ""},
		"run":      {"WS_HOOK=me/tools\n", "", "run its apply.sh there, its stdin empty"},
		"keychain": {"WS_HOOK=me/tools\nWS_HOOK_SECRET=tok\n", "", "security add-generic-password -a me -s tok -w"},
	} {
		h, m := newHookWs(tc.conf, tc.status)
		m.Results["security find-generic-password -s tok"] = effecttest.Result{Err: context.Canceled}
		p, err := h.Plan(context.Background())
		got := errText(err)
		for _, c := range p.Changes {
			got += c.String()
		}
		if (tc.want == "") != (got == "") || !strings.Contains(got, tc.want) {
			t.Errorf("%s: Plan = %q; want %q", name, got, tc.want)
		}
	}
}

func TestHookWsApply(t *testing.T) {
	h, m := newHookWs("WS_HOOK=me/tools\n", "")
	apply := "ssh " + strings.Join(h.Host.SSH.argv([]string{"-o", "BatchMode=yes", "ws-ssh", "/root/repository/tools/apply.sh"}), " ")
	m.Results[remoteLine(h.Files, "bundle", "me/tools")] = effecttest.Result{}
	m.Results[apply] = effecttest.Result{}
	m.Results[remoteLine(h.Files, "stamp", "me/tools")] = effecttest.Result{}
	if err := h.Apply(context.Background(), step.Plan{}); err == nil || !strings.Contains(err.Error(), "failed, as above") {
		t.Errorf("Apply where the clone op fails = %v", err)
	}
	m.Results[remoteLine(h.Files, "clone", "me/tools")] = effecttest.Result{}
	if err := h.Apply(context.Background(), step.Plan{}); err != nil || !h.ran {
		t.Errorf("Apply = %v, ran %v", err, h.ran)
	}
}

func TestHookWsApplyUnreadItem(t *testing.T) {
	h, m := newHookWs("WS_HOOK=me/tools\nWS_HOOK_SECRET=tok\n", "")
	m.Results["security find-generic-password -s tok -w"] = effecttest.Result{Err: context.Canceled}
	if err := h.Apply(context.Background(), step.Plan{}); err == nil || !strings.Contains(err.Error(), "can't read the item") {
		t.Errorf("Apply with an unreadable item = %v", err)
	}
	for _, c := range m.Calls {
		if strings.Contains(c, "ssh") || strings.Contains(c, "bundle") {
			t.Errorf("Apply touched ws before reading the item: %q", c)
		}
	}
}
