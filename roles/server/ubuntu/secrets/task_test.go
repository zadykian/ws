package secrets_test

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// runLine and addHint are the words of the task's refusals that a caller acts on.
const (
	runLine = "sudo env MISE_LOCKED=1 mise -C /opt/ws/roles/server run secrets"
	addHint = "ws client secrets add NAME"
	hostKey = "/etc/ssh/ssh_host_ed25519_key"
)

// fakes stand in for the commands of the task that would reach the machine, GitHub or a key: each
// logs how the task called it under FAKE_LOG. fnox runs what follows its --, as fnox exec does,
// but exits FAKE_DECRYPT for the task's first exec, of true.
var fakes = map[string]string{
	"id":       `echo "${FAKE_UID:-0}"`,
	"hostname": `echo "${FAKE_HOST-ws-a}"`,
	"mise": `echo "$*" >>"$FAKE_LOG/mise"
echo "umask=$(umask) git=${GIT_SSH_COMMAND+s}${GIT_CONFIG_GLOBAL+g}${GIT_CONFIG_NOSYSTEM+n}${GIT_TERMINAL_PROMPT+p}" >>"$FAKE_LOG/mise-env"
exit "${FAKE_STATUS:-0}"`,
	"ssh": `echo "$*" >>"$FAKE_LOG/ssh"
echo 'git@github.com: Permission denied (publickey).' >&2
exit 255`,
	"fnox": `{
	echo "cwd=$PWD"
	echo "argv=$*"
	echo "key=${FNOX_AGE_KEY_FILE-}"
	echo "age_key=${FNOX_AGE_KEY+set}"
	echo "age_identity=${FNOX_AGE_IDENTITY+set}"
	echo "config_dir=${FNOX_CONFIG_DIR-}"
	echo "config=$(ls -A -- "$FNOX_CONFIG_DIR")"
	echo "daemon=${FNOX_DAEMON-}"
} >>"$FAKE_LOG/fnox"
while [ "$1" != -- ]; do shift; done
shift
[ "$1" != true ] || exit "${FAKE_DECRYPT:-0}"
exec "$@"`,
}

// fixture is a machine for the task under a temporary directory. A prefix stands for /, a bare
// repository of secrets is on main, a work tree pushes to it, and the fakes come first in PATH.
type fixture struct {
	t      *testing.T
	root   string
	prefix string
	origin string
	seed   string
	logs   string
	side   string
	env    map[string]string
}

