package claude_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/kind"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/server/values"
)

const (
	self       = "/usr/local/bin/ws"
	systemPath = "PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
)

// specs is claude's spec, git's, whose tokens claude's units follow, and the frame's, whose unit
// git's chain waits on, as kind.Load finds them.
func specs(t *testing.T) fstest.MapFS {
	fsys := fstest.MapFS{}
	for _, name := range []string{"claude/units.toml.tmpl", "git/units.toml.tmpl", "git/files/ws.gitconfig.tmpl",
		"git/files/allowed_signers.tmpl", "frame/units.toml.tmpl", "frame/files/values.yml.tmpl", "claude/files/managed-settings.json"} {
		text, err := os.ReadFile(filepath.Join("..", name)) //nolint:gosec // a file of the roles, named above
		if err != nil {
			t.Fatal(err)
		}
		fsys[name] = &fstest.MapFile{Data: text}
	}
	return fsys
}

// machine is a machine with root's home, and its env for the values of an empty file.
func machine(t *testing.T) (*effecttest.Machine, kind.Env) {
	v, err := values.Parse(nil)
	if err != nil {
		t.Fatal(err)
	}
	m := effecttest.NewMachine()
	m.Results["getent passwd root"] = effecttest.Result{Out: []byte("root:x:0:0:root:/root:/bin/bash\n")}
	m.Dirs["/root"] = 0o750
	return m, kind.Env{M: m, U: &effecttest.Units{}, Out: &effecttest.Out{}, Data: v, Self: self}
}

// units loads the specs on e, and keeps claude's.
func units(t *testing.T, e kind.Env) []kind.Unit {
	all, err := kind.Load(e, specs(t))
	if err != nil {
		t.Fatal(err)
	}
	return slices.DeleteFunc(all, func(u kind.Unit) bool {
		return !strings.HasPrefix(u.Name, "state/claude-") && !strings.HasPrefix(u.Name, "file/claude-")
	})
}

// unit returns claude's unit of the name.
func unit(t *testing.T, e kind.Env, name string) kind.Unit {
	us := units(t, e)
	i := slices.IndexFunc(us, func(u kind.Unit) bool { return u.Name == name })
	if i < 0 {
		t.Fatalf("no unit %s in %q", name, kind.Names(us))
	}
	return us[i]
}

// changes returns the headlines of the unit's plan.
func changes(t *testing.T, u kind.Unit) []string {
	p, err := u.Step.Plan(t.Context())
	if err != nil {
		t.Fatalf("plan %s: %v", u.Name, err)
	}
	var got []string
	for _, c := range p.Changes {
		got = append(got, step.Headline(c))
	}
	return got
}

// apply plans the unit and applies its changes.
func apply(t *testing.T, u kind.Unit) {
	p, err := u.Step.Plan(t.Context())
	if err != nil {
		t.Fatalf("plan %s: %v", u.Name, err)
	}
	if err := u.Step.Apply(t.Context(), p); err != nil {
		t.Fatalf("apply %s: %v", u.Name, err)
	}
}

func TestClaudeUnitsChainInTheStepsOrder(t *testing.T) {
	_, e := machine(t)
	want := []string{"state/claude-install", "state/claude-cld-install", "state/claude-update",
		"state/claude-completion", "state/claude-settings", "file/claude-managed-settings", "state/claude-telemetry",
		"state/claude-marketplaces",
		"state/claude-linger", "state/claude-restore"}
	if got := kind.Names(units(t, e)); !slices.Equal(got, want) {
		t.Errorf("order = %q, want %q", got, want)
	}
	all, err := kind.Load(e, specs(t))
	if err != nil {
		t.Fatal(err)
	}
	names := kind.Names(all)
	if slices.Index(names, "state/git-tokens") > slices.Index(names, "state/claude-install") {
		t.Errorf("order = %q, want git's tokens before claude's install", names)
	}
}

// installer is the line by which root runs an installer from url, with env and shell, and with
// ~/.local/bin first on its PATH. root's needs no switch of user.
func installer(url, env, shell string) string {
	script := `script=$(curl -fsSL --retry 3 "$1") && printf "%s\n" "$script" | env "$2" "$3"`
	path := "PATH=/root/.local/bin:" + strings.TrimPrefix(systemPath, "PATH=")
	return "env HOME=/root env -u XDG_RUNTIME_DIR -u DBUS_SESSION_BUS_ADDRESS " + path + " sh -c " + script + " sh " + url + " " + env + " " + shell
}

func TestClaudeInstallersRunAsRootOnce(t *testing.T) {
	for _, c := range []struct{ unit, url, env, shell, creates string }{
		{"state/claude-install", "https://claude.ai/install.sh", "CLAUDE_INSTALL_ALLOW_SUDO=1", "bash",
			"/root/.local/bin/claude"},
		{"state/claude-cld-install", "https://github.com/zadykian/cld/releases/latest/download/install.sh",
			"CLD_INSTALL_DIR=/root/.local/bin", "sh", "/root/.local/bin/cld"},
	} {
		m, e := machine(t)
		line := installer(c.url, c.env, c.shell)
		m.Results[line] = effecttest.Result{}
		apply(t, unit(t, e, c.unit))
		if !slices.ContainsFunc(m.Calls, func(call string) bool { return strings.HasSuffix(call, " "+line) }) {
			t.Errorf("%s: calls %q, want the installer as root", c.unit, m.Calls)
		}
		m.Files[c.creates] = nil
		if got := changes(t, unit(t, e, c.unit)); got != nil {
			t.Errorf("%s: second plan = %q, want none", c.unit, got)
		}
	}
}

