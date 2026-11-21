package cli

import (
	"strings"
	"testing"
)

func TestValuesCheckPassesAGoodFileAndNone(t *testing.T) {
	for name, file := range map[string]string{"a good file": "---\nWS_BASE_LOCALE: C\n", "an empty file": ""} {
		t.Run(name, func(t *testing.T) {
			p, _ := valuesProcess(file)
			status, stdout, stderr := runWs(p, "", "server", "values", "check")
			if status != 0 || stdout != valuesChecked+"\n" || stderr != "" {
				t.Errorf("status = %d, stdout %q, stderr %q", status, stdout, stderr)
			}
		})
	}
	p, m := valuesProcess("")
	delete(m.Files, serverVars)
	if status, stdout, _ := runWs(p, "", "server", "values", "check"); status != 0 || stdout != valuesChecked+"\n" {
		t.Errorf("no file: status = %d, stdout %q", status, stdout)
	}
}

func TestValuesCheckNamesTheKeyItsRuleAndTheFile(t *testing.T) {
	p, m := valuesProcess("---\nWS_BASE_LOCALE: C\nWS_BASE_SWAPPINESS: 400\n")
	status, stdout, stderr := runWs(p, "", "server", "values", "check")
	if status != 1 || stdout != "" {
		t.Fatalf("status = %d, stdout %q, stderr %q", status, stdout, stderr)
	}
	for _, want := range []string{"WS_BASE_SWAPPINESS is 400: it takes", "edit " + serverVars + " by hand"} {
		if !strings.Contains(stderr, want) {
			t.Errorf("stderr %q lacks %q", stderr, want)
		}
	}
	if strings.Contains(stderr, "values set") {
		t.Errorf("stderr names values set, which refuses while a value is bad: %q", stderr)
	}
	for _, call := range m.Calls {
		if !strings.HasPrefix(call, "read") {
			t.Errorf("a call besides a read: %s", call)
		}
	}
}
