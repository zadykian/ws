package account_test

import (
	"context"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/resource/account"
)

const rootLine = "env -i -C /root HOME=/root " + account.SystemPath + " git clone -- a b"

func TestUserRunsAsRootFromAClearedEnv(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[rootLine] = effecttest.Result{Out: []byte("ok")}
	root := account.User{Name: "root", Home: "/root"}
	if err := root.Pass(context.Background(), m, "git", "clone", "--", "a", "b"); err != nil {
		t.Fatal(err)
	}
	out, err := root.Output(context.Background(), m, "git", "clone", "--", "a", "b")
	if err != nil || string(out) != "ok" {
		t.Errorf("output %q, %v", out, err)
	}
	if want := []string{"run " + rootLine, "run " + rootLine}; !slices.Equal(m.Calls, want) {
		t.Errorf("calls %q, want %q", m.Calls, want)
	}
}

func TestUserPassesStdin(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["env -i -C /root HOME=/root "+account.SystemPath+" bash"] = effecttest.Result{}
	err := account.User{Name: "root", Home: "/root"}.PassInput(context.Background(), m, []byte("echo"), "bash")
	if err != nil || !slices.Equal(m.Inputs, []string{"echo"}) {
		t.Errorf("err %v, inputs %q", err, m.Inputs)
	}
}

// Another user's name fails, and no command runs: ws runs commands as root alone.
func TestUserOtherThanRootFails(t *testing.T) {
	m := effecttest.NewMachine()
	ws := account.User{Name: "ws", Home: "/home/ws"}
	ctx := context.Background()
	if err := ws.Pass(ctx, m, "git"); err == nil {
		t.Error("Pass ran as ws")
	}
	if _, err := ws.Output(ctx, m, "git"); err == nil {
		t.Error("Output ran as ws")
	}
	if err := ws.PassInput(ctx, m, nil, "git"); err == nil {
		t.Error("PassInput ran as ws")
	}
	if len(m.Calls) != 0 {
		t.Errorf("calls %q, want none", m.Calls)
	}
}

// A command sees none of ws's variables, as one may hold a secret input's value. The test runs the
// command line that User built, with root's home swapped for a directory of the test's, and a
// variable set in the caller's env.
func TestUserCommandGetsNoneOfTheCallersEnv(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("env -C is GNU's")
	}
	home := t.TempDir()
	probe := home + "/probe"
	script := "#!/bin/sh\n[ -z \"${WS_CI_SHARED+set}\" ] && [ \"$HOME\" = \"" + home + "\" ] && [ \"$PWD\" = \"" + home + "\" ]\n"
	if err := os.WriteFile(probe, []byte(script), 0o700); err != nil { //nolint:gosec // the probe must run
		t.Fatal(err)
	}
	callerEnv := []string{"WS_CI_SHARED=x", "HOME=/root", "PATH=/usr/bin:/bin"}
	direct := exec.Command(probe) //nolint:gosec // the test wrote the probe in its own directory
	direct.Env = callerEnv
	if direct.Run() == nil {
		t.Fatal("the probe passes with the variable set, so it proves nothing")
	}
	m := effecttest.NewMachine()
	m.Results["env -i -C /root HOME=/root "+account.SystemPath+" "+probe] = effecttest.Result{}
	if err := (account.User{Name: "root", Home: "/root"}).Pass(context.Background(), m, probe); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(m.Calls[0], "run env -i -C /root HOME=/root "+account.SystemPath+" ") {
		t.Fatalf("the call is %q, want root's command from a cleared env", m.Calls[0])
	}
	line := strings.NewReplacer("-C /root", "-C "+home, "HOME=/root", "HOME="+home).Replace(strings.TrimPrefix(m.Calls[0], "run "))
	argv := strings.Fields(line)
	cleared := exec.Command(argv[0], argv[1:]...) //nolint:gosec // the line that User built for the probe
	cleared.Env = callerEnv
	if out, err := cleared.CombinedOutput(); err != nil {
		t.Errorf("the command saw the caller's env: %v %s", err, out)
	}
}