// installScript is the script of claude's installer units, as units.toml.tmpl gives it.
func installScript(t *testing.T) string {
	text, err := os.ReadFile("units.toml.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	m := regexp.MustCompile(`"-c", '([^']*)'`).FindSubmatch(text)
	if m == nil {
		t.Fatal("units.toml.tmpl holds no installer's script")
	}
	return string(m[1])
}

// runInstaller runs the installers' script with a curl that runs curlBody, and returns what the
// shell it pipes the script into printed.
func runInstaller(t *testing.T, curlBody string) (string, error) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "curl"), []byte("#!/bin/sh\n"+curlBody+"\n"), 0o755); err != nil { //nolint:gosec // G306: the test's own curl runs
		t.Fatal(err)
	}
	cmd := exec.Command("sh", "-c", installScript(t), "sh", "https://example.test/install.sh", "WS_TEST=given", "sh") //nolint:gosec // G204: the role's own script
	cmd.Env = []string{"PATH=" + dir + ":/usr/bin:/bin"}
	out, err := cmd.Output()
	return string(out), err
}

func TestClaudeInstallerRunsTheFetchedScriptWithItsEnv(t *testing.T) {
	out, err := runInstaller(t, `echo 'echo "ran with $WS_TEST"'`)
	if err != nil || out != "ran with given\n" {
		t.Errorf("out %q, %v; want the script run with the env", out, err)
	}
}

func TestClaudeInstallerRunsNothingWhereCurlFails(t *testing.T) {
	out, err := runInstaller(t, `echo 'echo partial'; exit 22`)
	if err == nil || out != "" {
		t.Errorf("out %q, %v; want a failure and no part of the script run", out, err)
	}
}

func TestClaudeCommandsRunAsRoot(t *testing.T) {
	m, e := machine(t)
	names := []string{"update", "completion", "settings", "telemetry", "marketplaces"}
	for _, name := range names {
		line := "env HOME=/root env -u XDG_RUNTIME_DIR -u DBUS_SESSION_BUS_ADDRESS " + systemPath + " " + self + " server claude " + name
		m.Results[line+" --check"] = effecttest.Result{Err: os.ErrInvalid}
		m.Results[line] = effecttest.Result{}
		apply(t, unit(t, e, "state/claude-"+name))
	}
	var ran []string
	for _, c := range m.Calls {
		if strings.HasPrefix(c, "run env HOME=/root") && !strings.HasSuffix(c, "--check") {
			ran = append(ran, c[strings.LastIndex(c, " ")+1:])
		}
	}
	if !slices.Equal(ran, names) {
		t.Errorf("ran %q, want %q as root; calls %q", ran, names, m.Calls)
	}
}

func TestClaudeLingerIsRootsAndRestoreRunsAsRoot(t *testing.T) {
	m, e := machine(t)
	if got := changes(t, unit(t, e, "state/claude-linger")); !slices.Equal(got, []string{"enable lingering for root", "start user@0.service"}) {
		t.Errorf("linger plan = %q", got)
	}
	line := "env HOME=/root env -u XDG_RUNTIME_DIR -u DBUS_SESSION_BUS_ADDRESS " +
		systemPath + " " + self + " server claude restore"
	m.Results[line+" --check"] = effecttest.Result{Err: os.ErrInvalid}
	m.Results[line] = effecttest.Result{}
	apply(t, unit(t, e, "state/claude-restore"))
	if !slices.Contains(m.Calls, "run "+line) {
		t.Errorf("calls %q, want restore run as root", m.Calls)
	}
}

// managed is claude's managed file on a Linux machine.
const managed = "/etc/claude-code/managed-settings.json"

func TestClaudeManagedSettingsTurnOffEachAttribution(t *testing.T) {
	m, e := machine(t)
	apply(t, unit(t, e, "file/claude-managed-settings"))
	var got map[string]map[string]any
	if err := json.Unmarshal(m.Files[managed], &got); err != nil {
		t.Fatalf("the file %q is no JSON object: %v", m.Files[managed], err)
	}
	want := map[string]map[string]any{"attribution": {"commit": "", "pr": "", "sessionUrl": false}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("settings = %v, want %v", got, want)
	}
	if mode, owner := m.Modes[managed], m.Owners[managed]; mode != 0o644 || owner != [2]int{} {
		t.Errorf("file mode %o, owner %v; want root's at 644", mode, owner)
	}
	if mode, owner := m.Dirs["/etc/claude-code"], m.Owners["/etc/claude-code"]; mode != 0o755 || owner != [2]int{} {
		t.Errorf("directory mode %o, owner %v; want root's at 755", mode, owner)
	}
	if got := changes(t, unit(t, e, "file/claude-managed-settings")); got != nil {
		t.Errorf("second plan = %q, want none", got)
	}
}

func TestClaudeManagedSettingsReplaceWhatDiffers(t *testing.T) {
	m, e := machine(t)
	m.Dirs["/etc"], m.Dirs["/etc/claude-code"] = 0o755, 0o755
	m.Files[managed] = []byte(`{"attribution":{"commit":"x"}}`)
	if got := changes(t, unit(t, e, "file/claude-managed-settings")); !slices.Equal(got, []string{"write " + managed}) {
		t.Errorf("plan = %q, want a write of the file alone", got)
	}
}
