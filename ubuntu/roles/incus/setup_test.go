package incus

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

// exit is a command's error with the exit status that ws reads.
type exit int

func (e exit) Error() string { return fmt.Sprintf("exit status %d", int(e)) }
func (e exit) ExitCode() int { return int(e) }

const runningLab = `[{"name":"lab","status":"Running","profiles":["ws"]}]`

// key is the command line of incus as the fake machine keys it.
func key(args []string) string { return "incus " + strings.Join(args, " ") }

func TestAFileThatTheGuestHoldsAlikeIsNotPushed(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/usr/local/bin/ws"] = []byte("binary")
	m.Results[guestList] = effecttest.Result{Out: []byte(runningLab)}
	sum := fmt.Sprintf("%x  /usr/local/bin/ws\n", sha256.Sum256([]byte("binary")))
	m.Results[key(inGuest("lab", sumScript, "/usr/local/bin/ws"))] = effecttest.Result{Out: []byte(sum)}
	p := Pushed{M: m, Guest: "lab", From: "/usr/local/bin/ws", To: "/usr/local/bin/ws", Mode: "0755"}
	if changes, err := p.Plan(t.Context()); err != nil || len(changes) != 0 {
		t.Errorf("changes = %v, err = %v, want none", changes, err)
	}
}

func TestAFileThatDiffersOrIsMissingInTheGuestIsPushedAsRoots(t *testing.T) {
	for name, res := range map[string]effecttest.Result{
		"differs": {Out: []byte("0000  /etc/ws/values.yml\n")},
		"missing": {Err: exit(missing)},
	} {
		m := effecttest.NewMachine()
		m.Files["/etc/ws/guests/lab/values.yml"] = []byte("WS_X: 1\n")
		m.Results[guestList] = effecttest.Result{Out: []byte(runningLab)}
		m.Results[key(inGuest("lab", sumScript, "/etc/ws/values.yml"))] = res
		m.Results["incus file push --create-dirs --uid 0 --gid 0 --mode 0600 /etc/ws/guests/lab/values.yml lab/etc/ws/values.yml"] = effecttest.Result{}
		p := Pushed{M: m, Guest: "lab", From: "/etc/ws/guests/lab/values.yml", To: "/etc/ws/values.yml", Mode: "0600"}
		changes, err := p.Plan(t.Context())
		if err != nil || len(changes) != 1 {
			t.Fatalf("%s: changes = %v, err = %v", name, changes, err)
		}
		if err := changes[0].Apply(t.Context()); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

func TestAGuestThatDoesNotRunHoldsNoFileAndIsNotRead(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/usr/local/bin/ws"] = []byte("binary")
	m.Results[guestList] = effecttest.Result{Out: []byte(`[{"name":"lab","status":"Stopped"}]`)}
	changes, err := Pushed{M: m, Guest: "lab", From: "/usr/local/bin/ws", To: "/usr/local/bin/ws", Mode: "0755"}.Plan(t.Context())
	if err != nil || len(changes) != 1 {
		t.Errorf("changes = %v, err = %v, want the push due", changes, err)
	}
}

func TestAnOptionalFileThatTheHostLacksIsLeftAlone(t *testing.T) {
	m := effecttest.NewMachine()
	p := Pushed{M: m, Guest: "lab", From: "/etc/ws/guests/lab/values.yml", To: "/etc/ws/values.yml", Mode: "0600", Optional: true}
	if changes, err := p.Plan(t.Context()); err != nil || len(changes) != 0 {
		t.Errorf("changes = %v, err = %v, want none", changes, err)
	}
	p.Optional = false
	if _, err := p.Plan(t.Context()); err == nil {
		t.Errorf("a file that the host lacks is no error where it is not optional")
	}
}

func readyKey() string { return key(inGuest("lab", readyScript)) }

func TestAGuestInSyncPlansNoSetup(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[guestList] = effecttest.Result{Out: []byte(runningLab)}
	m.Results[readyKey()] = effecttest.Result{}
	m.Results[key(inGuest("lab", planScript, wsInGuest))] = effecttest.Result{}
	if changes, err := (Setup{M: m, Name: "lab"}).Plan(t.Context()); err != nil || len(changes) != 0 {
		t.Errorf("changes = %v, err = %v, want none", changes, err)
	}
}

func TestAGuestWhoseOwnPlanIsDueOrWhoseWsIsMissingGetsASetup(t *testing.T) {
	for _, code := range []int{2, missing} {
		m := effecttest.NewMachine()
		m.Results[guestList] = effecttest.Result{Out: []byte(runningLab)}
		m.Results[readyKey()] = effecttest.Result{}
		m.Results[key(inGuest("lab", planScript, wsInGuest, "--secrets"))] = effecttest.Result{Err: exit(code)}
		m.Results["incus exec lab -- /usr/local/bin/ws ubuntu guest setup --secrets"] = effecttest.Result{}
		changes, err := Setup{M: m, Name: "lab", Secrets: true}.Plan(t.Context())
		if err != nil || len(changes) != 1 {
			t.Fatalf("exit %d: changes = %v, err = %v", code, changes, err)
		}
		if err := changes[0].Apply(t.Context()); err != nil {
			t.Errorf("exit %d: %v", code, err)
		}
		if !slices.Contains(m.Calls, "run incus exec lab -- /usr/local/bin/ws ubuntu guest setup --secrets") {
			t.Errorf("exit %d: calls %q lack the setup with --secrets", code, m.Calls)
		}
	}
}

func TestAGuestPlanThatFailsOtherwiseFailsTheRun(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[guestList] = effecttest.Result{Out: []byte(runningLab)}
	m.Results[readyKey()] = effecttest.Result{}
	boom := errors.New("boom")
	m.Results[key(inGuest("lab", planScript, wsInGuest))] = effecttest.Result{Err: boom}
	if _, err := (Setup{M: m, Name: "lab"}).Plan(t.Context()); !errors.Is(err, boom) {
		t.Errorf("err = %v, want the plan's own error", err)
	}
}

func TestAGuestThatCloudInitFailedInIsNotSetUp(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[guestList] = effecttest.Result{Out: []byte(runningLab)}
	m.Results[readyKey()] = effecttest.Result{Err: exit(1)}
	if _, err := (Setup{M: m, Name: "lab"}).Plan(t.Context()); err == nil || !strings.Contains(err.Error(), "cloud-init failed") {
		t.Errorf("err = %v, want cloud-init's failure named", err)
	}
	m.Results[readyKey()] = effecttest.Result{Err: exit(2)}
	m.Results[key(inGuest("lab", planScript, wsInGuest))] = effecttest.Result{}
	if _, err := (Setup{M: m, Name: "lab"}).Plan(t.Context()); err != nil {
		t.Errorf("a cloud-init that ended recoverably, status 2, is over: %v", err)
	}
}

func TestAStoppedGuestAndAForcedSetupAreDue(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[guestList] = effecttest.Result{Out: []byte(`[{"name":"lab","status":"Stopped"}]`)}
	if changes, err := (Setup{M: m, Name: "lab"}).Plan(t.Context()); err != nil || len(changes) != 1 {
		t.Errorf("stopped: changes = %v, err = %v", changes, err)
	}
	if changes, err := (Setup{M: effecttest.NewMachine(), Name: "lab", Force: true}).Plan(t.Context()); err != nil || len(changes) != 1 {
		t.Errorf("forced: changes = %v, err = %v", changes, err)
	}
}

// A guest's root writes what the operator's terminal prints, and an escape sequence of it, such as
// OSC 52, writes the operator's clipboard.
func TestWhatAGuestPrintsKeepsNoControlCharacter(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[guestList] = effecttest.Result{Out: []byte(runningLab)}
	m.Results[readyKey()] = effecttest.Result{}
	m.Results["incus exec lab -- /usr/local/bin/ws ubuntu guest setup"] = effecttest.Result{Out: []byte("step\tok\n\x1b]52;c;ZXZpbA==\a done\u009b\n")}
	var shown bytes.Buffer
	changes, err := Setup{M: m, Name: "lab", Force: true, Out: effect.NewTerminal(&shown)}.Plan(t.Context())
	if err != nil || len(changes) != 1 {
		t.Fatalf("changes = %v, err = %v", changes, err)
	}
	if err := changes[0].Apply(t.Context()); err != nil {
		t.Fatal(err)
	}
	if got, want := shown.String(), "step\tok\n]52;c;ZXZpbA== done\n"; got != want {
		t.Errorf("shown %q, want %q", got, want)
	}
}

func TestAnErrorThatCarriesAGuestsBytesKeepsNoControlCharacter(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[guestList] = effecttest.Result{Out: []byte(runningLab)}
	m.Results[readyKey()] = effecttest.Result{}
	boom := errors.New("boom \x1b]52;c;ZXZpbA==\a")
	m.Results[key(inGuest("lab", planScript, wsInGuest))] = effecttest.Result{Err: boom}
	_, err := Setup{M: m, Name: "lab"}.Plan(t.Context())
	if !errors.Is(err, boom) || strings.ContainsAny(err.Error(), "\x1b\a") {
		t.Errorf("err = %q, want the plan's own error with no control character", err)
	}
}

// hanging is a machine whose guests never answer an exec.
type hanging struct{ *effecttest.Machine }

func (h hanging) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	if len(args) > 0 && args[0] == "exec" {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return h.Machine.Run(ctx, name, args...)
}

// A guest whose root stalls its boot check would hold up the guests and the units after it.
func TestAGuestThatNeverAnswersEndsTheWaitAtItsLimit(t *testing.T) {
	old := readyTimeout
	readyTimeout = 20 * time.Millisecond
	t.Cleanup(func() { readyTimeout = old })
	m := effecttest.NewMachine()
	m.Results[guestList] = effecttest.Result{Out: []byte(runningLab)}
	_, err := Setup{M: hanging{m}, Name: "lab"}.Plan(t.Context())
	if err == nil || !strings.Contains(err.Error(), "did not answer") {
		t.Errorf("err = %v, want the wait ended and the guest named", err)
	}
}
