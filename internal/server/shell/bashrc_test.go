package shell

import (
	"context"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// fakeTmux runs new-session's pane on the terminal after its line %s sets TMUX or not, and prints
// "screen" for capture-pane. Run by sh, it never reads BASH_ENV. It refuses a second session.
const fakeTmux = `#!/bin/sh
for a do [ "$a" != capture-pane ] || { echo screen; exit; }; done
[ -z "${FAKE_TMUX-}" ] || { echo recursed; exit 99; }; export FAKE_TMUX=1
while [ $# -gt 0 ] && [ "$1" != new-session ]; do shift; done; shift
n=0; for a do [ "$a" = ';' ] && break; n=$((n + 1)); done
i=0; for a do shift; [ "$i" -lt "$n" ] && set -- "$@" "$a"; i=$((i + 1)); done
%s
exec "$@" >/dev/tty
`

// login runs bash -c command as sshd would, with ~/.bashrc as BASH_ENV, in script(1)'s terminal
// where tty is true. It returns the status and output within 20 s. fakes are scripts on the PATH.
func login(t *testing.T, fakes map[string]string, tty bool, env []string, command string) (int, string) {
	dir := t.TempDir()
	for _, name := range []string{"bash", "mktemp", "mkdir", "rm", "cat"} {
		path, err := exec.LookPath(name)
		if err != nil {
			t.Skip(err)
		}
		if _, fake := fakes[name]; !fake {
			if err := os.Symlink(path, filepath.Join(dir, name)); err != nil {
				t.Fatal(err)
			}
		}
	}
	rc, err := os.ReadFile("testdata/bashrc")
	if err != nil {
		t.Fatal(err)
	}
	files := map[string]string{"bashrc": strings.ReplaceAll(string(rc), "/usr/local/lib/ws/zed/tmux.conf", "/dev/null")}
	maps.Copy(files, fakes)
	for name, text := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0o755); err != nil { //nolint:gosec // the test's own scripts
			t.Fatal(err)
		}
	}
	name, args := filepath.Join(dir, "bash"), []string{"-c", command}
	if tty {
		name, args = "script", []string{"-qec", "bash -c '" + command + "'", "/dev/null"}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...) //nolint:gosec // the test's own command
	cmd.Env = append([]string{"PATH=" + dir, "HOME=" + dir, "TMPDIR=" + dir, "BASH_ENV=" + filepath.Join(dir, "bashrc"),
		"SHELL=/bin/sh", "TERM=dumb"}, env...)
	cmd.WaitDelay = time.Second
	out, err := cmd.CombinedOutput()
	if err != nil && cmd.ProcessState == nil {
		t.Fatal(err)
	}
	return cmd.ProcessState.ExitCode(), strings.TrimSpace(strings.ReplaceAll(string(out), "\r", ""))
}

// The wrapper of ssh's logins runs a login with a terminal in tmux and prints its screen after, any
// other as before. It never loses the command, shuts ssh out or starts tmux inside its own.
func TestBashrcRunsALoginWithATerminalInTmux(t *testing.T) {
	if _, err := exec.LookPath("script"); err != nil || runtime.GOOS != "linux" {
		t.Skip("no script(1) on this machine, or not Linux")
	}
	const command, screen = `echo ${TMUX:-plain}; exit 7`, "\nscreen\x1b[m"
	ssh, tmux := []string{"SSH_TTY=/dev/pts/9"}, map[string]string{"tmux": strings.Replace(fakeTmux, "%s", "export TMUX=fake", 1)}
	for _, c := range []struct {
		name, pre, want string
		fakes           map[string]string
		tty             bool
		env             []string
	}{
		{"a terminal", "", "fake" + screen, tmux, true, ssh},
		{"a tmux that gives no TMUX", "", "plain" + screen, map[string]string{"tmux": strings.Replace(fakeTmux, "%s", "unset TMUX", 1)}, true, ssh},
		{"a tmux that fails", "", "plain", map[string]string{"tmux": "#!/bin/sh\nexit 1\n"}, true, ssh},
		{"no tmux", "", "plain", nil, true, ssh},
		{"a full /tmp", "", "plain", map[string]string{"tmux": tmux["tmux"], "mktemp": "#!/bin/sh\nmkdir -p \"$TMPDIR/d/command\" && echo \"$TMPDIR/d\"\n"}, true, ssh},
		{"Zed's terminal", ": /usr/local/lib/ws/zed/terminal; ", "plain", tmux, true, ssh},
		{"LC_WS_BAR=off", "", "plain", tmux, true, append(ssh, "LC_WS_BAR=off")},
		{"a login in tmux", "", "outer", tmux, true, append(ssh, "TMUX=outer")},
		{"a login in ws-bar's tmux", "", "plain", tmux, true, append(ssh, "WS_BAR_LOGIN=1")},
		{"no terminal, as Zed's server", "", "plain", tmux, false, ssh},
		{"no SSH_TTY", "", "plain", tmux, true, nil},
	} {
		if status, out := login(t, c.fakes, c.tty, c.env, c.pre+command); status != 7 || out != c.want {
			t.Errorf("%s: status %d, output %q; want 7 and %q", c.name, status, out, c.want)
		}
	}
}
