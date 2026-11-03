package wizard

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/zadykian/ws/mac/internal/effect/effecttest"
)

// installs gives m the results of the install's commands, in a directory /tmp/x.
func installs(m *effecttest.Machine) {
	m.Results["mktemp -d"] = effecttest.Result{Out: []byte("/tmp/x\n")}
	for _, c := range []string{
		"curl -fsSL --retry 3 --proto-redir =https -o /tmp/x/install.sh http://127.0.0.1:22080/download/v1.3.0/install.sh",
		"env WS_VERSION=v1.3.0 WS_RELEASES_URL=http://127.0.0.1:22080 sh /tmp/x/install.sh", "rm -rf /tmp/x"} {
		m.Results[c] = effecttest.Result{}
	}
}

func TestMacUpdateApply(t *testing.T) {
	u, m, _ := newMacUpdate("v1.2.3", "http://127.0.0.1:22080/tag/v1.3.0")
	installs(m)
	p, _ := u.Plan(context.Background())
	if err := u.Apply(context.Background(), p); err != nil {
		t.Fatalf("Apply = %v", err)
	}
	want := []string{
		"run curl -fsS -o /dev/null -w %{redirect_url} http://127.0.0.1:22080/latest",
		"run mktemp -d",
		"run curl -fsSL --retry 3 --proto-redir =https -o /tmp/x/install.sh http://127.0.0.1:22080/download/v1.3.0/install.sh",
		"run env WS_VERSION=v1.3.0 WS_RELEASES_URL=http://127.0.0.1:22080 sh /tmp/x/install.sh",
		"run rm -rf /tmp/x",
		"exec /h/.local/bin/ws mac wizard --update",
	}
	if got := strings.Join(m.Calls, "\n"); got != strings.Join(want, "\n") {
		t.Errorf("Apply made the calls\n%s\nwant\n%s", got, strings.Join(want, "\n"))
	}
}

func TestMacUpdateApplyFails(t *testing.T) {
	u, m, _ := newMacUpdate("v1.2.3", "http://127.0.0.1:22080/tag/v1.3.0")
	installs(m)
	p, _ := u.Plan(context.Background())
	m.Results["env WS_VERSION=v1.3.0 WS_RELEASES_URL=http://127.0.0.1:22080 sh /tmp/x/install.sh"] =
		effecttest.Result{Err: errors.New("bad sum")}
	err := u.Apply(context.Background(), p)
	calls := strings.Join(m.Calls, "\n")
	if err == nil || !strings.Contains(calls, "rm -rf /tmp/x") || strings.Contains(calls, "exec") {
		t.Errorf("Apply = %v, calls %q; want an error, the directory removed, no exec", err, calls)
	}
}
