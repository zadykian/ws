package tui

import (
	"errors"
	"slices"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

// The commands of sudo, as the fake's table names them.
const (
	sudoLook  = "sudo -n -v"
	sudoCheck = "sudo -S -p  -v"
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
	if want := []string{"run " + sudoLook}; !slices.Equal(m.Calls, want) {
		t.Errorf("calls = %q, want %q", m.Calls, want)
	}
}

func TestSudoChecksThePasswordAnswered(t *testing.T) {
	m := noTicket()
	m.Results[sudoCheck] = effecttest.Result{}
	p := &effecttest.Prompt{T: t, Hidden: []string{" a pass\tword "}}
	s := &sudo{m: m, p: p}
	if err := s.ticket(t.Context()); err != nil {
		t.Fatal(err)
	}
	if want := []string{" a pass\tword \n"}; !slices.Equal(m.Inputs, want) {
		t.Errorf("sudo read %q, want %q", m.Inputs, want)
	}
	if want := []string{sudoQuestion}; !slices.Equal(p.Asked, want) {
		t.Errorf("asked = %q, want %q", p.Asked, want)
	}
}
