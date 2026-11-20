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

func TestUserRunsWithItsHome(t *testing.T) {
	cases := []struct {
		user account.User
		want string
	}{
		{account.User{Name: "root", Home: "/root"}, "run env HOME=/root git clone -- a b"},
		{account.User{Name: "ws", Home: "/home/ws"}, "root runuser -u ws -- env -i -C /home/ws HOME=/home/ws PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin git clone -- a b"},
	}
	for _, c := range cases {
		m := effecttest.NewMachine()
		line := c.want[len("root "):]
		if c.user.Name == "root" {
			line = c.want[len("run "):]
		}
		m.Results[line] = effecttest.Result{Out: []byte("ok")}
		if err := c.user.Pass(context.Background(), m, "git", "clone", "--", "a", "b"); err != nil {
			t.Fatal(err)
		}
		out, err := c.user.Output(context.Background(), m, "git", "clone", "--", "a", "b")
		if err != nil || string(out) != "ok" {
			t.Errorf("%s: output %q, %v", c.user.Name, out, err)
		}
		if want := []string{c.want, c.want}; !slices.Equal(m.Calls, want) {
			t.Errorf("%s: calls %q, want %q", c.user.Name, m.Calls, want)
		}
	}
}

func TestUserPassesStdin(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["runuser -u ws -- env -i -C /home/ws HOME=/home/ws "+account.SystemPath+" bash"] = effecttest.Result{}
	err := account.User{Name: "ws", Home: "/home/ws"}.PassInput(context.Background(), m, []byte("echo"), "bash")
	if err != nil || !slices.Equal(m.Inputs, []string{"echo"}) {
		t.Errorf("err %v, inputs %q", err, m.Inputs)
	}
}

// A user's command sees none of the caller's variables, as one may hold a secret's value. The test
// runs the command line after runuser's switch, which needs root, with such a variable set.
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
	m.Results["runuser -u ws -- env -i -C "+home+" HOME="+home+" "+account.SystemPath+" "+probe] = effecttest.Result{}
	if err := (account.User{Name: "ws", Home: home}).Pass(context.Background(), m, probe); err != nil {
		t.Fatal(err)
	}
	argv := strings.Fields(strings.TrimPrefix(m.Calls[0], "root runuser -u ws -- "))
	switched := exec.Command(argv[0], argv[1:]...) //nolint:gosec // the line that User built for the probe
	switched.Env = callerEnv
	if out, err := switched.CombinedOutput(); err != nil {
		t.Errorf("the user's command saw the caller's env: %v %s", err, out)
	}
}
