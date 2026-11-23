package cli

import (
	"encoding/base64"
	"os/exec"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/zadykian/ws/engine/cli/clitest"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/server/ubuntu/ikeward"
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

// ikewardProfile is the env of a run with the secrets, as the task's fnox sets the profile's inputs.
var ikewardProfile = map[string]string{ikeward.P12Var: base64.StdEncoding.EncodeToString([]byte("p12")),
	ikeward.SswanVar: "sswan", ikeward.PasswordVar: ikewardPassword}

// ikewardNoProfile is the env of a run with the secrets on a host whose profile is empty.
var ikewardNoProfile = map[string]string{ikeward.P12Var: "", ikeward.SswanVar: "", ikeward.PasswordVar: ""}

// ikewardMachine is a machine with ikeward's values.
func ikewardMachine() *effecttest.Machine {
	m := effecttest.NewMachine()
	m.Dirs["/etc/ws"] = 0o755
	m.Files[serverVars] = []byte(ikewardValues)
	return m
}

// runIkeward runs ws server ikeward with args on m, in a run whose env is env.
func runIkeward(m *effecttest.Machine, env map[string]string, args ...string) (int, string, string) {
	p := clitest.WithEnv(fakeProcess(), env)
	p.Machine, p.Euid = m, 0
	return runWs(p, "", append([]string{"server", "ikeward"}, args...)...)
}

func TestIkewardInstallCheckRunsTheDryRun(t *testing.T) {
	m := ikewardMachine()
	m.Results[strings.TrimPrefix(ikewardInstall, "run ")+" --dry-run --exit-code"] = effecttest.Result{
		Out: []byte("write /etc/x\n"), Err: exitCode(t, "4")}
	status, _, stderr := runIkeward(m, ikewardProfile, "install", "--check")
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
	m := ikewardMachine()
	m.Results[strings.TrimPrefix(ikewardInstall, "run ")+" --dry-run --exit-code"] = effecttest.Result{Err: exitCode(t, "4")}
	m.Results[strings.TrimPrefix(ikewardInstall, "run ")+" --exit-code"] = effecttest.Result{}
	status, stdout, stderr := runIkeward(m, ikewardProfile, "install")
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
	m := ikewardMachine()
	status, _, stderr := runIkeward(m, ikewardNoProfile, "install", "--check")
	if status != 1 || !strings.Contains(stderr, "WS_IKEWARD_P12 or WS_IKEWARD_SSWAN is empty") {
		t.Errorf("status = %d, stderr %q; want the profile missing", status, stderr)
	}
	if slices.ContainsFunc(m.Calls, func(c string) bool { return strings.Contains(c, ikewardBin) }) {
		t.Errorf("calls %q; want no run of ikeward", m.Calls)
	}
	m.Files[serverVars] = []byte("---\n")
	if status, _, stderr := runIkeward(m, ikewardProfile, "install"); status != 1 || !strings.Contains(stderr, "WS_IKEWARD_PROFILE is empty") {
		t.Errorf("no profile: status = %d, stderr %q", status, stderr)
	}
}

func TestIkewardCheckNeedsTheProfile(t *testing.T) {
	m := ikewardMachine()
	status, _, stderr := runIkeward(m, ikewardNoProfile, "check")
	if status != 1 || !strings.Contains(stderr, "WS_IKEWARD_P12 or WS_IKEWARD_SSWAN is empty") {
		t.Errorf("no profile: status = %d, stderr %q", status, stderr)
	}
	// A run without the secrets, and one with a profile, pass.
	for _, env := range []map[string]string{nil, ikewardProfile} {
		m = ikewardMachine()
		if status, stdout, stderr := runIkeward(m, env, "check"); status != 0 || stdout != "" || stderr != "" {
			t.Errorf("status = %d, stdout %q, stderr %q", status, stdout, stderr)
		}
		if slices.ContainsFunc(m.Calls, func(c string) bool { return strings.Contains(c, ikewardBin) }) || len(m.Inputs) != 0 {
			t.Errorf("calls %q, %d inputs; want no run of ikeward", m.Calls, len(m.Inputs))
		}
	}
}

// A run without the secrets env leaves the VPN alone: install plans nothing and runs no ikeward.
func TestIkewardInstallWithoutTheSecretsKeepsTheVPN(t *testing.T) {
	m := ikewardMachine()
	status, _, stderr := runIkeward(m, nil, "install", "--check")
	if status != 0 || !strings.Contains(stderr, "the VPN stays as it is") {
		t.Errorf("status = %d, stderr %q; want the VPN kept", status, stderr)
	}
	if slices.ContainsFunc(m.Calls, func(c string) bool { return strings.Contains(c, ikewardBin) }) || len(m.Inputs) != 0 {
		t.Errorf("calls %q, %d inputs; want no run of ikeward", m.Calls, len(m.Inputs))
	}
}

func TestIkewardWaitGivesUp(t *testing.T) {
	m := ikewardMachine()
	m.Results[ikewardBin+" status"] = effecttest.Result{Err: exitCode(t, "1")}
	p := clitest.WithEnv(fakeProcess(), ikewardProfile)
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
	m := ikewardMachine()
	m.Results[ikewardBin+" status"] = effecttest.Result{}
	if status, _, stderr := runIkeward(m, nil, "wait"); status != 0 || stderr != "" {
		t.Errorf("status = %d, stderr %q", status, stderr)
	}
}

// The run state's check is due where the VPN is down in a run with the secrets. A run without them
// installs no VPN, so there the check passes, and the run neither waits nor fails.
func TestIkewardWaitCheckWantsTheSecrets(t *testing.T) {
	m := ikewardMachine()
	m.Results[ikewardBin+" status"] = effecttest.Result{Err: exitCode(t, "1")}
	if status, _, stderr := runIkeward(m, ikewardProfile, "wait", "--check"); status != 1 || !strings.Contains(stderr, "to do: wait for the VPN") {
		t.Errorf("with the secrets: status = %d, stderr %q; want the wait to do", status, stderr)
	}
	if status, _, stderr := runIkeward(m, nil, "wait", "--check"); status != 0 || !strings.Contains(stderr, "waits for none") {
		t.Errorf("without the secrets: status = %d, stderr %q; want no wait, said", status, stderr)
	}
}
