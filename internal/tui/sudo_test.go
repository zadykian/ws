package tui

import (
	"context"
	"errors"
	"io"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/effect/effecttest"
)

// The commands of sudo, as the fake's table names them.
const (
	sudoLook  = "sudo -n -v"
	sudoCheck = "sudo -S -p  -v"
	sudoKept  = "sudo -n -- true"
)

// noTicket returns a machine whose sudo holds no ticket.
func noTicket() *effecttest.Machine {
	m := effecttest.NewMachine()
	m.Results[sudoLook] = effecttest.Result{Err: errors.New("sudo: a password is required")}
	return m
}

func TestSudoAsksNothingOfAValidTicket(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[sudoLook] = effecttest.Result{}
	s := &sudo{m: m, p: &effecttest.Prompt{T: t}}
	if err := s.ticket(t.Context()); err != nil {
		t.Fatal(err)
	}
	if want := []string{"run " + sudoLook}; !slices.Equal(m.Calls, want) || !s.held.Load() {
		t.Errorf("calls = %q, held = %v", m.Calls, s.held.Load())
	}
}

func TestSudoChecksThePasswordAnswered(t *testing.T) {
	m := noTicket()
	m.Results[sudoCheck], m.Results[sudoKept] = effecttest.Result{}, effecttest.Result{}
	p := &effecttest.Prompt{T: t, Hidden: []string{" a pass\tword "}}
	s := &sudo{m: m, p: p}
	if err := s.ticket(t.Context()); err != nil {
		t.Fatal(err)
	}
	if want := []string{" a pass\tword \n"}; !slices.Equal(m.Inputs, want) {
		t.Errorf("sudo read %q, want %q", m.Inputs, want)
	}
	if want := []string{sudoQuestion}; !slices.Equal(p.Asked, want) || !s.held.Load() {
		t.Errorf("asked = %q, held = %v", p.Asked, s.held.Load())
	}
}

func TestSudoEndsWhereARightPasswordLeavesNoTicket(t *testing.T) {
	m := noTicket()
	m.Results[sudoCheck] = effecttest.Result{}
	m.Results[sudoKept] = effecttest.Result{Err: errors.New("sudo: a password is required")}
	s := &sudo{m: m, p: &effecttest.Prompt{T: t, Hidden: []string{"right"}}}
	if err := s.ticket(t.Context()); !errors.Is(err, effect.ErrNoTicket) || s.held.Load() {
		t.Errorf("err = %v, held = %v", err, s.held.Load())
	}
}

func TestSudoStopsAfterItsTries(t *testing.T) {
	m := noTicket()
	m.Results[sudoCheck] = effecttest.Result{Err: errors.New("exit status 1: Sorry, try again")}
	p := &effecttest.Prompt{T: t, Hidden: []string{"first", "second", "third"}}
	s := &sudo{m: m, p: p}
	err := s.ticket(t.Context())
	if err == nil || !strings.Contains(err.Error(), "Sorry, try again") || s.held.Load() {
		t.Fatalf("err = %v, held = %v", err, s.held.Load())
	}
	for _, secret := range []string{"first", "second", "third"} {
		if strings.Contains(err.Error(), secret) {
			t.Errorf("the error holds a password: %v", err)
		}
	}
	want := []string{sudoQuestion, sudoRefused + sudoQuestion, sudoRefused + sudoQuestion}
	if !slices.Equal(p.Asked, want) {
		t.Errorf("asked = %q, want %q", p.Asked, want)
	}
}

func TestSudoEndsWithTheQuestion(t *testing.T) {
	m := noTicket()
	b, _ := testBridge(io.Discard)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	s := &sudo{m: m, p: b}
	if err := s.ticket(ctx); !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v", err)
	}
	if want := []string{"run " + sudoLook}; !slices.Equal(m.Calls, want) {
		t.Errorf("calls = %q, want %q", m.Calls, want)
	}
}

// lookedMachine counts sudo's looks, which come from hold's goroutine, and refuses them once lapsed.
type lookedMachine struct {
	effect.Machine
	looks  atomic.Int32
	lapsed atomic.Bool
}

func (m *lookedMachine) Run(context.Context, string, ...string) ([]byte, error) {
	m.looks.Add(1)
	if m.lapsed.Load() {
		return nil, errors.New("sudo: a password is required")
	}
	return nil, nil
}

func TestSudoHoldsATicketItFound(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		m := &lookedMachine{}
		s := &sudo{m: m}
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		go s.hold(ctx)
		looks := func(after time.Duration) int32 {
			time.Sleep(after)
			synctest.Wait()
			return m.looks.Load()
		}
		if n := looks(3 * sudoRenewal); n != 0 {
			t.Errorf("%d looks with no ticket", n)
		}
		if err := s.ticket(ctx); err != nil {
			t.Fatal(err)
		}
		if n := looks(2 * sudoRenewal); n != 3 {
			t.Errorf("%d looks past the ticket and two renewals", n)
		}
		m.lapsed.Store(true)
		if n := looks(3 * sudoRenewal); n != 4 || s.held.Load() {
			t.Errorf("%d looks past a lapsed ticket, held = %v", n, s.held.Load())
		}
	})
}
