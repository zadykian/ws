package effecttest_test

import (
	"context"
	"errors"
	"os/exec"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

const queryFormat = "${Package} ${Version} ${db:Status-Abbrev}\n"

func query(m *effecttest.Machine, names ...string) ([]byte, error) {
	args := append([]string{"-W", "-f", queryFormat}, names...)
	return m.Run(context.Background(), "dpkg-query", args...)
}

func TestMachineAnswersDpkgQueryInNameOrder(t *testing.T) {
	m := effecttest.NewMachine()
	m.Packages = effecttest.Packages{"tmux": {"3.5", "ii"}, "curl": {"8.1", "rc"}}
	out, err := query(m, "tmux", "curl")
	if err != nil {
		t.Fatal(err)
	}
	if want := "curl 8.1 rc \ntmux 3.5 ii \n"; string(out) != want {
		t.Errorf("out %q, want %q", out, want)
	}
}

func TestMachineDpkgQueryPrintsTheRestAndExitsOne(t *testing.T) {
	m := effecttest.NewMachine()
	m.Packages = effecttest.Packages{"tmux": {"3.5", "ii"}}
	out, err := query(m, "tmux", "absent")
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 1 {
		t.Errorf("err %v, want exit 1", err)
	}
	if want := "tmux 3.5 ii \n"; string(out) != want {
		t.Errorf("out %q, want %q", out, want)
	}
}
