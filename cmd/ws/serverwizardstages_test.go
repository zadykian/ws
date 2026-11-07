package main

import (
	"slices"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

// wizardProcess returns a root process on a machine that holds vars as host_vars, booted or not,
// whose remote script reports status.
func wizardProcess(vars string, booted bool, status string) (process, *effecttest.Machine) {
	p, m := valuesProcess(vars)
	p.goos = "linux"
	if booted {
		m.Dirs["/run/systemd/system"] = 0o755
	}
	m.Results["sh -s -- status"] = effecttest.Result{Out: []byte(status)}
	return p, m
}

// ws's run waits on ws's values, and reads the target and the user that host_vars names.
func TestServerWizardStagesReadHostVars(t *testing.T) {
	p, m := wizardProcess("---\nsecurity_outbound: false\nws_user: root\n", false, "")
	stages := serverWizardStages(p, nil, nil, nil, false)
	if got := []string{stages[0].Name(), stages[1].Name(), stages[1].WaitsOn()}; !slices.Equal(got,
		[]string{"ws's values", "ws's run", "ws's values"}) {
		t.Errorf("stages %q", got)
	}
	if c, err := wsWant(m); err != nil || c.Target != "cloud" || c.User != "root" {
		t.Errorf("wsWant = %+v, %v", c, err)
	}
	if c, err := wsWant(effecttest.NewMachine()); err != nil || c.Target != "lan" || c.User != "zadykian" {
		t.Errorf("wsWant of no file = %+v, %v", c, err)
	}
}
