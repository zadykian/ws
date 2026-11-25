package secrets_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"filippo.io/age"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/macos/roles/secrets"
)

// The lines of init that the fixture's Mac runs: the list of open pull requests, fnox set of an
// input, and the first run's change.
const (
	pullsLine = secrets.GH + " pr list --repo o/n --state open --limit 100 --json headRefName,url --jq " +
		".[] | [.headRefName, .url] | @tsv"
	fnoxSet = "env -u FNOX_AGE_KEY -u FNOX_PROFILE PATH=/usr/local/bin:/usr/bin FNOX_AGE_KEY_FILE=" + keyFile +
		` sh -c cd "$1" && shift && exec "$@" sh ` + cloneDir + " fnox set "
	startTitle = "chore: set up the recipients and the inputs"
	startBody  = "The recipients nano and backup, and the inputs, each empty."
)

// initMac is a Mac whose tree names o/n, and whose server roles declare the inputs. Its Nano holds
// recipientB in slot 2, with the stub stubText. A change of title and body lands there.
func initMac(title, body string, inputs ...string) *effecttest.Machine {
	m := cloneable()
	m.Files[secrets.Tree+"/mise.toml"] = []byte("[env]\nWS_SECRETS_REPO = \"git@github.com:o/n.git\"\n")
	m.Dirs[roles], m.Dirs[roles+"/devtools"] = 0o755, 0o755
	var declared strings.Builder
	declared.WriteString("[bootstrap.secrets]\n")
	for _, name := range inputs {
		declared.WriteString(name + ` = { env = "` + name + `", allow_empty = true }` + "\n")
		m.Results[fnoxSet+name+" --provider age"] = effecttest.Result{}
	}
	m.Files[roles+"/devtools/"+secrets.InputsFile] = []byte(declared.String())
	m.Files["/usr/local/bin/fnox"], m.Files[secrets.Plugin] = []byte("fnox"), []byte("plugin")
	m.Results[listLine] = effecttest.Result{Out: []byte(twoSlots)}
	m.Results[identityLine] = effecttest.Result{Out: []byte(stubText)}
	commit := "git -C " + cloneDir + " commit -q -S -m " + title + " -m " + body
	create := secrets.GH + " pr create --repo o/n --base main --head " + branch + " --title " + title + " --body " + body
	for line, out := range map[string]string{
		pullsLine: "", statusLine: "?? fnox.toml\n", switchLine: "", addLine: "", commit: "", headLine: "abc123\n", pushLine: "",
		create: pullURL + "\n", checksLine: bothGreen, mergeLine: "",
	} {
		m.Results[line] = effecttest.Result{Out: []byte(out)}
	}
	return m
}

// initOn is init on m, under the home /h, with in on its stdin and backup as the key it makes.
func initOn(m *effecttest.Machine, in io.Reader, backup *age.X25519Identity) (secrets.Init, *strings.Builder) {
	out := &strings.Builder{}
	k := &clock{now: time.Date(2026, 11, 19, 18, 0, 0, 0, time.UTC)}
	env := map[string]string{"HOME": "/h", "PATH": "/usr/bin"}
	return secrets.Init{
		M: m, Getenv: func(name string) string { return env[name] }, Out: out, In: in, Tree: secrets.Tree,
		NewBackup: func() (*age.X25519Identity, error) { return backup, nil }, Sleep: k.Sleep, Now: k.Now,
	}, out
}

// newBackup is a backup key made for the test, whose secret no file of the test holds.
func newBackup(t *testing.T) *age.X25519Identity {
	backup, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatal(err)
	}
	return backup
}

// count is the number of calls that start with prefix.
func count(calls []string, prefix string) int {
	n := 0
	for _, c := range calls {
		if strings.HasPrefix(c, prefix) {
			n++
		}
	}
	return n
}

