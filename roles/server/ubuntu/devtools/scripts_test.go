package devtools_test

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/pelletier/go-toml/v2"
)

// run is a run state's fields, as units.toml.tmpl gives them before a render.
type run struct {
	User     string   `toml:"user"`
	Unbooted bool     `toml:"unbooted"`
	Command  []string `toml:"command"`
	Check    []string `toml:"check"`
	Creates  []string `toml:"creates"`
}

// readRuns returns the run states of units.toml.tmpl by name.
func readRuns(t *testing.T) map[string]run {
	text, err := os.ReadFile("units.toml.tmpl")
	if err != nil {
		t.Fatal(err)
	}
	var u struct {
		State map[string]struct {
			Run run `toml:"run"`
		} `toml:"state"`
	}
	if err := toml.Unmarshal(text, &u); err != nil {
		t.Fatal(err)
	}
	runs := map[string]run{}
	for name, s := range u.State {
		runs[name] = s.Run
	}
	return runs
}

// script runs argv's shell line, argv[:4], with arg as its $1, and the fake commands of bin first
// on PATH. It returns the output, and whether the line exited 0.
func script(t *testing.T, argv []string, bin, arg string) (string, bool) {
	if len(argv) < 4 || argv[0] != "sh" || argv[1] != "-c" {
		t.Fatalf("%q is not a shell line with arguments", argv)
	}
	cmd := exec.Command("sh", "-c", argv[2], argv[3], arg) //nolint:gosec // G204: the role's own line
	cmd.Env = []string{"PATH=" + bin + ":/usr/bin:/bin"}
	out, err := cmd.CombinedOutput()
	return string(out), err == nil
}