func newFixture(t *testing.T) *fixture {
	if runtime.GOOS != "linux" {
		t.Skip("the task runs on a Linux server alone")
	}
	root := t.TempDir()
	side, err := filepath.Abs("../..")
	if err == nil {
		side, err = filepath.EvalSymlinks(side)
	}
	if err != nil {
		t.Fatal(err)
	}
	f := &fixture{
		t: t, root: root, prefix: filepath.Join(root, "prefix"), origin: filepath.Join(root, "origin.git"),
		seed: filepath.Join(root, "seed"), logs: filepath.Join(root, "logs"), side: side,
	}
	bin := filepath.Join(root, "bin")
	for _, d := range []string{bin, f.logs, filepath.Dir(f.prefix + hostKey), filepath.Join(root, "home"), filepath.Join(root, "tmp")} {
		if err := os.MkdirAll(d, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(f.prefix+hostKey, []byte("not a key\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for name, body := range fakes {
		if err := os.WriteFile(filepath.Join(bin, name), []byte("#!/bin/bash\n"+body+"\n"), 0o700); err != nil { //nolint:gosec // a fake that the task runs
			t.Fatal(err)
		}
	}
	f.git(root, "init", "-q", "--bare", "-b", "main", f.origin)
	f.git(root, "init", "-q", "-b", "main", f.seed)
	f.git(f.seed, "remote", "add", "origin", f.origin)
	f.push("fnox.toml", "root = true\n")
	f.push("fnox.ws-a.toml", "[providers.host]\ntype = \"age\"\n")
	f.env = map[string]string{
		"PATH": bin + ":/usr/bin:/bin", "HOME": filepath.Join(root, "home"), "TMPDIR": filepath.Join(root, "tmp"),
		"MISE_LOCKED":     "1",
		"WS_SECRETS_REPO": f.origin, "WS_SECRETS_PREFIX": f.prefix, "FAKE_LOG": f.logs,
	}
	return f
}

// git runs git in dir with an environment of its own, so that no config of the caller's applies.
func (f *fixture) git(dir string, args ...string) string {
	args = append([]string{"-c", "user.name=ws", "-c", "user.email=ws@example.invalid", "-c", "commit.gpgsign=false"}, args...)
	cmd := exec.CommandContext(f.t.Context(), "git", args...) //nolint:gosec // the test's own git arguments
	cmd.Dir = dir
	cmd.Env = []string{"PATH=/usr/bin:/bin", "HOME=" + f.root, "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1"}
	out, err := cmd.CombinedOutput()
	if err != nil {
		f.t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// push commits a file to origin's main.
func (f *fixture) push(name, text string) {
	if err := os.WriteFile(filepath.Join(f.seed, name), []byte(text), 0o600); err != nil {
		f.t.Fatal(err)
	}
	f.git(f.seed, "add", name)
	f.git(f.seed, "commit", "-q", "-m", name)
	f.git(f.seed, "push", "-q", "origin", "main")
}

// run runs the task with args and the fixture's environment alone, and returns its exit code and
// its output.
func (f *fixture) run(args ...string) (int, string) {
	task := filepath.Join(f.side, "mise-tasks/secrets")
	cmd := exec.CommandContext(f.t.Context(), "bash", append([]string{task}, args...)...) //nolint:gosec // the task under test
	cmd.Dir = f.root
	for k, v := range f.env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	out, err := cmd.CombinedOutput()
	var exit *exec.ExitError
	if err != nil && !errors.As(err, &exit) {
		f.t.Fatal(err)
	}
	return cmd.ProcessState.ExitCode(), string(out)
}

// log returns what a fake logged, empty where it never ran.
func (f *fixture) log(name string) string {
	text, err := os.ReadFile(filepath.Join(f.logs, name)) //nolint:gosec // the test's own log
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		f.t.Fatal(err)
	}
	return string(text)
}

func (f *fixture) clone() string {
	return filepath.Join(f.prefix, "var/lib/ws/secrets")
}

// cloned fails the test unless the clone is root's alone and at origin's main.
func (f *fixture) cloned() {
	info, err := os.Stat(f.clone())
	if err != nil {
		f.t.Fatal(err)
	}
	if info.Mode().Perm() != 0o700 {
		f.t.Errorf("the clone's mode is %v, want 0700", info.Mode().Perm())
	}
	if got, want := f.git(f.clone(), "rev-parse", "HEAD"), f.git(f.origin, "rev-parse", "main"); got != want {
		f.t.Errorf("the clone is at %s, want origin's main, %s", got, want)
	}
}

// The caller's key, identity and config of fnox's take no part: the task's own config directory,
// under its temporary one, is empty.
func TestTaskClonesAndRunsTheBootstrapUnderFnox(t *testing.T) {
	f := newFixture(t)
	f.env["FNOX_AGE_KEY"] = "AGE-SECRET-KEY-OF-THE-CALLER"
	f.env["FNOX_AGE_IDENTITY"] = filepath.Join(f.root, "identity")
	f.env["FNOX_CONFIG_DIR"] = filepath.Join(f.root, "home")
	if err := os.WriteFile(filepath.Join(f.root, "home/config.toml"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if code, out := f.run(); code != 0 {
		t.Fatalf("the task exits %d, want 0:\n%s", code, out)
	}
	f.cloned()
	bootstrap := "-C " + f.side + " -E ws,secrets bootstrap"
	fnox := f.log("fnox")
	for _, want := range []string{
		"cwd=" + f.clone(),
		"argv=-P ws-a --if-missing error exec -- true",
		"argv=-P ws-a --if-missing error exec -- mise " + bootstrap,
		"key=" + f.prefix + hostKey,
		"age_key=\n", "age_identity=\n", "config=\n", "daemon=off",
	} {
		if !strings.Contains(fnox, want) {
			t.Errorf("fnox's log lacks %q:\n%s", want, fnox)
		}
	}
	own := regexp.MustCompile(`(?m)^config_dir=` + regexp.QuoteMeta(filepath.Join(f.root, "tmp")) + `/[^/]+/fnox$`)
	if n := len(own.FindAllString(fnox, -1)); n != 2 {
		t.Errorf("fnox ran %d times in the task's own config directory, want 2:\n%s", n, fnox)
	}
	if got := f.log("mise"); got != bootstrap+"\n" {
		t.Errorf("mise ran %q, want %q", got, bootstrap)
	}
}

func TestTaskPassesTheBootstrapsArgumentsAndStatus(t *testing.T) {
	f := newFixture(t)
	f.env["FAKE_STATUS"] = "2"
	if code, out := f.run("plan", "--detailed-exitcode"); code != 2 {
		t.Fatalf("the task exits %d, want the bootstrap's 2:\n%s", code, out)
	}
	if got := f.log("mise"); !strings.HasSuffix(got, "bootstrap plan --detailed-exitcode\n") {
		t.Errorf("mise ran %q, want the task's arguments last", got)
	}
}

func TestTaskFastForwardsTheClone(t *testing.T) {
	f := newFixture(t)
	if code, out := f.run(); code != 0 {
		t.Fatalf("the first run exits %d:\n%s", code, out)
	}
	f.push("fnox.toml", "root = true\nif_missing = \"error\"\n")
	if code, out := f.run(); code != 0 {
		t.Fatalf("the second run exits %d:\n%s", code, out)
	}
	f.cloned()
}

// refused fails the test unless the task exited 1 with want in its output, and ran no bootstrap.
func (f *fixture) refused(code int, out, want string) {
	if code != 1 || !strings.Contains(out, want) {
		f.t.Errorf("the task exits %d, want 1 and %q:\n%s", code, want, out)
	}
	if got := f.log("mise"); got != "" {
		f.t.Errorf("mise ran %q, want no run", got)
	}
}

func TestTaskRefusesAUserButRoot(t *testing.T) {
	f := newFixture(t)
	f.env["FAKE_UID"] = "1000"
	code, out := f.run()
	f.refused(code, out, runLine)
	if _, err := os.Lstat(f.clone()); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("a refused user made the clone: %v", err)
	}
	if got := f.log("fnox"); got != "" {
		t.Errorf("fnox ran for a refused user:\n%s", got)
	}
}

func TestTaskRefusesARunWithoutMiseLocked(t *testing.T) {
	f := newFixture(t)
	delete(f.env, "MISE_LOCKED")
	code, out := f.run()
	f.refused(code, out, runLine)
	if _, err := os.Lstat(f.clone()); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("a run without MISE_LOCKED made the clone: %v", err)
	}
}

func TestTaskNamesAddWhereGitHubRefusesTheKey(t *testing.T) {
	f := newFixture(t)
	f.env["WS_SECRETS_REPO"] = "git@github.com:o/r.git"
	code, out := f.run()
	f.refused(code, out, addHint)
	ssh := f.log("ssh")
	for _, want := range []string{
		"-F /dev/null", "-i " + f.prefix + hostKey,
		"-o IdentitiesOnly=yes", "-o IdentityAgent=none", "-o BatchMode=yes",
		"-o UserKnownHostsFile=" + filepath.Join(f.side, "ubuntu/secrets/github_known_hosts"),
		"-o GlobalKnownHostsFile=/dev/null", "-o StrictHostKeyChecking=yes", "-o UpdateHostKeys=no",
	} {
		if !strings.Contains(ssh, want) {
			t.Errorf("ssh's arguments lack %q:\n%s", want, ssh)
		}
	}
}

func TestTaskNamesAddWithoutAProfile(t *testing.T) {
	f := newFixture(t)
	f.env["FAKE_HOST"] = "ws-b"
	code, out := f.run()
	f.refused(code, out, addHint)
}

func TestTaskNamesAddWhereTheKeyOpensNothing(t *testing.T) {
	f := newFixture(t)
	f.env["FAKE_DECRYPT"] = "1"
	code, out := f.run()
	f.refused(code, out, addHint)
}
