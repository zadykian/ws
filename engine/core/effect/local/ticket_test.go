package local_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/effect/local"
)

// rootCalls are the ways through OSMachine that run a command as root.
var rootCalls = map[string]func(context.Context, local.OSMachine, string) error{
	"RunRoot": func(ctx context.Context, m local.OSMachine, _ string) error {
		_, err := m.RunRoot(ctx, "true")
		return err
	},
	"RunShown": func(ctx context.Context, m local.OSMachine, _ string) error {
		_, err := m.RunShown(ctx, true, "true")
		return err
	},
	"Pass": func(ctx context.Context, m local.OSMachine, _ string) error { return m.Pass(ctx, true, "true") },
	"PassForeground": func(ctx context.Context, m local.OSMachine, _ string) error {
		return m.PassForeground(ctx, true, "true")
	},
	"WriteRootFile": func(_ context.Context, m local.OSMachine, dir string) error {
		return m.WriteRootFile(filepath.Join(dir, "file"), []byte("x"), 0o600)
	},
}

func TestOSMachineTicketGoesBeforeACommandAsRoot(t *testing.T) {
	refused := errors.New("refused")
	for name, call := range rootCalls {
		t.Run(name, func(t *testing.T) {
			mark := filepath.Join(t.TempDir(), "mark")
			ran := func() bool { _, err := os.Stat(mark); return err == nil }
			m := local.OSMachine{Sudo: []string{"sh", "-c", `: >"$0"; exec "$@"`, mark},
				Owner: fmt.Sprintf("%d:%d", os.Getuid(), os.Getgid())}
			m.Ticket = func() error { return refused }
			if err := call(t.Context(), m, t.TempDir()); !errors.Is(err, refused) || ran() {
				t.Errorf("past a refusal: err = %v, ran = %v", err, ran())
			}
			m.Ticket = func() error {
				if ran() {
					t.Error("the command ran before its ticket")
				}
				return nil
			}
			if err := call(t.Context(), m, t.TempDir()); err != nil || !ran() {
				t.Errorf("past a ticket: err = %v, ran = %v", err, ran())
			}
		})
	}
}

func TestOSMachineTicketLeavesTheUsersCommands(t *testing.T) {
	m := local.OSMachine{Ticket: func() error { return errors.New("a ticket for the user's command") }}
	_, run := m.Run(t.Context(), "true")
	_, shown := m.RunShown(t.Context(), false, "true")
	if err := errors.Join(run, shown, m.Pass(t.Context(), false, "true"),
		m.PassForeground(t.Context(), false, "true")); err != nil {
		t.Error(err)
	}
}

func TestValidateRunsSudoAtTheTerminal(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["sudo -v"] = effecttest.Result{}
	if err := local.Validate(t.Context(), m); err != nil {
		t.Fatal(err)
	}
	if want := []string{"run sudo -v"}; !slices.Equal(m.Calls, want) {
		t.Errorf("calls = %q, want %q", m.Calls, want)
	}
}

func TestValidateTakesAMachinesTicket(t *testing.T) {
	refused := errors.New("refused")
	m := local.OSMachine{Sudo: []string{"false"}, Ticket: func() error { return refused }}
	if err := local.Validate(t.Context(), m); !errors.Is(err, refused) {
		t.Errorf("err = %v", err)
	}
}