// fake writes a command of the name into bin, whose body is a shell script.
func fake(t *testing.T, bin, name, body string) {
	if err := os.WriteFile(filepath.Join(bin, name), []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil { //nolint:gosec // G306: the test's own fake runs
		t.Fatal(err)
	}
}

// templateAction matches an action of the spec's template, the one way a value reaches a command's
// text. A word of a command's own, as the guard's message, is no value.
var templateAction = regexp.MustCompile(`\{\{.*?\}\}`)

// Each run is WS_USER's, but the token file's group, which root alone may change, and runs without
// booted systemd too. No secret reaches a command's text, which a run's headline and mise's log show.
func TestDevtoolsRunsAreTheAccounts(t *testing.T) {
	runs := readRuns(t)
	want := []string{"devtools-gh-login", "devtools-gh-telemetry", "devtools-gh-token-group", "devtools-npm",
		"devtools-roslyn", "devtools-rustup"}
	var got []string
	for name, r := range runs {
		got = append(got, name)
		user := "{{ .User }}"
		if name == "devtools-gh-token-group" {
			user = "root"
		}
		if r.User != user || !r.Unbooted {
			t.Errorf("%s: user %q, unbooted %v; want %s, unbooted", name, r.User, r.Unbooted, user)
		}
		for _, action := range templateAction.FindAllString(strings.Join(slices.Concat(r.Command, r.Check), " "), -1) {
			if strings.Contains(action, "secrets") || strings.Contains(action, "Token") {
				t.Errorf("%s: %q reads the secrets", name, action)
			}
		}
	}
	if slices.Sort(got); !slices.Equal(got, want) {
		t.Errorf("runs %q, want %q", got, want)
	}
}

func TestTheLockfilePinsEachPackageOfPackageJSON(t *testing.T) {
	var pkg struct{ Dependencies map[string]string }
	var lock struct {
		LockfileVersion int
		Packages        map[string]struct{ Version, Integrity, Resolved string }
	}
	for name, into := range map[string]any{"files/npm/package.json": &pkg, "files/npm/package-lock.json": &lock} {
		data, err := os.ReadFile(name) //nolint:gosec // a file of the role, named above
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(data, into); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
	}
	if len(pkg.Dependencies) != 4 || lock.LockfileVersion != 3 {
		t.Fatalf("%d dependencies, lockfileVersion %d", len(pkg.Dependencies), lock.LockfileVersion)
	}
	for name, version := range pkg.Dependencies {
		if strings.ContainsAny(version, "^~<>=*| ") || version == "" {
			t.Errorf("%s: %q is not an exact version", name, version)
		}
		got := lock.Packages["node_modules/"+name]
		if got.Version != version || !strings.HasPrefix(got.Integrity, "sha512-") || got.Resolved == "" {
			t.Errorf("%s: lock has %+v, want version %s with integrity and resolved", name, got, version)
		}
	}
}

// npm ci's stamp holds both files' sums: its check fails where either changes, or node_modules
// goes, and a failed npm ci leaves no stamp.
func TestNpmStampCoversBothFiles(t *testing.T) {
	r := readRuns(t)["devtools-npm"]
	bin, dir := t.TempDir(), t.TempDir()
	fake(t, bin, "npm", `[ "$1" = ci ] && rm -rf node_modules && mkdir node_modules && [ ! -e fail ]`)
	for _, name := range []string{"package.json", "package-lock.json"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(name+"\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if _, ok := script(t, r.Check, bin, dir); ok {
		t.Error("the check passed before npm ci")
	}
	if out, ok := script(t, r.Command, bin, dir); !ok {
		t.Fatalf("npm ci: %s", out)
	}
	if _, ok := script(t, r.Check, bin, dir); !ok {
		t.Error("the check failed after npm ci")
	}
	for _, change := range []func() error{
		func() error { return os.WriteFile(filepath.Join(dir, "package.json"), []byte("bumped\n"), 0o600) },
		func() error { return os.WriteFile(filepath.Join(dir, "package-lock.json"), []byte("bumped\n"), 0o600) },
		func() error { return os.RemoveAll(filepath.Join(dir, "node_modules")) },
	} {
		if _, ok := script(t, r.Command, bin, dir); !ok {
			t.Fatal("npm ci failed")
		}
		if err := change(); err != nil {
			t.Fatal(err)
		}
		if _, ok := script(t, r.Check, bin, dir); ok {
			t.Error("the check passed after a change")
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "fail"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, ok := script(t, r.Command, bin, dir); ok {
		t.Error("a failed npm ci passed")
	}
	if _, err := os.Stat(filepath.Join(dir, "node_modules/.ws-sha256")); err == nil {
		t.Error("a failed npm ci left a stamp")
	}
}

// gh's login takes the token on its input, never in its arguments; its check compares gh's token
// with the file's, and passes without the file.
func TestGHLoginPassesTheTokenOnItsInput(t *testing.T) {
	r := readRuns(t)["devtools-gh-login"]
	bin, dir := t.TempDir(), t.TempDir()
	held, args := filepath.Join(dir, "held"), filepath.Join(dir, "args")
	fake(t, bin, "gh", `echo "$@" >>`+args+`
case $1-$2 in
auth-login) cat >`+held+` ;;
auth-token) cat `+held+` ;;
esac`)
	tokenFile := filepath.Join(dir, "gh-token")
	if _, ok := script(t, r.Check, bin, tokenFile); !ok {
		t.Error("the check failed without the token file")
	}
	if err := os.WriteFile(tokenFile, []byte("# Written by github.com/zadykian/ws\n# a comment\ndummy-a\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, ok := script(t, r.Check, bin, tokenFile); ok {
		t.Error("the check passed before a login")
	}
	if out, ok := script(t, r.Command, bin, tokenFile); !ok {
		t.Fatalf("login: %s", out)
	}
	if got, _ := os.ReadFile(held); string(got) != "dummy-a\n" { //nolint:gosec // the test's own file
		t.Errorf("gh got %q on its input, want the last line", got)
	}
	if _, ok := script(t, r.Check, bin, tokenFile); !ok {
		t.Error("the check failed after the login")
	}
	if err := os.WriteFile(held, []byte("dummy-b\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, ok := script(t, r.Check, bin, tokenFile); ok {
		t.Error("the check passed with another token held")
	}
	if got, _ := os.ReadFile(args); strings.Contains(string(got), "dummy") { //nolint:gosec // the test's own file
		t.Errorf("gh's arguments %q hold the token", got)
	}
}

func TestGHTelemetryCheckWantsDisabled(t *testing.T) {
	r := readRuns(t)["devtools-gh-telemetry"]
	if !slices.Equal(r.Command, []string{"env", "GH_TELEMETRY=0", "gh", "config", "set", "telemetry", "disabled"}) {
		t.Errorf("command %q", r.Command)
	}
	bin := t.TempDir()
	for answer, pass := range map[string]bool{"disabled": true, "enabled": false} {
		fake(t, bin, "gh", `[ "$GH_TELEMETRY" = 0 ] && echo `+answer)
		if _, ok := script(t, append(r.Check, "sh", ""), bin, ""); ok != pass {
			t.Errorf("gh says %s: the check passes %v, want %v", answer, ok, pass)
		}
	}
}

// roslyn's check reads the store's one directory, which NuGet names by the version in lower case.
func TestRoslynCheckNamesTheCommandsVersion(t *testing.T) {
	r := readRuns(t)["devtools-roslyn"]
	i := slices.Index(r.Command, "--version")
	if i < 0 || len(r.Check) != 6 || r.Check[5] != r.Command[i+1] || r.Check[5] != strings.ToLower(r.Check[5]) {
		t.Fatalf("command %q and check %q name another version", r.Command, r.Check)
	}
	if r.Check[4] != "{{ .Home }}/.dotnet/tools/.store/roslyn-language-server" {
		t.Errorf("the check reads %q", r.Check[4])
	}
	store := t.TempDir()
	check := func() bool {
		cmd := exec.Command(r.Check[0], r.Check[1], r.Check[2], r.Check[3], store, r.Check[5]) //nolint:gosec // G204: the role's own line
		cmd.Env = []string{"PATH=/usr/bin:/bin"}
		return cmd.Run() == nil
	}
	if check() {
		t.Error("an empty store passed")
	}
	for _, v := range []string{r.Check[5], "5.11.0"} {
		if err := os.Mkdir(filepath.Join(store, v), 0o750); err != nil {
			t.Fatal(err)
		}
		if got, want := check(), v == r.Check[5]; got != want {
			t.Errorf("with %s: the check passes %v, want %v", v, got, want)
		}
	}
}

func TestRustupCreatesItsBinary(t *testing.T) {
	r := readRuns(t)["devtools-rustup"]
	if !slices.Equal(r.Creates, []string{"{{ .Home }}/.cargo/bin/rustup"}) || len(r.Check) != 0 {
		t.Errorf("creates %q, check %q", r.Creates, r.Check)
	}
}
