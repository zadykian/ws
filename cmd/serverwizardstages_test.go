package cmd

import (
	"github.com/zadykian/ws/engine/core/wizard"
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

// wizardProcess returns a root process on a machine that holds vars as host_vars, booted or not,
// whose remote script reports status as host_vars' ws_user reads it.
func wizardProcess(vars string, booted bool, status string) (cli.Env, *effecttest.Machine) {
	p, m := valuesProcess(vars)
	p.GOOS = "linux"
	if booted {
		m.Dirs["/run/systemd/system"] = 0o755
	}
	c, _ := wsWant(m)
	m.Results["env SUDO_USER="+c.User+" sh -s -- status"] = effecttest.Result{Out: []byte(status)}
	return p, m
}

// ws's run waits on ws's values, and reads the target and the user that host_vars names. claude and
// the signing key wait on the run.
func TestServerWizardStagesReadHostVars(t *testing.T) {
	p, m := wizardProcess("---\nsecurity_outbound: false\nws_user: root\n", false, "")
	stages := serverWizardStages(wizardEffects{p: p, hand: &wizard.Hand{}}, wizardOptions{})
	var got []string
	for _, s := range stages {
		got = append(got, s.Name()+" < "+s.WaitsOn())
	}
	if !slices.Equal(got, []string{"ws's values < ", "ws's run < ws's values", "claude on ws < ws's run",
		"Signing key < ws's run"}) {
		t.Errorf("stages %q", got)
	}
	if c, err := wsWant(m); err != nil || c.Target != "cloud" || c.User != "root" {
		t.Errorf("wsWant = %+v, %v", c, err)
	}
	if c, err := wsWant(effecttest.NewMachine()); err != nil || c.Target != "lan" || c.User != "zadykian" {
		t.Errorf("wsWant of no file = %+v, %v", c, err)
	}
}
