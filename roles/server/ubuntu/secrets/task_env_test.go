package secrets_test

import (
	"os/exec"
	"strings"
	"testing"
)

// fnox passes its environment on. So the task's umask and the fetch's settings of git stay the
// task's, where the bootstrap's own git and files would take them.
func TestTaskLeavesTheBootstrapTheCallersUmaskAndNoGitSettings(t *testing.T) {
	f := newFixture(t)
	mask, err := exec.CommandContext(t.Context(), "bash", "-c", "umask").Output()
	if err != nil {
		t.Fatal(err)
	}
	if code, out := f.run(); code != 0 {
		t.Fatalf("the task exits %d, want 0:\n%s", code, out)
	}
	want := "umask=" + strings.TrimSpace(string(mask)) + " git=\n"
	if got := f.log("mise-env"); got != want {
		t.Errorf("mise ran with %q, want %q", got, want)
	}
}