func TestInitFirstRunLandsBothRecipientsAndEmptyInputs(t *testing.T) {
	m := initMac(startTitle, startBody, "OTEL_CLAUDE_EXPORTER_OTLP_ENDPOINT", "WS_GITHUB_TOKEN")
	backup := newBackup(t)
	i, out := initOn(m, strings.NewReader("\n"), backup)
	if err := i.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if string(m.Files[keyFile]) != stubText || m.Modes[keyFile] != 0o600 {
		t.Errorf("the key file is %q, %o, want the stub, 600", m.Files[keyFile], m.Modes[keyFile])
	}
	want, err := secrets.Start(recipientB, backup.Recipient().String())
	if err != nil || !bytes.Equal(m.Files[sharedFile], want) {
		t.Errorf("fnox.toml is\n%s\nwant Start's\n%s", m.Files[sharedFile], want)
	}
	for _, name := range []string{"OTEL_CLAUDE_EXPORTER_OTLP_ENDPOINT", "WS_GITHUB_TOKEN"} {
		if count(m.Calls, "run "+fnoxSet+name+" --provider age") != 1 {
			t.Errorf("init declared %s other than once: %q", name, m.Calls)
		}
	}
	if len(m.Inputs) != 2 || m.Inputs[0] != "" || m.Inputs[1] != "" {
		t.Errorf("fnox set's stdin is %q, want two empty ones", m.Inputs)
	}
	if count(m.Calls, "run "+mergeLine) != 1 || !strings.Contains(out.String(), "merged "+pullURL) {
		t.Errorf("init merged no pull request: %q", m.Calls)
	}
	if m.Calls[len(m.Calls)-1] != "run rm -rf "+cloneBase {
		t.Errorf("the clone stays: %q", m.Calls)
	}
}

func TestInitWritesTheBackupsSecretNowhere(t *testing.T) {
	m := initMac(startTitle, startBody, "WS_GITHUB_TOKEN")
	m.Results[mergeLine] = effecttest.Result{Err: errors.New("merge refused")}
	backup := newBackup(t)
	secret := backup.String()
	i, out := initOn(m, strings.NewReader("\n"), backup)
	err := i.Run(context.Background())
	if err == nil || strings.Contains(err.Error(), secret) {
		t.Errorf("a failed merge gives an error of %d characters, want one without the secret line",
			len(errString(err)))
	}
	if n := strings.Count(out.String(), secret); n != 1 {
		t.Errorf("init printed the secret line %d times, want once", n)
	}
	for _, c := range append(m.Calls, m.Inputs...) {
		if strings.Contains(c, secret) {
			t.Errorf("a call or an input of %d characters holds the secret line", len(c))
		}
	}
	for file, data := range m.Files {
		if bytes.Contains(data, []byte(secret)) {
			t.Errorf("%s holds the secret line", file)
		}
	}
}

func TestInitRemovesTheCloneWhereTheLandingFails(t *testing.T) {
	for name, line := range map[string]string{"push": pushLine, "check": checksLine, "merge": mergeLine} {
		m := initMac(startTitle, startBody, "WS_GITHUB_TOKEN")
		m.Results[line] = effecttest.Result{Err: errors.New("refused")}
		if line == checksLine {
			m.Results[line] = effecttest.Result{Out: []byte(`[{"name":"values","bucket":"fail"}]`)}
		}
		i, _ := initOn(m, strings.NewReader("\n"), newBackup(t))
		if err := i.Run(context.Background()); err == nil {
			t.Errorf("init with a failed %s passes", name)
		}
		if count(m.Calls, "run "+line) != 1 || m.Calls[len(m.Calls)-1] != "run rm -rf "+cloneBase {
			t.Errorf("init with a failed %s leaves the clone: %q", name, m.Calls)
		}
	}
}

// errString is err's text, or "" for none.
func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func TestInitStopsWhereTheBackupIsNotKept(t *testing.T) {
	m := initMac(startTitle, startBody, "WS_GITHUB_TOKEN")
	i, _ := initOn(m, strings.NewReader(""), newBackup(t))
	if err := i.Run(context.Background()); err == nil || !strings.Contains(err.Error(), "nothing landed") {
		t.Errorf("init with no answer gives %v, want an error that nothing landed", err)
	}
	if count(m.Calls, "run "+pushLine) != 0 || count(m.Calls, "run "+secrets.GH+" pr create") != 0 {
		t.Errorf("init with no answer pushed or opened a pull request: %q", m.Calls)
	}
}

func TestInitNeedsTheTools(t *testing.T) {
	m := initMac(startTitle, startBody, "WS_GITHUB_TOKEN")
	delete(m.Files, "/usr/local/bin/fnox")
	i, _ := initOn(m, strings.NewReader("\n"), newBackup(t))
	err := i.Run(context.Background())
	if err == nil || !strings.Contains(err.Error(), "run ws macos setup first") {
		t.Errorf("init without fnox gives %v, want an error naming ws macos setup", err)
	}
	if count(m.Calls, "run "+secrets.GH) != 0 {
		t.Errorf("init without fnox ran gh: %q", m.Calls)
	}
}
