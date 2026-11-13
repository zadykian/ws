package cli

import (
	"strings"
	"testing"
)

func TestServerApplyHelpNamesJSON(t *testing.T) {
	status, stdout, _ := runWs(fakeProcess(), "", "server", "ubuntu", "apply", "-h")
	if status != 0 || !strings.Contains(stdout, "--json") || !strings.Contains(stdout, "no screen is drawn") {
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

// A terminal changes nothing for --json: the events reach stdout, and no screen takes them.
func TestServerApplyJSONDrawsNoScreenAtATerminal(t *testing.T) {
	p, _ := serverTerminal()
	status, stdout, stderr := runWs(p, "", "server", "ubuntu", "apply", "--json", "--check", "--only", "tmux")
	if status != 0 || !strings.Contains(stdout, `"event":"done","parent":"tmux"`) || strings.Contains(stderr, enterScreen) {
		t.Errorf("status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}
}

// With no selection the list would open here. The run ends in Prepare, so that no role runs.
func TestServerApplyJSONOpensNoListAtATerminal(t *testing.T) {
	p, m := serverTerminal()
	m.Packages = nil
	status, stdout, stderr := runWs(p, "q", "server", "ubuntu", "apply", "--json")
	if status != 1 || !strings.Contains(stdout, `"event":"failed"`) || strings.Contains(stderr, enterScreen) {
		t.Errorf("status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}
}
