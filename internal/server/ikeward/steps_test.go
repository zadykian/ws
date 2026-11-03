package ikeward

import (
	"crypto/sha256"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/internal/secrets"
	"github.com/zadykian/ws/internal/server/values"
	"github.com/zadykian/ws/step"
)

// held holds the profile's files, as the secrets of a machine with a VPN do.
var held = secrets.Secrets{File: secrets.Files{VPNP12: []byte("p12"), VPNSswan: []byte("sswan")}}

func TestStepsAreNoneWithoutAProfile(t *testing.T) {
	if got := steps(values.Ikeward{}, secrets.Secrets{}, effecttest.NewMachine(), &effecttest.Out{}, 1, time.Millisecond); got != nil {
		t.Errorf("steps = %v, want none", got)
	}
}

func TestStepsWaitOnTheirPredecessorsInOrder(t *testing.T) {
	ss := steps(good(), held, effecttest.NewMachine(), &effecttest.Out{}, 1, time.Millisecond)
	var names, after []string
	for _, s := range ss {
		names, after = append(names, s.Name()), append(after, s.WaitsOn())
	}
	wantNames := []string{downloadStep, forwardStep, installStep, statusStep}
	wantAfter := []string{"", "", downloadStep, installStep}
	if !slices.Equal(names, wantNames) || !slices.Equal(after, wantAfter) {
		t.Errorf("steps = %q after %q, want %q after %q", names, after, wantNames, wantAfter)
	}
}

func TestTheDownloadStepRefusesBadValuesBeforeAnythingChanges(t *testing.T) {
	v := good()
	v.DNS = ""
	m := effecttest.NewMachine()
	_, err := step.Run(t.Context(), steps(v, held, m, &effecttest.Out{}, 1, time.Millisecond), &effecttest.Out{}, true)
	if err == nil || !strings.Contains(err.Error(), "ikeward_dns") {
		t.Errorf("error = %v, want one naming ikeward_dns", err)
	}
	if len(m.Calls) != 0 {
		t.Errorf("calls = %q, want none", m.Calls)
	}
}

func TestTheStepsRefuseAProfileThatTheSecretsLack(t *testing.T) {
	m := effecttest.NewMachine()
	ss := steps(good(), secrets.Secrets{}, m, &effecttest.Out{}, 1, time.Millisecond)
	_, err := step.Run(t.Context(), ss, &effecttest.Out{}, true)
	if err == nil || !strings.Contains(err.Error(), "no profile") {
		t.Errorf("error = %v, want one that names the missing profile", err)
	}
	if len(m.Calls) != 0 {
		t.Errorf("calls = %q, want none", m.Calls)
	}
}

func TestStepsApplyOnceAndThenPlanNothing(t *testing.T) {
	v := good()
	v.ForwardFrom = nil
	m := effecttest.NewMachine()
	m.Dirs["/run/systemd/system"] = 0o755
	m.Files[ipForward] = []byte("1\n")
	m.Results["dpkg --print-architecture"] = effecttest.Result{Out: []byte("amd64\n")}
	m.Results["rm -rf -- "+sysctlPath] = effecttest.Result{}
	m.Results[binary+" status"] = effecttest.Result{}
	i := Install{Values: v}
	dry := binary + " " + strings.Join(append(i.args(profile{name: v.Profile}), "--dry-run", "--exit-code"), " ")
	m.Results[dry] = effecttest.Result{}
	m.Pages[releases+"/download/v"+version+"/ikeward-linux-amd64"] = []byte("binary")
	// The fake checks the body against the pin, so the test pins its own.
	pin := sums["amd64"]
	sums["amd64"] = fmt.Sprintf("%x", sha256.Sum256([]byte("binary")))
	t.Cleanup(func() { sums["amd64"] = pin })
	ss := steps(v, held, m, &effecttest.Out{}, 1, time.Millisecond)
	out := &effecttest.Out{}
	if changed, err := step.Run(t.Context(), ss, out, false); err != nil || !changed {
		t.Fatalf("first run = %v, %v; want changes", changed, err)
	}
	if changed, err := step.Run(t.Context(), ss, out, false); err != nil || changed {
		t.Errorf("second run = %v, %v; want nothing to change", changed, err)
	}
}
