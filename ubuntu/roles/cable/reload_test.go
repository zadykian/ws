package cable

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/ubuntu/roles/cable/internal/netplan"
)

const (
	// cablePath is the cable's file, and liveFile its text on the machine.
	cablePath = netplan.Path
	liveFile  = "the live file\n"
)

// planReload plans r and returns its headlines.
func planReload(t *testing.T, r Reload) []string {
	changes, err := r.Plan(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, c := range changes {
		got = append(got, step.Headline(c))
	}
	return got
}

func TestReloadRunsGenerateThenNetworkctlOnce(t *testing.T) {
	m, u := portMachine()
	m.Files[cablePath] = []byte(liveFile)
	r := Reload{M: m, U: u}
	want := []string{"write networkd's files from netplan's", "reload networkd, which sets up again each link whose file changed",
		"note the cable's file that is in force"}
	if got := planReload(t, r); !slices.Equal(got, want) {
		t.Fatalf("plan = %q, want %q", got, want)
	}
	m.Results["netplan generate"], m.Results["networkctl reload"] = effecttest.Result{}, effecttest.Result{}
	changes, _ := r.Plan(t.Context())
	for _, c := range changes {
		if err := c.Apply(t.Context()); err != nil {
			t.Fatal(err)
		}
	}
	gen, rel := slices.Index(m.Calls, "root netplan generate"), slices.Index(m.Calls, "root networkctl reload")
	if gen < 0 || rel < gen {
		t.Errorf("want generate, then reload, in %q", m.Calls)
	}
	if got := planReload(t, r); got != nil {
		t.Errorf("second plan = %q, want none", got)
	}
}

func TestReloadLeavesOtherNetplanFilesAlone(t *testing.T) {
	m, u := portMachine()
	m.Files["/etc/netplan/50-cloud-init.yaml"] = []byte("an uplink's edit, not yet applied\n")
	if got := planReload(t, Reload{M: m, U: u}); got != nil {
		t.Errorf("plan = %q, want none without the cable's file", got)
	}
	m.Files[cablePath] = []byte(liveFile)
	m.Files[inForceMark] = []byte(sumOfText(liveFile) + "\n")
	if got := planReload(t, Reload{M: m, U: u, Cable: cableOf}); got != nil {
		t.Errorf("plan = %q, want none where the file in force stays", got)
	}
	if slices.ContainsFunc(m.Calls, func(c string) bool { return strings.HasPrefix(c, "root ") }) {
		t.Errorf("calls %q, want no command", m.Calls)
	}
}

func TestReloadAfterTheFileWent(t *testing.T) {
	m, u := portMachine()
	m.Files[inForceMark] = []byte(sumOfText(liveFile) + "\n")
	changes, err := Reload{M: m, U: u}.Plan(t.Context())
	if err != nil || len(changes) != 3 {
		t.Fatalf("plan = %v, %v; want the reload and the mark", changes, err)
	}
	m.Results["netplan generate"] = effecttest.Result{}
	m.Results["networkctl reload"] = effecttest.Result{Err: errors.New("networkd refused")}
	for _, c := range changes {
		if err := c.Apply(t.Context()); err != nil {
			break
		}
	}
	if _, ok := m.Files[inForceMark]; !ok {
		t.Fatal("a failed reload removed the mark")
	}
	m.Results["networkctl reload"] = effecttest.Result{}
	changes, _ = Reload{M: m, U: u}.Plan(t.Context())
	for _, c := range changes {
		if err := c.Apply(t.Context()); err != nil {
			t.Fatal(err)
		}
	}
	if _, ok := m.Files[inForceMark]; ok {
		t.Error("the mark stays after the reload")
	}
}

func TestReloadRefusesAnotherEntryOnThePort(t *testing.T) {
	m, u := portMachine()
	m.Files[cablePath] = []byte(liveFile)
	m.Results["netplan get --root-dir "+scratchDir] = effecttest.Result{Out: []byte("network:\n  ethernets:\n    enp5s0: {}\n    lan:\n      match:\n        name: enp*\n")}
	changes, err := Reload{M: m, U: u, Cable: cableOf}.Plan(t.Context())
	if err == nil || changes != nil || !strings.Contains(err.Error(), "netplan's entry lan fits enp5s0") {
		t.Errorf("plan = %v, %v; want the entry that takes the port refused", changes, err)
	}
	if !slices.Contains(m.Calls, "read root "+cablePath) {
		t.Errorf("calls %q, want the live file checked", m.Calls)
	}
}

// sumOfText is the SHA-256 of text, in hex.
func sumOfText(text string) string { return fmt.Sprintf("%x", sha256.Sum256([]byte(text))) }
