package engine

import (
	"errors"
	"reflect"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
)

func TestLinesPrefixLines(t *testing.T) {
	out := Lines{Out: &effecttest.Out{}, Prog: "ws client macos apply", Seen: map[string]bool{}}
	out.Say("wrote %s", "f")
	out.Say("%s: %s", "Answers", "bad")
	out.Warn("w %d", 1)
	out.Warn("w %d", 1)
	want := []string{
		"say: ws client macos apply: wrote f", "say: ws client macos apply: Answers: bad", "say: ws client macos apply: warning: w 1",
	}
	if got := out.Out.(*effecttest.Out).Lines; !reflect.DeepEqual(got, want) {
		t.Errorf("lines = %q, want %q", got, want)
	}
}

func TestLinesObserveAPlanAndWhatAStepIsBehind(t *testing.T) {
	out := Lines{Out: &effecttest.Out{}, Prog: "ws client macos apply", Seen: map[string]bool{}}
	out.Change = func(c step.Change) (string, bool) {
		switch c.String() {
		case "bare\n":
			return "bare", true
		case "silent":
			return "", true
		}
		return "", false
	}
	plan := []step.Change{textChange("bare\n"), textChange("silent"), textChange("would run x")}
	out.Observe(step.Event{Kind: step.Planned, Step: "ssh", Changes: plan})
	out.Observe(step.Event{Kind: step.Started, Step: "ssh", Changes: plan})
	out.Observe(step.Event{Kind: step.Planned, Step: "ssh", Check: true, Changes: plan})
	out.Observe(step.Event{Kind: step.Skipped, Step: "git", After: "ssh"})
	out.Observe(step.Event{Kind: step.Skipped, Step: "zed"})
	out.Observe(step.Event{Kind: step.Planned, Step: "ssh", Check: true, Changes: plan, LeftOut: true})
	out.Observe(step.Event{Kind: step.Skipped, Step: "git", After: "ssh", LeftOut: true})
	out.Observe(step.Event{Kind: step.Skipped, Step: "cld", Err: errors.New("no file"), LeftOut: true})
	want := []string{
		"say: bare", "say: ws client macos apply: would run x",
		"say: ws client macos apply: git: after ssh", "say: ws client macos apply: warning: left out: no file",
	}
	if got := out.Out.(*effecttest.Out).Lines; !reflect.DeepEqual(got, want) {
		t.Errorf("lines = %q, want %q", got, want)
	}
}

func TestCauseDropsTheStepThatNamesIt(t *testing.T) {
	inner := errors.New("boom")
	if got := Cause(&stepError{inner}); got != inner { //nolint:errorlint // Cause returns the inner error itself, not one that wraps it
		t.Errorf("Cause = %v, want %v", got, inner)
	}
	if got := Cause(inner); got != inner { //nolint:errorlint // Cause returns the inner error itself, not one that wraps it
		t.Errorf("Cause of a plain error = %v", got)
	}
}

type stepError struct{ err error }

func (e *stepError) Error() string { return "step: " + e.err.Error() }

func (e *stepError) Unwrap() error { return e.err }
