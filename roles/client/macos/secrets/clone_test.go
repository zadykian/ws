package secrets_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/client/macos/secrets"
)

// The repository o/n, cloned into the directory that mktemp names.
const (
	cloneBase = "/tmp/ws.x"
	cloneDir  = cloneBase + "/repo"
	apiLine   = secrets.GH + " api repos/o/n --jq .private"
	cloneLine = "env GIT_TERMINAL_PROMPT=0 git clone -q --branch main --single-branch https://github.com/o/n.git " +
		cloneDir
	checkFile = cloneDir + "/.github/workflows/check.yml"
)

var repo = secrets.Repo{Owner: "o", Name: "n"}

// cloneable is a Mac whose gh reads o/n as private, and whose clone of it holds the checks.
func cloneable() *effecttest.Machine {
	m := effecttest.NewMachine()
	m.Results[apiLine] = effecttest.Result{Out: []byte("true\n")}
	m.Results["mktemp -d"] = effecttest.Result{Out: []byte(cloneBase + "\n")}
	m.Results[cloneLine] = effecttest.Result{}
	m.Results["rm -rf "+cloneBase] = effecttest.Result{}
	m.Files[checkFile] = []byte("on: pull_request\n")
	return m
}

// open opens o/n on m, and fails the test where it cannot.
func open(t *testing.T, m *effecttest.Machine) *secrets.Clone {
	c, err := secrets.Open(context.Background(), m, repo)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestOpenRefusesAMissingRepository(t *testing.T) {
	m := cloneable()
	m.Results[apiLine] = effecttest.Result{Err: errors.New("[gh api]: exit status 1: gh: Not Found (HTTP 404)")}
	_, err := secrets.Open(context.Background(), m, repo)
	if err == nil || !strings.Contains(err.Error(), "gh repo create o/n --private") {
		t.Errorf("a missing repository gives %v, want an error naming gh repo create o/n --private", err)
	}
	if slices.Contains(m.Calls, "run mktemp -d") {
		t.Errorf("a missing repository made a directory: %q", m.Calls)
	}
}

func TestOpenRefusesAPublicRepository(t *testing.T) {
	m := cloneable()
	m.Results[apiLine] = effecttest.Result{Out: []byte("false\n")}
	_, err := secrets.Open(context.Background(), m, repo)
	if err == nil || !strings.Contains(err.Error(), "public") {
		t.Errorf("a public repository gives %v, want an error", err)
	}
	if slices.Contains(m.Calls, "run "+cloneLine) {
		t.Errorf("a public repository was cloned: %q", m.Calls)
	}
}

func TestOpenRefusesACloneWithoutItsChecks(t *testing.T) {
	m := cloneable()
	delete(m.Files, checkFile)
	_, err := secrets.Open(context.Background(), m, repo)
	if err == nil || !strings.Contains(err.Error(), ".github/workflows/check.yml") {
		t.Errorf("a clone without its checks gives %v, want an error naming check.yml", err)
	}
	if m.Calls[len(m.Calls)-1] != "run rm -rf "+cloneBase {
		t.Errorf("the clone's directory stays: %q", m.Calls)
	}
}

func TestOpenClonesMainByHTTPS(t *testing.T) {
	m := cloneable()
	c := open(t, m)
	if c.Dir != cloneDir || c.Repo != repo {
		t.Errorf("Open gives the clone %s of %+v, want %s of o/n", c.Dir, c.Repo, cloneDir)
	}
	c.Close(context.Background())
	want := []string{"run " + apiLine, "run mktemp -d", "run " + cloneLine, "stat " + checkFile, "run rm -rf " + cloneBase}
	if !slices.Equal(m.Calls, want) {
		t.Errorf("Open and Close ran\n%q\nwant\n%q", m.Calls, want)
	}
}

func TestFnoxRunsInTheCloneWithItsEnv(t *testing.T) {
	m := cloneable()
	c := open(t, m)
	c.FnoxEnv = []string{"PATH=/usr/local/bin:/usr/bin:/bin", "FNOX_AGE_KEY_FILE=/h/.config/fnox/age.txt"}
	line := "env -u FNOX_AGE_KEY -u FNOX_PROFILE PATH=/usr/local/bin:/usr/bin:/bin FNOX_AGE_KEY_FILE=/h/.config/fnox/age.txt " +
		`sh -c cd "$1" && shift && exec "$@" sh ` + cloneDir + " fnox set A --provider age"
	m.Results[line] = effecttest.Result{Out: []byte("set\n")}
	out, err := c.Fnox(context.Background(), []byte("v"), "set", "A", "--provider", "age")
	if err != nil || string(out) != "set\n" {
		t.Fatalf("Fnox gives %q, %v, want the line's output", out, err)
	}
	if !slices.Equal(m.Inputs, []string{"v"}) {
		t.Errorf("fnox's stdin is %q, want the input alone", m.Inputs)
	}
}
