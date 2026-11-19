package secrets_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"filippo.io/age"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/client/macos/secrets"
)

// The change of a later run, which declares one input that fnoxWrote lacks.
const (
	inputsTitle = "chore: declare the inputs of a newer release"
	inputsBody  = "The inputs that a newer release of ws declares, each empty: WS_GIT_TOKENS."
)

// unread is a stdin that a test must not read.
type unread struct{ t *testing.T }

func (u unread) Read([]byte) (int, error) {
	u.t.Error("init read its stdin")
	return 0, nil
}

func TestInitAgainDeclaresTheNewInputsAlone(t *testing.T) {
	m := initMac(inputsTitle, inputsBody, "OTEL_CLAUDE_EXPORTER_OTLP_ENDPOINT", "WS_GITHUB_TOKEN", "WS_GIT_TOKENS")
	m.Files[sharedFile] = []byte(fnoxWrote)
	m.Files[keyFile] = []byte(stubText)
	m.Results[statusLine] = effecttest.Result{Out: []byte(" M fnox.toml\n")}
	i, out := initOn(m, unread{t}, nil)
	if err := i.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if n := count(m.Calls, "run "+fnoxSet); n != 1 || count(m.Calls, "run "+fnoxSet+"WS_GIT_TOKENS ") != 1 {
		t.Errorf("init ran fnox set %d times, want once, of WS_GIT_TOKENS: %q", n, m.Calls)
	}
	if string(m.Files[sharedFile]) != fnoxWrote || count(m.Calls, "write ") != 0 {
		t.Errorf("init wrote a file where fnox alone changes fnox.toml: %q", m.Calls)
	}
	if strings.Contains(out.String(), "backup key") || count(m.Calls, "run "+mergeLine) != 1 {
		t.Errorf("init printed a backup or merged nothing:\n%s", out)
	}
}

func TestInitAgainWithNothingNewOpensNothing(t *testing.T) {
	m := initMac(inputsTitle, inputsBody, "OTEL_CLAUDE_EXPORTER_OTLP_ENDPOINT", "WS_GITHUB_TOKEN")
	m.Files[sharedFile] = []byte(fnoxWrote)
	m.Files[keyFile] = []byte(stubText)
	i, out := initOn(m, unread{t}, nil)
	if err := i.Run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if count(m.Calls, "run "+fnoxSet) != 0 || count(m.Calls, "run "+switchLine) != 0 {
		t.Errorf("init with nothing new declared or committed: %q", m.Calls)
	}
	if want := "o/n holds both recipients and every input: nothing to land\n"; !strings.HasSuffix(out.String(), want) {
		t.Errorf("init with nothing new says\n%s\nwant it to end with %q", out, want)
	}
}

func TestInitRefusesAFnoxTomlWithoutBothRecipients(t *testing.T) {
	m := initMac(inputsTitle, inputsBody, "WS_GITHUB_TOKEN")
	m.Files[sharedFile] = []byte(strings.Replace(fnoxWrote, "backup = ", "spare = ", 1))
	i, _ := initOn(m, unread{t}, nil)
	err := i.Run(context.Background())
	if err == nil || !strings.Contains(err.Error(), "no recipient backup") {
		t.Errorf("a fnox.toml without backup gives %v, want an error naming backup", err)
	}
	if count(m.Calls, "run "+secrets.Plugin) != 0 || count(m.Calls, "run "+fnoxSet) != 0 {
		t.Errorf("init went on past a fnox.toml without backup: %q", m.Calls)
	}
}

func TestInitRefusesWhileItsPullRequestIsOpen(t *testing.T) {
	m := initMac(startTitle, startBody, "WS_GITHUB_TOKEN")
	m.Results[pullsLine] = effecttest.Result{Out: []byte("ws/add-ws-a-20261119-170000\thttps://github.com/o/n/pull/4\n" +
		"ws/init-20261119-170000\thttps://github.com/o/n/pull/3\n")}
	i, out := initOn(m, unread{t}, nil)
	i.NewBackup = func() (*age.X25519Identity, error) {
		t.Error("init made a backup while its pull request is open")
		return nil, errors.New("no backup")
	}
	err := i.Run(context.Background())
	if err == nil || !strings.Contains(err.Error(), "https://github.com/o/n/pull/3 is still open") ||
		strings.Contains(err.Error(), "pull/4") {
		t.Errorf("init with its pull request open gives %v, want an error naming that one alone", err)
	}
	if count(m.Calls, "run "+secrets.Plugin) != 0 || count(m.Calls, "run "+fnoxSet) != 0 || out.Len() != 0 {
		t.Errorf("init went on past its open pull request: %q, saying %q", m.Calls, out)
	}
	if m.Calls[len(m.Calls)-1] != "run rm -rf "+cloneBase {
		t.Errorf("the clone stays: %q", m.Calls)
	}
}

func TestInitFailsWhereFnoxLeavesFnoxTomlAsItWas(t *testing.T) {
	m := initMac(inputsTitle, inputsBody, "OTEL_CLAUDE_EXPORTER_OTLP_ENDPOINT", "WS_GITHUB_TOKEN", "WS_GIT_TOKENS")
	m.Files[sharedFile] = []byte(fnoxWrote)
	m.Files[keyFile] = []byte(stubText)
	m.Results[statusLine] = effecttest.Result{}
	i, _ := initOn(m, unread{t}, nil)
	err := i.Run(context.Background())
	if err == nil || !strings.Contains(err.Error(), "nothing landed: WS_GIT_TOKENS stay undeclared") {
		t.Errorf("a fnox set that leaves fnox.toml as it was gives %v, want an error that nothing landed", err)
	}
	if count(m.Calls, "run "+switchLine) != 0 {
		t.Errorf("init committed with no change: %q", m.Calls)
	}
}
