package ikeward

import (
	"errors"
	"os/exec"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/server/values"
)

const installArgs = binary + " install --p12 " + runDir + "/vpnclient.p12 --sswan " + runDir +
	"/vpnclient.sswan --dns 10.20.0.53 --domain=example.internal --floor=10.20.0.0/16"

// good are the values of a machine with a VPN, which the values' check takes.
func good() values.Ikeward {
	return values.Ikeward{Profile: "/root/vpn/vpnclient", DNS: "10.20.0.53",
		Domains: []string{"example.internal"}, ForwardFrom: []string{"10.99.0.2"}}
}

// exitError is the error of a command that exits with code.
func exitError(t *testing.T, code string) error {
	_, err := exec.Command("sh", "-c", "exit "+code).Output() //nolint:gosec // a test runs a shell that exits
	if err == nil {
		t.Fatal("exit " + code + " succeeded")
	}
	return err
}

func installOf(m *effecttest.Machine) Install {
	m.Dirs["/run/systemd/system"] = 0o755
	v := good()
	v.Floors = []string{"10.20.0.0/16"}
	return Install{M: m, Out: &effecttest.Out{}, Values: v, InRun: true,
		Profile: Profile{P12: []byte("p12"), Sswan: []byte("sswan"), Password: "hunter2"}}
}

func TestRunPassesThePasswordOnStdinAndReadsExitFourAsAChange(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[installArgs+" --dry-run --exit-code"] = effecttest.Result{Out: []byte("write /etc/x\n"), Err: exitError(t, "4")}
	i := installOf(m)
	code, out, err := i.run(t.Context(), i.profile(), "--dry-run")
	if code != exitChanged || out != "write /etc/x\n" || err != nil {
		t.Errorf("run = %d, %q, %v; want 4 with the dry run's lines", code, out, err)
	}
	if !slices.Equal(m.Inputs, []string{"hunter2"}) {
		t.Errorf("inputs = %q, want the password", m.Inputs)
	}
}

func TestRunFailsOnAnyOtherExit(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[installArgs+" --exit-code"] = effecttest.Result{Err: exitError(t, "1")}
	i := installOf(m)
	if _, _, err := i.run(t.Context(), i.profile()); err == nil {
		t.Error("exit 1 passed")
	}
}

func TestInstallRunsWhereTheDryRunExitsFourAndPassesThePasswordOnStdin(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[installArgs+" --dry-run --exit-code"] = effecttest.Result{Out: []byte("write /etc/x\n"), Err: exitError(t, "4")}
	m.Results[installArgs+" --exit-code"] = effecttest.Result{Err: exitError(t, "4")}
	i := installOf(m)
	changes, err := i.Plan(t.Context())
	if err != nil || len(changes) != 1 || !strings.Contains(changes[0].String(), "write /etc/x") {
		t.Fatalf("plan = %v, %v; want the install with the dry run's lines", changes, err)
	}
	if err := changes[0].Apply(t.Context()); err != nil {
		t.Fatal(err)
	}
	if want := []string{"hunter2", "hunter2"}; !slices.Equal(m.Inputs, want) {
		t.Errorf("inputs = %q, want the password twice", m.Inputs)
	}
	if strings.Contains(strings.Join(m.Calls, "\n"), "hunter2") {
		t.Error("the password is in a command")
	}
}

func TestInstallPlansNothingWhereTheDryRunExitsZero(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[installArgs+" --dry-run --exit-code"] = effecttest.Result{}
	if changes, err := installOf(m).Plan(t.Context()); err != nil || len(changes) != 0 {
		t.Errorf("plan = %v, %v; want nothing", changes, err)
	}
}

func TestInstallShowsIkewardsErrorAndLeavesNoFiles(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[installArgs+" --dry-run --exit-code"] = effecttest.Result{Err: errors.Join(errors.New("bad profile"), exitError(t, "1"))}
	_, err := installOf(m).Plan(t.Context())
	if err == nil || !strings.Contains(err.Error(), "bad profile") {
		t.Errorf("error = %v, want ikeward's", err)
	}
	if !slices.Contains(m.Calls, "root rm -rf -- "+runDir) {
		t.Errorf("calls = %q, lack the removal of the files", m.Calls)
	}
}

// A run without the secrets env sets none of the profile's variables: the VPN stays untouched.
func TestPlanWithoutSecretsKeepsTheVPN(t *testing.T) {
	m := effecttest.NewMachine()
	i := installOf(m)
	i.Profile, i.InRun = Profile{}, false
	out := i.Out.(*effecttest.Out)
	if changes, err := i.Plan(t.Context()); err != nil || len(changes) != 0 || len(out.Lines) != 1 ||
		!strings.Contains(out.Lines[0], "the VPN stays as it is") {
		t.Errorf("plan = %v, %v, said %q; want nothing and the reason", changes, err, out.Lines)
	}
	if slices.ContainsFunc(m.Calls, func(c string) bool { return strings.Contains(c, binary) }) || len(m.Inputs) != 0 {
		t.Errorf("calls = %q, %d inputs; want no run of ikeward", m.Calls, len(m.Inputs))
	}
}

func TestReadyWithoutTheRunsSecrets(t *testing.T) {
	i := installOf(effecttest.NewMachine())
	i.Profile, i.InRun = Profile{}, false
	if err := i.Ready(); err != nil {
		t.Errorf("ready = %v, want nil: the run keeps the VPN", err)
	}
}

// A run with the secrets whose host has no profile sets its variables empty: the guard stops it,
// naming the variables alone.
func TestReadyFailsOnAnEmptyProfile(t *testing.T) {
	for _, p := range []Profile{{}, {P12: []byte("p12")}, {Sswan: []byte("sswan"), Password: "hunter2"}} {
		i := installOf(effecttest.NewMachine())
		i.Profile = p
		err := i.Ready()
		if err == nil || !strings.Contains(err.Error(), P12Var+" or "+SswanVar+" is empty") {
			t.Errorf("ready = %v, want the variables named", err)
			continue
		}
		if slices.ContainsFunc([]string{"hunter2", "p12", "sswan"}, func(v string) bool { return strings.Contains(err.Error(), v) }) {
			t.Errorf("ready's error %q holds a value", err)
		}
		if _, err := i.Plan(t.Context()); err == nil {
			t.Error("plan passed on an empty profile")
		}
	}
	if err := installOf(effecttest.NewMachine()).Ready(); err != nil {
		t.Errorf("ready = %v on a whole profile", err)
	}
}

func TestInstallSaysThatSystemdIsNotRunning(t *testing.T) {
	m := effecttest.NewMachine()
	i := installOf(m)
	delete(m.Dirs, "/run/systemd/system")
	out := i.Out.(*effecttest.Out)
	if changes, err := i.Plan(t.Context()); err != nil || len(changes) != 0 || len(out.Lines) != 1 ||
		!strings.Contains(out.Lines[0], "systemd") {
		t.Errorf("plan = %v, %v, said %q; want nothing and the reason", changes, err, out.Lines)
	}
}
