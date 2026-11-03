package wizard

import (
	"context"
	"crypto/sha256"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/step"
)

func newHookWs(conf, status string) (*HookWs, *effecttest.Machine) {
	r, m := newRun("lan", "run v1.2.3 0 on\nactive -\n"+status, false, false)
	m.Files["/h/mac.conf"] = append(m.Files["/h/mac.conf"], []byte(conf)...)
	mac := &HookMac{Home: "/h", Gh: &Gh{LookPath: func(string) (string, error) { return "/bin/gh", nil }, M: m}}
	files := &Remote{SSH: r.Host.SSH, Script: []byte("y")}
	return &HookWs{Host: r.Host, Run: r, Mac: mac, Files: files, User: "me", Tty: true, M: m, Out: r.Host.Out}, m
}

func TestHookWsPlan(t *testing.T) {
	for name, tc := range map[string]struct{ conf, status, want string }{
		"no hook":  {"", "", ""},
		"stamped":  {"WS_HOOK=me/tools@v1.0.0\n", "hook other/x me/tools@v1.0.0\n", ""},
		"old tag":  {"WS_HOOK=me/tools@v1.0.1\n", "hook me/tools@v1.0.0\n", "release v1.0.1 with the Mac's gh"},
		"run":      {"WS_HOOK=me/tools@v1.0.0\n", "", "then run tools apply there as root, its stdin empty"},
		"keychain": {"WS_HOOK=me/tools@v1.0.0\nWS_HOOK_SECRET=tok\n", "", "security add-generic-password -a me -s tok -w"},
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

// sshLine returns the call that the machine records for an ssh of the arguments.
func sshLine(h *HookWs, args ...string) string {
	return "ssh " + strings.Join(h.Host.SSH.argv(args), " ")
}

// stubRelease gives the fake machine a release of me/tools@v1.0.0 for arch, and the calls that
// send and install it. sums is the content of tools.sha256.
func stubRelease(h *HookWs, m *effecttest.Machine, build, sums string) {
	m.Results[remoteLine(h.Files, "arch")] = effecttest.Result{Out: []byte("arm64\n")}
	m.Results["env GH_PROMPT_DISABLED=1 /bin/gh release download v1.0.0 -R me/tools -D /l/hook-release "+
		"--clobber -p tools-linux-arm64 -p tools.sha256"] = effecttest.Result{}
	m.Files["/l/hook-release/tools-linux-arm64"] = []byte(build)
	m.Files["/l/hook-release/tools.sha256"] = []byte(sums)
	m.Results[remoteLine(h.Files, "stage", "me/tools@v1.0.0")] = effecttest.Result{Out: []byte("/s/hook-tools.bin\n")}
	m.Results[sshLine(h, "-o", "BatchMode=yes", "ws-ssh", "umask 077 && cat >/s/hook-tools.bin")] = effecttest.Result{}
}

func sumLine(build, name string) string {
	return fmt.Sprintf("%x  %s\n", sha256.Sum256([]byte(build)), name)
}

func TestHookWsApply(t *testing.T) {
	h, m := newHookWs("WS_HOOK=me/tools@v1.0.0\n", "")
	bin := hookState + "/bin/tools-v1.0.0"
	sums := sumLine("build", "tools-linux-arm64") + sumLine("other", "tools-linux-amd64")
	stubRelease(h, m, "build", sums)
	sum := fmt.Sprintf("%x", sha256.Sum256([]byte("build")))
	install := remoteLine(h.Files, "install", "me/tools@v1.0.0", sum)
	run := sshLine(h, "-o", "BatchMode=yes", "ws-ssh", bin, "apply")
	m.Results[run] = effecttest.Result{}
	m.Results[remoteLine(h.Files, "stamp", "me/tools@v1.0.0")] = effecttest.Result{}
	if err := h.Apply(context.Background(), step.Plan{}); err == nil || !strings.Contains(err.Error(), "failed, as above") {
		t.Errorf("Apply where install fails = %v", err)
	}
	m.Results[install] = effecttest.Result{}
	if err := h.Apply(context.Background(), step.Plan{}); err != nil || !h.ran {
		t.Errorf("Apply = %v, ran %v", err, h.ran)
	}
	calls := strings.Join(m.Calls, "\n")
	if !strings.Contains(calls, "\n"+"run "+run+"\n") || strings.Contains(calls, "clone") || strings.Contains(calls, " -t ") {
		t.Errorf("Apply ran\n%s", calls)
	}
	if !slices.Contains(m.Inputs, "build") {
		t.Errorf("Apply sent %q; want the build", m.Inputs)
	}
}

func TestHookWsApplyBadSum(t *testing.T) {
	for name, sums := range map[string]string{
		"differs": sumLine("other", "tools-linux-arm64"),
		"absent":  sumLine("build", "tools-linux-amd64"),
		"none":    "",
	} {
		h, m := newHookWs("WS_HOOK=me/tools@v1.0.0\n", "")
		stubRelease(h, m, "build", sums)
		err := h.Apply(context.Background(), step.Plan{})
		if err == nil || !strings.Contains(err.Error(), "is not the one that tools.sha256 lists") {
			t.Errorf("%s: Apply = %v", name, err)
		}
		for _, c := range m.Calls {
			if strings.Contains(c, "stage") || strings.Contains(c, "cat >") {
				t.Errorf("%s: Apply went on to ws: %q", name, c)
			}
		}
	}
}

func TestHookWsApplyNoRelease(t *testing.T) {
	h, m := newHookWs("WS_HOOK=me/tools@v1.0.0\n", "")
	m.Results[remoteLine(h.Files, "arch")] = effecttest.Result{Out: []byte("amd64\n")}
	err := h.Apply(context.Background(), step.Plan{})
	if err == nil || !strings.Contains(err.Error(), "needs tools-linux-amd64 and tools.sha256") {
		t.Errorf("Apply where gh fails = %v", err)
	}
}

func TestHookWsApplyUnreadItem(t *testing.T) {
	h, m := newHookWs("WS_HOOK=me/tools@v1.0.0\nWS_HOOK_SECRET=tok\n", "")
	m.Results["security find-generic-password -s tok -w"] = effecttest.Result{Err: context.Canceled}
	if err := h.Apply(context.Background(), step.Plan{}); err == nil || !strings.Contains(err.Error(), "can't read the item") {
		t.Errorf("Apply with an unreadable item = %v", err)
	}
	for _, c := range m.Calls {
		if strings.Contains(c, "ssh") || strings.Contains(c, "release") {
			t.Errorf("Apply touched ws or GitHub before reading the item: %q", c)
		}
	}
}

func TestHookWsCheck(t *testing.T) {
	h, m := newHookWs("WS_HOOK=me/tools@v1.0.0\n", "")
	h.Host.Check = true
	bin := hookState + "/bin/tools-v1.0.0"
	have := remoteLine(h.Files, "have", "me/tools@v1.0.0")
	m.Results[have] = effecttest.Result{}
	if _, err := h.Plan(context.Background()); err != nil || strings.Contains(strings.Join(m.Calls, "\n"), "--check") {
		t.Errorf("Plan where ws lacks the command = %v, calls %q", err, m.Calls)
	}
	m.Results[have] = effecttest.Result{Out: []byte(bin + "\n")}
	m.Results[sshLine(h, "-o", "BatchMode=yes", "ws-ssh", bin, "apply", "--check")] = effecttest.Result{}
	p, err := h.Plan(context.Background())
	if err != nil || len(p.Changes) != 1 || !slices.Contains(m.Calls, "run "+sshLine(h, "-o", "BatchMode=yes", "ws-ssh", bin, "apply", "--check")) {
		t.Errorf("Plan where ws holds the command = %v, %v, calls %q", p, err, m.Calls)
	}
}

func TestHookWsApplyGuard(t *testing.T) {
	h, m := newHookWs("WS_HOOK=me/tools@v1.0.0\nWS_HOOK_SECRET=tok\n", "")
	m.Results["security find-generic-password -s tok -w"] = effecttest.Result{Out: []byte("s3cret\n")}
	stubRelease(h, m, "build", sumLine("build", "tools-linux-arm64"))
	sum := fmt.Sprintf("%x", sha256.Sum256([]byte("build")))
	m.Results[remoteLine(h.Files, "install", "me/tools@v1.0.0", sum)] = effecttest.Result{}
	bin := hookState + "/bin/tools-v1.0.0"
	m.Results[sshLine(h, "-o", "BatchMode=yes", "ws-ssh", "sh -c '"+hookGuard+"'", bin, "apply")] = effecttest.Result{}
	m.Results[remoteLine(h.Files, "stamp", "me/tools@v1.0.0")] = effecttest.Result{}
	if err := h.Apply(context.Background(), step.Plan{}); err != nil {
		t.Fatalf("Apply = %v", err)
	}
	if !slices.Contains(m.Inputs, "s3cret\n") {
		t.Errorf("the command's stdin was none of %q; want the item's secret", m.Inputs)
	}
}
