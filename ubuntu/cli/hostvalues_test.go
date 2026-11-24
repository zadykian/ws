package cli

import (
	"strings"
	"testing"
)

func TestHostValuesSetWritesOneKey(t *testing.T) {
	p, m := valuesProcess("---\n# the machine\nWS_BASE_LOCALE: C\n")
	status, _, stderr := runWs(p, "", "ubuntu", "host", "values", "set", "WS_BASE_SWAPPINESS", "20")
	if status != 0 || !strings.Contains(stderr, "wrote "+hostVars) {
		t.Fatalf("status = %d, stderr %q", status, stderr)
	}
	if got := string(m.Files[hostVars]); got != "---\n# the machine\nWS_BASE_LOCALE: C\nWS_BASE_SWAPPINESS: 20\n" {
		t.Errorf("file = %q", got)
	}
}

// The host reads the keys of its own roles: a guest's key is none of its.
func TestHostValuesSetRefusesAGuestsKey(t *testing.T) {
	p, _ := valuesProcess("---\n")
	status, _, stderr := runWs(p, "", "ubuntu", "host", "values", "set", "WS_ZED_NOFILE", "1")
	if status == 0 || !strings.Contains(stderr, "no line holds WS_ZED_NOFILE") {
		t.Errorf("status = %d, stderr %q", status, stderr)
	}
}

func TestHostValuesCheck(t *testing.T) {
	for _, tc := range []struct {
		name, file, stdout, stderr string
		status                     int
	}{
		{"a pass", "---\nWS_BASE_LOCALE: C\n", valuesChecked + "\n", "", 0},
		{"a bad value", "WS_BASE_SWAPPINESS: 400\n", "", "WS_BASE_SWAPPINESS is 400", 1},
		{"a guest's key", "WS_ZED_NOFILE: 1\n", "", "edit " + hostVars + " by hand", 1},
	} {
		p, _ := valuesProcess(tc.file)
		status, stdout, stderr := runWs(p, "", "ubuntu", "host", "values", "check")
		if status != tc.status || stdout != tc.stdout || !strings.Contains(stderr, tc.stderr) {
			t.Errorf("%s: status %d, stdout %q, stderr %q", tc.name, status, stdout, stderr)
		}
	}
}
