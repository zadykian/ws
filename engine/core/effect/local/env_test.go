package local

import (
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"
)

func TestWithEnv(t *testing.T) {
	base := []string{"PATH=/bin", "LANG=C", "A=1"}
	got := withEnv(base, []string{"LANG=C.UTF-8", "NEW=x"})
	want := []string{"PATH=/bin", "A=1", "LANG=C.UTF-8", "NEW=x"}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if !slices.Equal(base, []string{"PATH=/bin", "LANG=C", "A=1"}) {
		t.Errorf("base changed: %v", base)
	}
}

// repoVars holds each variable that the git on PATH lists as one that names a repository.
func TestRepoVarsHoldGitsList(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("no git on PATH")
	}
	cmd := exec.CommandContext(t.Context(), "git", "rev-parse", "--local-env-vars")
	cmd.Dir, cmd.Env = t.TempDir(), []string{"PATH=" + os.Getenv("PATH"), "HOME=" + t.TempDir()}
	out, err := cmd.Output()
	if err != nil {
		t.Fatal(err)
	}
	for name := range strings.FieldsSeq(string(out)) {
		if !slices.Contains(repoVars, name) {
			t.Errorf("repoVars lacks %s", name)
		}
	}
}

func TestOSMachineExecNoCommand(t *testing.T) {
	if err := (OSMachine{}).Exec("ws-no-such-command", nil, nil); err == nil {
		t.Error("Exec of a missing command returned nil")
	}
}
