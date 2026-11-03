package ikeward

import (
	"errors"
	"os/exec"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/internal/server/secrets"
)

const installArgs = binary + " install --p12 " + runDir + "/vpnclient.p12 --sswan " + runDir +
	"/vpnclient.sswan --dns 10.20.0.53 --domain=example.internal --floor=10.20.0.0/16"

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
	return Install{M: m, Out: &effecttest.Out{}, Values: v, Secrets: secrets.Secrets{
		Ikeward: secrets.Ikeward{P12Password: "hunter2"},
		Files:   secrets.Files{P12: []byte("p12"), Sswan: []byte("sswan")}}}
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

func TestInstallSaysThatTheSecretsHoldNoProfile(t *testing.T) {
	i := installOf(effecttest.NewMachine())
	i.Secrets.Files = secrets.Files{}
	out := i.Out.(*effecttest.Out)
	if changes, err := i.Plan(t.Context()); err != nil || len(changes) != 0 || len(out.Lines) != 1 ||
		!strings.Contains(out.Lines[0], "no profile") {
		t.Errorf("plan = %v, %v, said %q; want nothing and the reason", changes, err, out.Lines)
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
