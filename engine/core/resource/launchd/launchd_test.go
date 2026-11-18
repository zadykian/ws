package launchd_test

import (
	"errors"
	"os/exec"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/resource/launchd"
)

// exited is the error of a command that ran and exited with a status other than 0.
func exited(t *testing.T, cmd *exec.Cmd) error {
	err := cmd.Run()
	if err == nil {
		t.Fatalf("%v passed, want an exit status", cmd.Args)
	}
	return err
}

// printed is launchctl print's output for a running daemon, as macOS 15 shows it.
const printed = `system/com.x.lab = {
	active count = 1
	path = /Library/LaunchDaemons/com.x.lab.plist
	type = LaunchDaemon
	state = running

	program = /usr/local/bin/ws
	arguments = {
		/usr/local/bin/ws
		run
		--server
		lab
	}

	inherited environment = {
		SSH_AUTH_SOCK => /private/tmp/x
	}

	pid = 6645
	immediate reason = speculative
	endpoints = {
		arguments = {
		}
	}
}
`

func ctl(m *effecttest.Machine) launchd.Launchctl {
	return launchd.Launchctl{M: m, Sleep: func(time.Duration) {}}
}

func TestPrintNotLoaded(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["launchctl print system/com.x.lab"] = effecttest.Result{Err: exited(t, exec.Command("sh", "-c", "exit 113"))}
	got, err := ctl(m).Print(t.Context(), "com.x.lab")
	if err != nil || got.Found {
		t.Errorf("Print = %+v, %v; want not found", got, err)
	}
	if !slices.Equal(m.Calls, []string{"run launchctl print system/com.x.lab"}) {
		t.Errorf("calls = %q, want print as the user", m.Calls)
	}
}

func TestPrintArguments(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["launchctl print system/com.x.lab"] = effecttest.Result{Out: []byte(printed)}
	got, err := ctl(m).Print(t.Context(), "com.x.lab")
	want := []string{"/usr/local/bin/ws", "run", "--server", "lab"}
	if err != nil || !got.Found || !slices.Equal(got.Arguments, want) || got.PID != 6645 {
		t.Errorf("Print = %+v, %v; want found, pid 6645 and arguments %q", got, err, want)
	}
}

func TestPrintFailsOnAnotherStatus(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["launchctl print system/com.x.lab"] = effecttest.Result{Err: exited(t, exec.Command("sh", "-c", "exit 1"))}
	if _, err := ctl(m).Print(t.Context(), "com.x.lab"); err == nil {
		t.Error("Print of exit 1 passed, want its error")
	}
}

func TestBootstrapRetries(t *testing.T) {
	m := effecttest.NewMachine()
	line := "launchctl bootstrap system /L/x.plist"
	m.Results[line] = effecttest.Result{Err: exited(t, exec.Command("sh", "-c", "exit 5"))}
	slept := 0
	l := launchd.Launchctl{M: m, Sleep: func(time.Duration) {
		if slept++; slept == 2 {
			m.Results[line] = effecttest.Result{}
		}
	}}
	if err := l.Bootstrap(t.Context(), "/L/x.plist"); err != nil || len(m.Calls) != 3 {
		t.Errorf("Bootstrap = %v after calls %q; want a pass at the third try", err, m.Calls)
	}
}

func TestBootstrapStopsAtTenTries(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["launchctl bootstrap system /L/x.plist"] = effecttest.Result{Err: exited(t, exec.Command("sh", "-c", "exit 5"))}
	err := ctl(m).Bootstrap(t.Context(), "/L/x.plist")
	if err == nil || !strings.Contains(err.Error(), "root's at mode 644") || len(m.Calls) != 10 {
		t.Errorf("Bootstrap = %v after %d calls; want the failure, with its hint, after 10", err, len(m.Calls))
	}
}

func TestBootstrapFailsAtOnceOnAnotherStatus(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["launchctl bootstrap system /L/x.plist"] = effecttest.Result{Err: exited(t, exec.Command("sh", "-c", "exit 1"))}
	if err := ctl(m).Bootstrap(t.Context(), "/L/x.plist"); err == nil || len(m.Calls) != 1 {
		t.Errorf("Bootstrap = %v after %d calls; want the failure at once", err, len(m.Calls))
	}
}

func TestBootoutGoneIsNil(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["launchctl print system/com.x.lab"] = effecttest.Result{Err: exited(t, exec.Command("sh", "-c", "exit 113"))}
	m.Results["launchctl bootout system/com.x.lab"] = effecttest.Result{Err: exited(t, exec.Command("sh", "-c", "exit 3"))}
	if err := ctl(m).Bootout(t.Context(), "com.x.lab"); err != nil {
		t.Errorf("Bootout of a daemon not loaded = %v, want nil", err)
	}
	m.Results["launchctl bootout system/com.x.lab"] = effecttest.Result{Err: exited(t, exec.Command("sh", "-c", "exit 1"))}
	if err := ctl(m).Bootout(t.Context(), "com.x.lab"); err == nil {
		t.Error("Bootout of exit 1 passed, want its error")
	}
}

// A daemon that print finds after its bootout is on its way out, and may still write its files. The
// bootout returns once print no longer finds it.
func TestBootoutWaitsUntilTheDaemonIsGone(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["launchctl bootout system/com.x.lab"] = effecttest.Result{}
	m.Results["launchctl print system/com.x.lab"] = effecttest.Result{Out: []byte(printed)}
	slept := 0
	l := launchd.Launchctl{M: m, Sleep: func(time.Duration) {
		if slept++; slept == 2 {
			m.Results["launchctl print system/com.x.lab"] = effecttest.Result{Err: exited(t, exec.Command("sh", "-c", "exit 113"))}
		}
	}}
	if err := l.Bootout(t.Context(), "com.x.lab"); err != nil || len(m.Calls) != 4 {
		t.Errorf("Bootout = %v after calls %q; want a pass at the third print", err, m.Calls)
	}
	m.Calls = nil
	m.Results["launchctl print system/com.x.lab"] = effecttest.Result{Out: []byte(printed)}
	err := ctl(m).Bootout(t.Context(), "com.x.lab")
	if err == nil || !strings.Contains(err.Error(), "launchd still has system/com.x.lab 30 s after its bootout") || len(m.Calls) != 31 {
		t.Errorf("Bootout = %v after %d calls; want the failure after 30 prints", err, len(m.Calls))
	}
}

func TestCallsAreRoot(t *testing.T) {
	m := effecttest.NewMachine()
	for _, line := range []string{"bootstrap system /L/x.plist", "bootout system/x", "kickstart -k system/x", "kill SIGHUP system/x"} {
		m.Results["launchctl "+line] = effecttest.Result{}
	}
	m.Results["launchctl print system/x"] = effecttest.Result{Err: exited(t, exec.Command("sh", "-c", "exit 113"))}
	l := ctl(m)
	ctx := t.Context()
	if err := errors.Join(l.Bootstrap(ctx, "/L/x.plist"), l.Bootout(ctx, "x"), l.Kickstart(ctx, "x"), l.Signal(ctx, "x", "SIGHUP")); err != nil {
		t.Fatal(err)
	}
	want := []string{"root launchctl bootstrap system /L/x.plist", "root launchctl bootout system/x", "run launchctl print system/x",
		"root launchctl kickstart -k system/x", "root launchctl kill SIGHUP system/x"}
	if !slices.Equal(m.Calls, want) {
		t.Errorf("calls = %q, want %q", m.Calls, want)
	}
}
