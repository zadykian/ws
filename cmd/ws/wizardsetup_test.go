package main

import (
	"context"
	"io"
	"slices"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/step"
)

// lastEvents keeps the steps that end, each after its parent.
type lastEvents []string

func (l *lastEvents) Observe(e step.Event) {
	if e.Kind >= step.Done {
		*l = append(*l, e.Parent+"/"+e.Step)
	}
}

func TestWizardSetupPassesTheObserverOn(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/h/mac.conf"] = []byte("WS_USER=ada\n")
	p := withEnv(fakeProcess(), map[string]string{"HOME": "/h"})
	p.machine = m
	var got lastEvents
	s := wizardSetup(wizardEffects{p: p, text: io.Discard, out: &effecttest.Out{}, obs: &got}, "/h/mac.conf")
	// The fake machine fails ssh's plan, and each step ends all the same.
	_, _ = s.Run(context.Background(), true)
	var want lastEvents
	for _, name := range setupStepNames(p) {
		want = append(want, s.Name()+"/"+name)
	}
	if !slices.Equal(got, want) {
		t.Errorf("steps = %q, want %q", got, want)
	}
}
