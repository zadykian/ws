package cli

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/cli/clitest"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/client/macos/secrets"
	"github.com/zadykian/ws/roles/client/macos/secrets/secretstest"
	"github.com/zadykian/ws/roles/client/values"
)

// The server lab, whose hostname is ws-a, and the lines that ws client secrets add lab runs.
const (
	labKey       = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAILab"
	olderKey     = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOld"
	fnoxKeyFile  = "/h/.config/fnox/age.txt"
	knownHosts   = "/h/.ssh/known_hosts"
	findLab      = "ssh-keygen -F lab -f " + knownHosts
	hostnameLine = "ssh lab hostname"
	runLine      = "ssh -t lab sudo env MISE_DATA_DIR=/usr/local/share/mise MISE_LOCKED=1 mise -C /opt/ws/roles/server " +
		"run secrets"
	keysLine    = secrets.GH + " api --paginate repos/o/n/keys"
	postLine    = secrets.GH + " api -X POST repos/o/n/keys -f title=ws-a -f key=" + labKey + " -F read_only=true"
	sealed      = "fnox-age-batch-v1:c2VhbGVk"
	fingerprint = "256 SHA256:Zm9yIHRoZSB0ZXN0 no comment (ED25519)"
)

// sharedText is fnox.toml as init leaves it, with one value.
const sharedText = `root = true
if_missing = "error"

[providers.age]
type = "age"

[providers.age.recipients]
nano = "age1yubikey1qnano"
backup = "age1qbackup"

[secrets]
S1 = { provider = "age", value = "` + sealed + `" }
`

// addFixture is lab on a Mac that ran init, with its fnox key file and lab's key in known_hosts. The
// secrets repository holds files, or fnox.toml alone where files is nil.
func addFixture(files map[string]string) (enginecli.Env, *secretstest.Mac) {
	m := effecttest.NewMachine()
	keyed(m)
	m.Files["/etc/ws/servers/lab/values.yml"] = []byte(values.Text("WS_USER", "ada", "WS_SLOT", "1"))
	m.Dirs["/etc/ws/servers/lab"] = 0o755
	m.Files[fnoxKeyFile] = []byte("AGE-PLUGIN-YUBIKEY-1QQQ\n")
	m.Files[knownHosts] = []byte("ws ssh-ed25519 AAAA\nlab " + labKey + "\n")
	m.Results[findLab] = effecttest.Result{Out: []byte("# Host lab found: line 2\nlab " + labKey + "\n")}
	m.Results[hostnameLine] = effecttest.Result{Out: []byte("ws-a\n")}
	m.Results[runLine] = effecttest.Result{}
	if files == nil {
		files = map[string]string{secrets.SharedFile: sharedText}
	}
	mac := secretstest.New(m, files)
	p := withEnv(fakeProcess(), map[string]string{"HOME": "/h"})
	p.Machine = mac
	return p, mac
}

// heldBy is the repository's files once key is ws-a's recipient, ws-a's profile holding a value.
func heldBy(t *testing.T, key string) map[string]string {
	f, _, err := secrets.SetHostKey(secrets.Files{secrets.SharedFile: []byte(sharedText)}, "ws-a", key)
	if err != nil {
		t.Fatal(err)
	}
	profile := secrets.ProfileFile("ws-a")
	return map[string]string{secrets.SharedFile: string(f[secrets.SharedFile]),
		profile: string(f[profile]) + `A1 = { provider = "host", value = "` + sealed + `" }` + "\n"}
}

// addWs runs ws client secrets add with stdin.
func addWs(p enginecli.Env, stdin string, args ...string) (status int, stdout, stderr string) {
	return clitest.Run(p, []*cobra.Command{Command(p)}, stdin,
		append([]string{"client", "secrets", "add"}, args...)...)
}

// callAt is the index of the first call that holds part, or -1.
func callAt(calls []string, part string) int {
	return slices.IndexFunc(calls, func(c string) bool { return strings.Contains(c, part) })
}

// cloned says whether the calls cloned the secrets repository.
func cloned(calls []string) bool { return callAt(calls, " git clone ") >= 0 }

func TestSecretsAddRunsItsStepsInOrder(t *testing.T) {
	p, mac := addFixture(nil)
	status, stdout, stderr := addWs(p, "", "lab")
	if status != 0 {
		t.Fatalf("add lab: status = %d, stderr = %q", status, stderr)
	}
	at := -1
	for _, part := range []string{findLab, hostnameLine, keysLine, " fnox reencrypt -p age -f", postLine,
		secrets.GH + " pr create", secrets.GH + " pr merge", runLine} {
		i := callAt(mac.Calls, part)
		if i <= at {
			t.Fatalf("%q comes at %d, after %d, in\n%q", part, i, at, mac.Calls)
		}
		at = i
	}
	want := secrets.Change{Topic: "add-ws-a", Title: "chore: add the host key of ws-a",
		Body: "Added by ws client secrets add lab.", Files: []string{"fnox.toml", "fnox.ws-a.toml"}}
	if landed := mac.Landed(); len(landed) != 1 || !equalChange(landed[0], want) {
		t.Errorf("add landed %+v, want %+v", landed, want)
	}
	if fnoxed := mac.Fnoxed(); len(fnoxed) != 1 {
		t.Errorf("add ran fnox %q, want one reencrypt of fnox.toml: the new profile holds no value", fnoxed)
	}
	if !strings.Contains(mac.Files()["fnox.ws-a.toml"], labKey) || !strings.Contains(stdout, secretstest.Pull) {
		t.Errorf("add left the profile %q and printed %q, want lab's key there and the link", mac.Files(), stdout)
	}
}

