package cli

import (
	"strings"
	"testing"
)

func TestServerApplyHelpNamesJSON(t *testing.T) {
	status, stdout, _ := runWs(fakeProcess(), "", "server", "ubuntu", "apply", "-h")
	if status != 0 || !strings.Contains(stdout, "--json") {
		t.Errorf("status = %d, help = %q", status, stdout)
	}
}

func TestServerApplyJSONRefusesNonRootOnStdoutAlone(t *testing.T) {
	p := fakeProcess()
	p.Euid = 1000
	status, stdout, stderr := runWs(p, "", "server", "ubuntu", "apply", "--json")
	if status != 1 || stdout != "" || !strings.Contains(stderr, "run it as root") {
		t.Errorf("status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}
}

// The events reach stdout and ws's own lines stderr.
func TestServerApplyJSONPrintsTheEventsOnStdout(t *testing.T) {
	p, _ := serverProcess()
	status, stdout, stderr := runWs(p, "", "server", "ubuntu", "apply", "--json", "--check", "--only", "tmux")
	if status != 0 || !strings.Contains(stdout, `"event":"done","parent":"tmux"`) {
		t.Errorf("status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}
}

// The run ends in Prepare, so that no role runs.
func TestServerApplyJSONReportsAFailureOnStdout(t *testing.T) {
	p, m := serverProcess()
	m.Packages = nil
	status, stdout, stderr := runWs(p, "", "server", "ubuntu", "apply", "--json")
	if status != 1 || !strings.Contains(stdout, `"event":"failed"`) {
		t.Errorf("status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}
}
