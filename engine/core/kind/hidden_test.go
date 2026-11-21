package kind

import (
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

// rootSession is how a run state runs a command as root, after the hidden names: with the caller's
// session dropped.
const rootSession = "-u XDG_RUNTIME_DIR -u DBUS_SESSION_BUS_ADDRESS " + systemPath

// hiddenEnv is a booted machine's env whose run hides two names, and whose ws is at /w/ws.
func hiddenEnv() (Env, *effecttest.Machine) {
	e, m, _, _ := testEnv(true)
	e.Self, e.Hidden = "/w/ws", []string{"A_TOKEN", "B_TOKEN"}
	return e, m
}

func TestRunDropsHiddenNamesFromAnotherProgram(t *testing.T) {
	e, m := hiddenEnv()
	const line = "env HOME=/root env -u A_TOKEN -u B_TOKEN " + rootSession + " sh -c x"
	m.Results[line] = effecttest.Result{}
	units := loadState(t, e, "run = { user = \"root\", command = [\"sh\", \"-c\", \"x\"], creates = [\"/x\"] }\n")
	if err := applyOnce(t, units[0].Step); err != nil || !slices.Contains(m.Calls, "run "+line) {
		t.Errorf("apply = %v, calls %q; want the command run without the hidden names", err, m.Calls)
	}
}

func TestRunKeepsHiddenNamesForWs(t *testing.T) {
	e, m := hiddenEnv()
	const line = "env HOME=/root env " + rootSession + " /w/ws x"
	m.Results[line] = effecttest.Result{}
	units := loadState(t, e, "run = { user = \"root\", command = [\"/w/ws\", \"x\"], creates = [\"/x\"] }\n")
	if err := applyOnce(t, units[0].Step); err != nil || !slices.Contains(m.Calls, "run "+line) {
		t.Errorf("apply = %v, calls %q; want ws run with the run's env whole", err, m.Calls)
	}
}