func equalChange(a, b secrets.Change) bool {
	return a.Topic == b.Topic && a.Title == b.Title && a.Body == b.Body && slices.Equal(a.Files, b.Files)
}

// scannable takes lab out of known_hosts, and has ssh reach it at 192.0.2.7:22022 by target.
func scannable(m *secretstest.Mac, target string) {
	delete(m.Machine.Files, knownHosts)
	m.Results["ssh -G "+target] = effecttest.Result{Out: []byte("user ada\nhostname 192.0.2.7\nport 22022\n")}
	m.Results["ssh-keyscan -t ed25519 -p 22022 192.0.2.7"] = effecttest.Result{
		Out: []byte("# 192.0.2.7:22022 SSH-2.0-OpenSSH\n[192.0.2.7]:22022 " + labKey + "\n")}
	m.Results["ssh-keygen -l -f -"] = effecttest.Result{Out: []byte(fingerprint + "\n")}
}

func TestSecretsAddScansAndRecordsTheKeyOnYes(t *testing.T) {
	p, mac := addFixture(nil)
	scannable(mac, "lab")
	status, _, stderr := addWs(p, "yes\n", "lab")
	if status != 0 {
		t.Fatalf("add lab: status = %d, stderr = %q", status, stderr)
	}
	if got := string(mac.Machine.Files[knownHosts]); got != "lab "+labKey+"\n" {
		t.Errorf("known_hosts holds %q, want lab's line", got)
	}
	if !strings.Contains(stderr, fingerprint) || !strings.Contains(stderr, "192.0.2.7:22022") {
		t.Errorf("stderr %q lacks the fingerprint and the address", stderr)
	}
}

func TestSecretsAddGoesDirectWithAnAddress(t *testing.T) {
	p, mac := addFixture(nil)
	target := "-o HostName=192.0.2.7 -o Port=22022 lab"
	scannable(mac, target)
	mac.Results["ssh "+target+" hostname"] = effecttest.Result{Out: []byte("ws-a\n")}
	run := strings.Replace(runLine, "-t lab", "-t "+target, 1)
	mac.Results[run] = effecttest.Result{}
	status, _, stderr := addWs(p, "yes\n", "lab", "--address", "192.0.2.7", "--port", "22022")
	if status != 0 {
		t.Fatalf("add with an address: status = %d, stderr = %q", status, stderr)
	}
	for _, line := range []string{"ssh -G " + target, "ssh-keyscan -t ed25519 -p 22022 192.0.2.7",
		"ssh " + target + " hostname", run} {
		if !slices.Contains(mac.Calls, "run "+line) {
			t.Errorf("calls lack %q:\n%q", line, mac.Calls)
		}
	}
}

func TestSecretsAddStopsWhereTheFingerprintDiffers(t *testing.T) {
	for _, stdin := range []string{"no\n", ""} {
		p, mac := addFixture(nil)
		scannable(mac, "lab")
		status, _, stderr := addWs(p, stdin, "lab")
		if status != 1 || !strings.Contains(stderr, "nothing changed") {
			t.Errorf("the answer %q: status = %d, stderr = %q", stdin, status, stderr)
		}
		if _, ok := mac.Machine.Files[knownHosts]; ok || cloned(mac.Calls) {
			t.Errorf("the answer %q wrote known_hosts or cloned: %q", stdin, mac.Calls)
		}
	}
}

