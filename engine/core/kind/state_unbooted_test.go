package kind

import (
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

func TestStateRunUnbootedRunsWithoutTheSession(t *testing.T) {
	e, m, _, out := testEnv(false)
	withAccount(m)
	const bare = "runuser -u u -- env -i -C /home/u HOME=/home/u " + systemPath + " env -u XDG_RUNTIME_DIR -u DBUS_SESSION_BUS_ADDRESS " + systemPath + " sh -c x"
	m.Results[bare] = effecttest.Result{}
	units := loadState(t, e, "run = { user = \"u\", command = [\"sh\", \"-c\", \"x\"], creates = [\"/x\"], unbooted = true }\n")
	if got := headlines(t, units[0].Step); !slices.Equal(got, []string{"run sh -c x as u"}) || out.Lines != nil {
		t.Errorf("plan = %q, lines %q; want the run, with no skip", got, out.Lines)
	}
	if err := applyOnce(t, units[0].Step); err != nil || !slices.Contains(m.Calls, "root "+bare) {
		t.Errorf("apply = %v, calls %q; want the command run as u, with no XDG_RUNTIME_DIR", err, m.Calls)
	}
}

func TestStateRunUnbootedKeepsTheSessionWhereSystemdRuns(t *testing.T) {
	e, m, _, _ := testEnv(true)
	withAccount(m)
	m.Dirs["/run/user/1001"] = 0o700
	m.Results[asU+"sh -c x"] = effecttest.Result{}
	units := loadState(t, e, "run = { user = \"u\", command = [\"sh\", \"-c\", \"x\"], creates = [\"/x\"], unbooted = true }\n")
	if err := applyOnce(t, units[0].Step); err != nil || !slices.Contains(m.Calls, "root "+asU+"sh -c x") {
		t.Errorf("apply = %v, calls %q; want the command run in u's session", err, m.Calls)
	}
}

// Booted systemd before the account's session starts: no /run/user/UID, so no session's env, and
// the caller's is dropped, which runuser would pass on.
func TestStateRunUnbootedLeavesOutASessionNotStarted(t *testing.T) {
	e, m, _, _ := testEnv(true)
	withAccount(m)
	const bare = "runuser -u u -- env -i -C /home/u HOME=/home/u " + systemPath + " env -u XDG_RUNTIME_DIR -u DBUS_SESSION_BUS_ADDRESS " + systemPath + " sh -c x"
	m.Results[bare] = effecttest.Result{}
	units := loadState(t, e, "run = { user = \"u\", command = [\"sh\", \"-c\", \"x\"], creates = [\"/x\"], unbooted = true }\n")
	if err := applyOnce(t, units[0].Step); err != nil || !slices.Contains(m.Calls, "root "+bare) {
		t.Errorf("apply = %v, calls %q; want the command run as u, with no XDG_RUNTIME_DIR", err, m.Calls)
	}
}
