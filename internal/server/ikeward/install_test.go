package ikeward

import (
	"os/exec"
	"slices"
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