func TestSecretsAddOpensNoPullRequestWhereNothingChanged(t *testing.T) {
	p, mac := addFixture(heldBy(t, labKey))
	mac.Keys = []secrets.DeployKey{{ID: 5, Title: "ws-a", Key: labKey, ReadOnly: true}}
	status, stdout, stderr := addWs(p, "", "lab")
	if status != 0 || !strings.Contains(stdout, "the secrets already hold the host key of ws-a") {
		t.Fatalf("add of a key in place: status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}
	if len(mac.Fnoxed()) != 0 || len(mac.Landed()) != 0 || callAt(mac.Calls, " -X ") >= 0 {
		t.Errorf("add of a key in place ran fnox, landed or changed a deploy key: %q", mac.Calls)
	}
	if !slices.Contains(mac.Calls, "run "+runLine) {
		t.Errorf("add of a key in place ran nothing on lab: %q", mac.Calls)
	}
}

func TestSecretsAddReplacesARebuiltServersKey(t *testing.T) {
	p, mac := addFixture(heldBy(t, olderKey))
	mac.Keys = []secrets.DeployKey{{ID: 7, Title: "ws-a", Key: olderKey, ReadOnly: true}}
	if status, _, stderr := addWs(p, "", "lab"); status != 0 {
		t.Fatalf("add of a rebuilt server: status = %d, stderr = %q", status, stderr)
	}
	deleted, posted := callAt(mac.Calls, secrets.GH+" api -X DELETE repos/o/n/keys/7"), callAt(mac.Calls, postLine)
	if deleted < 0 || posted < deleted {
		t.Errorf("the older deploy key goes at %d, the new one comes at %d, want both in that order", deleted, posted)
	}
	want := [][]string{{"reencrypt", "-p", "age", "-f"}, {"-P", "ws-a", "reencrypt", "-p", "host", "-f"}}
	if !slices.EqualFunc(mac.Fnoxed(), want, slices.Equal) {
		t.Errorf("add ran fnox %q, want %q", mac.Fnoxed(), want)
	}
	if landed := mac.Landed(); len(landed) != 1 || !slices.Equal(landed[0].Files, []string{"fnox.toml", "fnox.ws-a.toml"}) {
		t.Errorf("add landed %+v, want both files once", landed)
	}
	if files := mac.Files(); strings.Contains(files["fnox.toml"]+files["fnox.ws-a.toml"], olderKey) {
		t.Errorf("the older key stays in %q", files)
	}
}

func TestSecretsAddRefuses(t *testing.T) {
	for _, c := range []struct {
		name, want string
		args       []string
		set        func(p *enginecli.Env, m *secretstest.Mac)
	}{
		{"root", "not root", nil, func(p *enginecli.Env, _ *secretstest.Mac) { p.Euid = 0 }},
		{"a server flag", "the server's name alone", []string{"--server", "ws"}, nil},
		{"a bad port", "is not a port", []string{"--port", "70000"}, nil},
		{"no fnox key file", "secrets init first", nil,
			func(_ *enginecli.Env, m *secretstest.Mac) { delete(m.Machine.Files, fnoxKeyFile) }},
		{"a dotted hostname", "one label", nil, func(_ *enginecli.Env, m *secretstest.Mac) {
			m.Results[hostnameLine] = effecttest.Result{Out: []byte("ws.lan\n")}
		}},
		{"a label's hostname", "one label", nil, func(_ *enginecli.Env, m *secretstest.Mac) {
			m.Results[hostnameLine] = effecttest.Result{Out: []byte("nano\n")}
		}},
		{"a host key that ssh refuses", "drop the old one by ssh-keygen -R lab", nil,
			func(_ *enginecli.Env, m *secretstest.Mac) {
				m.Results[hostnameLine] = effecttest.Result{Err: errors.New("exit status 255")}
			}},
	} {
		p, mac := addFixture(nil)
		if c.set != nil {
			c.set(&p, mac)
		}
		status, _, stderr := addWs(p, "", append([]string{"lab"}, c.args...)...)
		touched := cloned(mac.Calls) || callAt(mac.Calls, keysLine) >= 0 || len(mac.Fnoxed()) > 0
		if status != 1 || !strings.Contains(stderr, c.want) || touched {
			t.Errorf("%s: status = %d, stderr = %q, secrets touched = %v; want 1 and %q", c.name, status, stderr,
				touched, c.want)
		}
	}
	for name, want := range map[string]string{"Lab": "not a server's name", "box": "no server named box"} {
		p, mac := addFixture(nil)
		if status, _, stderr := addWs(p, "", name); status != 1 || !strings.Contains(stderr, want) || cloned(mac.Calls) {
			t.Errorf("add %s: status = %d, stderr = %q; want 1 and %q", name, status, stderr, want)
		}
	}
}

// fnox prints a plain value, as where it passes and where it fails, and ws passes none of it on.
func TestSecretsAddKeepsTheValuesOutOfItsOutput(t *testing.T) {
	for _, fail := range []error{nil, errors.New("exit status 1")} {
		p, mac := addFixture(heldBy(t, olderKey))
		mac.FnoxErr = fail
		status, stdout, stderr := addWs(p, "", "lab")
		if (status != 0) != (fail != nil) || len(mac.Fnoxed()) == 0 {
			t.Fatalf("add lab, fnox failing with %v: status = %d, stderr = %q", fail, status, stderr)
		}
		for where, text := range map[string]string{"stdout": stdout, "stderr": stderr,
			"the calls": strings.Join(mac.Calls, "\n"), "the inputs": strings.Join(mac.Inputs, "\n")} {
			if strings.Contains(text, secretstest.Plain) || strings.Contains(text, "c2VhbGVk") {
				t.Errorf("fnox failing with %v: %s hold a value: %q", fail, where, text)
			}
		}
	}
}

func TestSecretsAddSaysToRunAgainWhereTheRunFails(t *testing.T) {
	p, mac := addFixture(nil)
	mac.Results[runLine] = effecttest.Result{Err: errors.New("exit status 255")}
	status, _, stderr := addWs(p, "", "lab")
	if status != 1 || !strings.Contains(stderr, "run ws client secrets add lab again") {
		t.Errorf("a failed run: status = %d, stderr = %q", status, stderr)
	}
}
