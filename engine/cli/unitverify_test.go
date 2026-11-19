package cli_test

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

// miseStatus is the command that ws unit verify reads mise's packages by.
const miseStatus = "mise bootstrap packages status --json"

// verifyEnv runs ws unit verify on specs, with a sample of mise's status in testdata. mise printed
// that sample for the units file/a and file/b and an apt package.
func verifyEnv(t *testing.T, specs string) (status int, stdout, stderr, runDir string) {
	sample, err := os.ReadFile("testdata/mise-status.json")
	if err != nil {
		t.Fatal(err)
	}
	e, m := unitEnv(t, map[string]string{"r/units.toml.tmpl": specs, "r/a": "1\n", "r/b": "2\n", "r/c": "3\n"})
	e.Euid = 1000
	m.Results[miseStatus] = effecttest.Result{Out: sample}
	status, stdout, stderr = runUnit(e, "verify")
	return status, stdout, stderr, filepath.Dir(e.Getenv("WS_UNITS"))
}

func TestVerifyPassesWithEveryUnit(t *testing.T) {
	status, stdout, stderr, _ := verifyEnv(t, fileUnit("a", "/etc/a", "")+fileUnit("b", "/etc/b", ""))
	if status != 0 || stdout != "ws units: none missing\n" || stderr != "" {
		t.Errorf("status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}
}

func TestVerifyNamesAMissingUnit(t *testing.T) {
	status, stdout, stderr, runDir := verifyEnv(t, fileUnit("a", "/etc/a", "")+fileUnit("c", "/etc/c", ""))
	if status != 1 || stdout != "" || !strings.Contains(stderr, "mise's plan lacks ws:file/c:") ||
		!strings.Contains(stderr, "or run mise in "+runDir) {
		t.Errorf("status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}
}

func TestVerifyFailsWithoutTheWsManager(t *testing.T) {
	for name, out := range map[string]string{
		"no plugin":  "{}\n",
		"not loaded": `{"ws": {"available": false, "packages": [{"package": "file/a"}]}}`,
	} {
		e, m := unitEnv(t, map[string]string{"r/units.toml.tmpl": fileUnit("a", "/etc/a", ""), "r/a": "1\n"})
		m.Results[miseStatus] = effecttest.Result{Out: []byte(out)}
		runDir := filepath.Dir(e.Getenv("WS_UNITS"))
		if status, _, stderr := runUnit(e, "verify"); status != 1 || !strings.Contains(stderr, "no ws plugin loaded") ||
			!strings.Contains(stderr, "or run mise in "+runDir) {
			t.Errorf("%s: status = %d, stderr = %q", name, status, stderr)
		}
	}
}

func TestVerifyFailsWithoutMise(t *testing.T) {
	e, m := unitEnv(t, nil)
	m.Results[miseStatus] = effecttest.Result{Err: fmt.Errorf("[mise]: %w", exec.ErrNotFound)}
	if status, _, stderr := runUnit(e, "verify"); status != 1 || !strings.Contains(stderr, "mise is not on PATH") {
		t.Errorf("status = %d, stderr = %q", status, stderr)
	}
}

func TestVerifyFailsWhereMiseFails(t *testing.T) {
	e, m := unitEnv(t, nil)
	m.Results[miseStatus] = effecttest.Result{Err: errors.New("exit status 1")}
	if status, _, stderr := runUnit(e, "verify"); status != 1 || !strings.Contains(stderr, "reading mise's packages") {
		t.Errorf("status = %d, stderr = %q", status, stderr)
	}
}
