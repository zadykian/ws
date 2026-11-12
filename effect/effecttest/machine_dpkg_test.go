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

func TestMachineDpkgQueryNeedsATableAndAResultWins(t *testing.T) {
	m := effecttest.NewMachine()
	if _, err := query(m, "tmux"); err == nil {
		t.Error("a nil table answered")
	}
	m.Packages = effecttest.Packages{"tmux": {"3.5", "ii"}}
	m.Results["dpkg-query -W -f "+queryFormat+" tmux"] = effecttest.Result{Out: []byte("line")}
	if out, _ := query(m, "tmux"); string(out) != "line" {
		t.Errorf("out %q, want the Results line", out)
	}
}

func TestMachineDpkgQueryMatchesGlobsOncePerPackage(t *testing.T) {
	m := effecttest.NewMachine()
	m.Packages = effecttest.Packages{"linux-image-6.8": {"6.8", "ii"}, "linux-image-6.9": {"6.9", "rc"}, "tmux": {"3.5", "ii"}}
	out, err := query(m, "linux-image-*", "linux-image-6.8", "tmux")
	if err != nil {
		t.Fatal(err)
	}
	if want := "linux-image-6.8 6.8 ii \nlinux-image-6.9 6.9 rc \ntmux 3.5 ii \n"; string(out) != want {
		t.Errorf("out %q, want %q", out, want)
	}
	if _, err := query(m, "nothing-*"); err == nil {
		t.Error("a glob with no match ended with no error")
	}
}

func TestMachineDpkgQueryRefusesAnUnknownField(t *testing.T) {
	m := effecttest.NewMachine()
	m.Packages = effecttest.Packages{}
	if _, err := m.Run(context.Background(), "dpkg-query", "-W", "-f", "${Depends}", "tmux"); err == nil {
		t.Error("an unknown field was answered")
	}
}
