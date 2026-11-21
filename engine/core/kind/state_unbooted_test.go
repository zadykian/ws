package kind

import (
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

func TestStateRunUnbootedRunsWithoutTheSession(t *testing.T) {
	e, m, _, out := testEnv(false)
	const bare = "env HOME=/root env -u XDG_RUNTIME_DIR -u DBUS_SESSION_BUS_ADDRESS " + systemPath + " sh -c x"
	m.Results[bare] = effecttest.Result{}
	units := loadState(t, e, "run = { user = \"root\", command = [\"sh\", \"-c\", \"x\"], creates = [\"/x\"], unbooted = true }\n")
	if got := headlines(t, units[0].Step); !slices.Equal(got, []string{"run sh -c x as root"}) || out.Lines != nil {
		t.Errorf("plan = %q, lines %q; want the run, with no skip", got, out.Lines)
	}
	if err := applyOnce(t, units[0].Step); err != nil || !slices.Contains(m.Calls, "run "+bare) {
		t.Errorf("apply = %v, calls %q; want the command run as root, with no XDG_RUNTIME_DIR", err, m.Calls)
	}
}

func TestStateRunUnbootedKeepsTheSessionWhereSystemdRuns(t *testing.T) {
	e, m, _, _ := testEnv(true)
	m.Dirs["/run/user/0"] = 0o700
	m.Results[rootRun+"sh -c x"] = effecttest.Result{}
	units := loadState(t, e, "run = { user = \"root\", command = [\"sh\", \"-c\", \"x\"], creates = [\"/x\"], unbooted = true }\n")
	if err := applyOnce(t, units[0].Step); err != nil || !slices.Contains(m.Calls, "run "+rootRun+"sh -c x") {
		t.Errorf("apply = %v, calls %q; want the command run in root's session", err, m.Calls)
	}
}

// Booted systemd before root's session starts: no /run/user/UID, so no session's env, and
// the caller's is dropped, as a run keeps ws's env.
func TestStateRunUnbootedLeavesOutASessionNotStarted(t *testing.T) {
	e, m, _, _ := testEnv(true)
	const bare = "env HOME=/root env -u XDG_RUNTIME_DIR -u DBUS_SESSION_BUS_ADDRESS " + systemPath + " sh -c x"
	m.Results[bare] = effecttest.Result{}
	units := loadState(t, e, "run = { user = \"root\", command = [\"sh\", \"-c\", \"x\"], creates = [\"/x\"], unbooted = true }\n")
	if err := applyOnce(t, units[0].Step); err != nil || !slices.Contains(m.Calls, "run "+bare) {
		t.Errorf("apply = %v, calls %q; want the command run as root, with no XDG_RUNTIME_DIR", err, m.Calls)
	}
}
