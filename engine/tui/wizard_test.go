package tui

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
)

// fakeScreens gives the loop the choices in turn, and keeps what it drew.
type fakeScreens struct {
	choices []QueueChoice
	drew    []string
}

func (f *fakeScreens) next(name string) (QueueChoice, error) {
	f.drew = append(f.drew, name)
	if len(f.choices) == 0 {
		return QueueChoice{}, errors.New("no choice left")
	}
	defer func() { f.choices = f.choices[1:] }()
	return f.choices[0], nil
}

func (f *fakeScreens) pick(context.Context, Round) (QueueChoice, error) { return f.next("lists") }
func (f *fakeScreens) reopen(context.Context, QueueChoice) (QueueChoice, error) {
	return f.next("lists again")
}
func (f *fakeScreens) state(_ context.Context, side Side, _ StateReader) error {
	f.drew = append(f.drew, "state of "+side)
	return nil
}

func (f *fakeScreens) configure(_ context.Context, sheets []Sheet) error {
	f.drew = append(f.drew, fmt.Sprintf("%d forms", len(sheets)))
	return nil
}

// fakeWizard counts its rounds and keeps what its works were given.
type fakeWizard struct {
	rounds, reads, done int
	ran                 []QueueChoice
}

func (f *fakeWizard) wizard() Wizard {
	end := func(context.Context) { f.done++ }
	return Wizard{
		Round: func() Round {
			f.rounds++
			return Round{
				Read: func(Side) (StateReader, func(context.Context)) { f.reads++; return nil, end },
				Run:  func(_ context.Context, c QueueChoice) error { f.ran = append(f.ran, c); return nil },
			}
		},
		Forms: func(context.Context) ([]Sheet, func(context.Context), error) { return []Sheet{{}, {}}, end, nil },
	}
}

// The state screen comes back to the lists with no scan, and the run takes what they chose then.
func TestWizardStateComesBackWithNoScan(t *testing.T) {
	run := QueueChoice{Next: NextRun, Only: []string{"Run"}}
	s := &fakeScreens{choices: []QueueChoice{{Next: NextState, Side: "ws"}, {Next: NextState, Side: "Mac"}, run}}
	f := &fakeWizard{}
	if err := f.wizard().loop(t.Context(), s); err != nil {
		t.Fatal(err)
	}
	want := []string{"lists", "state of ws", "lists again", "state of Mac", "lists again"}
	if !slices.Equal(s.drew, want) || f.rounds != 1 || f.reads != 2 || f.done != 2 {
		t.Errorf("drew %q in %d rounds, %d reads ended %d times", s.drew, f.rounds, f.reads, f.done)
	}
	if len(f.ran) != 1 || !slices.Equal(f.ran[0].Only, run.Only) {
		t.Errorf("ran %v", f.ran)
	}
}

// The forms lead to a new round, and a leave of its lists runs nothing.
func TestWizardFormsLeadToANewScan(t *testing.T) {
	s := &fakeScreens{choices: []QueueChoice{{Next: NextConfigure, Only: []string{"Run"}}, {}}}
	f := &fakeWizard{}
	if err := f.wizard().loop(t.Context(), s); err != nil {
		t.Fatal(err)
	}
	want := []string{"lists", "2 forms", "lists"}
	if !slices.Equal(s.drew, want) || f.rounds != 2 || f.done != 1 || len(f.ran) != 0 {
		t.Errorf("drew %q in %d rounds, forms ended %d times, ran %v", s.drew, f.rounds, f.done, f.ran)
	}
}

// A screen's error ends the loop, and forms that fail to open draw nothing.
func TestWizardEndsOnAnError(t *testing.T) {
	f := &fakeWizard{}
	if err := f.wizard().loop(t.Context(), &fakeScreens{}); err == nil || len(f.ran) != 0 {
		t.Errorf("err = %v, ran %v", err, f.ran)
	}
	w := f.wizard()
	w.Forms = func(context.Context) ([]Sheet, func(context.Context), error) { return nil, nil, errors.New("no conf") }
	s := &fakeScreens{choices: []QueueChoice{{Next: NextConfigure}}}
	if err := w.loop(t.Context(), s); err == nil || !slices.Equal(s.drew, []string{"lists"}) {
		t.Errorf("err = %v, drew %q", err, s.drew)
	}
}
