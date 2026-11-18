package cli

import (
	"errors"
	"os/exec"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/server/ubuntu/secrets"
)

const (
	ikewardBin    = "/usr/local/sbin/ikeward"
	ikewardValues = "---\nWS_IKEWARD_PROFILE: /root/vpn/vpnclient\nWS_IKEWARD_DNS: 10.20.0.53\n" +
		"WS_IKEWARD_DOMAINS: [example.internal]\n"
	// ikewardInstall is install's command line from those values, on the profile's files in /run.
	ikewardInstall = "run " + ikewardBin + " install --p12 /run/ikeward-profile/vpnclient.p12 --sswan " +
		"/run/ikeward-profile/vpnclient.sswan --dns 10.20.0.53 --domain=example.internal"
	ikewardPassword = "dummy-password"
)

// exitCode is the error of a command that exits with code.
func exitCode(t *testing.T, code string) error {
	_, err := exec.Command("sh", "-c", "exit "+code).Output() //nolint:gosec // a test runs a shell that exits
	if err == nil {
		t.Fatal("exit " + code + " succeeded")
	}
	return err
}

// ikewardMachine is a booted machine with ikeward's values and, with profile, a secrets file that
// holds the profile and its password.
func ikewardMachine(t *testing.T, profile bool) *effecttest.Machine {
	m, _ := credFixture()
	m.Dirs["/etc/ws"], m.Dirs["/run/systemd/system"] = 0o755, 0o755
	m.Files[serverVars] = []byte(ikewardValues)
	if profile {
		err := secrets.Store{M: m}.Edit(t.Context(), func(s *secrets.Secrets) error {
			s.Password.VPNP12 = ikewardPassword
			return errors.Join(s.SetFile("vpn_p12", []byte("p12")), s.SetFile("vpn_sswan", []byte("sswan")))
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	m.Calls = nil
	return m
}

// runIkeward runs ws server ubuntu ikeward with args on m.
func runIkeward(m *effecttest.Machine, args ...string) (int, string, string) {
	p := fakeProcess()
	p.Machine, p.Euid = m, 0
	return runWs(p, "", append([]string{"server", "ubuntu", "ikeward"}, args...)...)
}

func TestIkewardInstallCheckRunsTheDryRun(t *testing.T) {
	m := ikewardMachine(t, true)
	m.Results[strings.TrimPrefix(ikewardInstall, "run ")+" --dry-run --exit-code"] = effecttest.Result{
		Out: []byte("write /etc/x\n"), Err: exitCode(t, "4")}
	status, _, stderr := runIkeward(m, "install", "--check")
	if status != 1 || !strings.Contains(stderr, "to do: run ikeward install") {
		t.Errorf("status = %d, stderr %q; want install to do", status, stderr)
	}
	if slices.Contains(m.Calls, ikewardInstall+" --exit-code") {
		t.Errorf("calls %q; --check ran install", m.Calls)
	}
	if !slices.Equal(m.Inputs, []string{ikewardPassword}) {
		t.Errorf("inputs %d, want the password on the dry run's stdin alone", len(m.Inputs))
	}
}

func TestIkewardInstallWritesNoProfileAfter(t *testing.T) {
	m := ikewardMachine(t, true)
	m.Results[strings.TrimPrefix(ikewardInstall, "run ")+" --dry-run --exit-code"] = effecttest.Result{Err: exitCode(t, "4")}
	m.Results[strings.TrimPrefix(ikewardInstall, "run ")+" --exit-code"] = effecttest.Result{}
	status, stdout, stderr := runIkeward(m, "install")
	if status != 0 {
		t.Fatalf("status = %d, stderr %q", status, stderr)
	}
	if !slices.Contains(m.Calls, ikewardInstall+" --exit-code") {
		t.Errorf("calls %q, want install run", m.Calls)
	}
	if last := m.Calls[len(m.Calls)-1]; last != "root rm -rf -- /run/ikeward-profile" {
		t.Errorf("last call %q, want the profile's files removed", last)
	}
	if strings.Contains(strings.Join(m.Calls, "\n")+stdout+stderr, ikewardPassword) {
		t.Error("a call or the output shows the password")
	}
}

func TestIkewardInstallNeedsTheProfile(t *testing.T) {
	m := ikewardMachine(t, false)
	status, _, stderr := runIkeward(m, "install", "--check")
	if status != 1 || !strings.Contains(stderr, "the secrets hold no profile") {
		t.Errorf("status = %d, stderr %q; want the profile missing", status, stderr)
	}
	if slices.ContainsFunc(m.Calls, func(c string) bool { return strings.Contains(c, ikewardBin) }) {
		t.Errorf("calls %q; want no run of ikeward", m.Calls)
	}
	m.Files[serverVars] = []byte("---\n")
	if status, _, stderr := runIkeward(m, "install"); status != 1 || !strings.Contains(stderr, "WS_IKEWARD_PROFILE is empty") {
		t.Errorf("no profile: status = %d, stderr %q", status, stderr)
	}
}

func TestIkewardCheckNeedsTheProfile(t *testing.T) {
	m := ikewardMachine(t, false)
	if status, _, stderr := runIkeward(m, "check"); status != 1 || !strings.Contains(stderr, "the secrets hold no profile") {
		t.Errorf("no profile: status = %d, stderr %q", status, stderr)
	}
	m = ikewardMachine(t, true)
	if status, stdout, stderr := runIkeward(m, "check"); status != 0 || stdout != "" || stderr != "" {
		t.Errorf("status = %d, stdout %q, stderr %q", status, stdout, stderr)
	}
	if slices.ContainsFunc(m.Calls, func(c string) bool { return strings.Contains(c, ikewardBin) }) || len(m.Inputs) != 0 {
		t.Errorf("calls %q, %d inputs; want no run of ikeward", m.Calls, len(m.Inputs))
	}
}

func TestIkewardWaitGivesUp(t *testing.T) {
	m := ikewardMachine(t, false)
	m.Results[ikewardBin+" status"] = effecttest.Result{Err: exitCode(t, "1")}
	p := fakeProcess()
	p.Machine = m
	c := newIkewardWaitCmd(p, 2, time.Millisecond)
	var out strings.Builder
	c.SetOut(&out)
	c.SetErr(&out)
	c.SetArgs(nil)
	err := c.ExecuteContext(t.Context())
	if err == nil || !strings.Contains(err.Error(), "not up after 2 tries") {
		t.Errorf("wait = %v, want it to give up after 2 tries", err)
	}
	// One read to plan, then two tries.
	if n := strings.Count(strings.Join(m.Calls, "\n"), ikewardBin+" status"); n != 3 {
		t.Errorf("status ran %d times, want 3", n)
	}
}

func TestIkewardWaitDoneWhereUp(t *testing.T) {
	m := ikewardMachine(t, false)
	m.Results[ikewardBin+" status"] = effecttest.Result{}
	if status, _, stderr := runIkeward(m, "wait"); status != 0 || stderr != "" {
		t.Errorf("status = %d, stderr %q", status, stderr)
	}
}
