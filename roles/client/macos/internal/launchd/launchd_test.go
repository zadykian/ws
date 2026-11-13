package launchd_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/internal/launchd"
)

func TestLaunchctlSaysItsKind(t *testing.T) {
	if got := step.KindOf(launchd.Launchctl{}); got != step.Restart {
		t.Errorf("kind = %s, want %s", got, step.Restart)
	}
}

func launchctl(m *effecttest.Machine, out *effecttest.Out, args ...string) launchd.Launchctl {
	return launchd.Launchctl{M: m, Out: out, Sleep: func(time.Duration) {}, Args: args}
}

func TestLaunchctlSaysWhatItRan(t *testing.T) {
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	m.Results["launchctl kickstart -k system/x"] = effecttest.Result{}
	if err := launchctl(m, out, "kickstart", "-k", "system/x").Apply(context.Background()); err != nil ||
		len(out.Lines) != 1 || out.Lines[0] != "say: ran launchctl kickstart -k system/x" {
		t.Errorf("Apply = %v, said %q; want what it ran", err, out.Lines)
	}
}

func TestLaunchctlTriesABootstrapTenTimes(t *testing.T) {
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	tries := 0
	l := launchctl(m, out, "bootstrap", "system", "p")
	l.Sleep = func(time.Duration) { tries++ }
	err := l.Apply(context.Background())
	if err == nil || !strings.Contains(err.Error(), "launchctl bootstrap system p failed") || tries != 9 {
		t.Errorf("Apply = %v after %d sleeps; want the failure after 9", err, tries)
	}
}

func TestLaunchctlFallsBackWhereASignalIsRefused(t *testing.T) {
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	m.Results["launchctl kill SIGHUP system/x"] = effecttest.Result{Err: errors.New("not running")}
	m.Results["launchctl kickstart -k system/"+launchd.Label] = effecttest.Result{}
	l := launchctl(m, out, "kill", "SIGHUP", "system/x")
	l.Signal = true
	if err := l.Apply(context.Background()); err != nil || len(m.Calls) != 2 {
		t.Errorf("Apply = %v, calls %q; want a kickstart after the signal", err, m.Calls)
	}
}

func TestLaunchctlWarnsWhereTheRoutesSignalFails(t *testing.T) {
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	l := launchctl(m, out, "kill", "SIGHUP", "system/x")
	l.Routes = true
	if err := l.Apply(context.Background()); err != nil || len(out.Lines) != 1 || !strings.HasPrefix(out.Lines[0], "warn: cannot signal") {
		t.Errorf("Apply = %v, said %q; want a warning alone", err, out.Lines)
	}
}
