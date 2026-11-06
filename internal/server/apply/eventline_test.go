package apply_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/zadykian/ws/host"
	"github.com/zadykian/ws/internal/server/apply"
	"github.com/zadykian/ws/internal/setup"
	"github.com/zadykian/ws/step"
)

// fullDisk is a write that fails.
type fullDisk struct{ setup.Write }

func (fullDisk) Apply(context.Context) error { return errors.New("no space left on device") }

// A file's text stays off the line: a token that a person added by hand must not reach a journal.
func TestEventLineHoldsNoFilesText(t *testing.T) {
	c := setup.Write{Path: "/a/.bashrc", Before: []byte("TOKEN=old\n"), After: []byte("new\n"), Exists: true}
	if !strings.Contains(c.String(), "TOKEN") {
		t.Fatalf("the change holds no text to keep off: %q", c)
	}
	for _, kind := range []step.Kind{step.Planned, step.Started, step.Done} {
		line, err := json.Marshal(apply.NewEventLine(step.Event{Kind: kind, Step: "Files", Changes: []step.Change{c}}))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(line), "TOKEN") || !strings.Contains(string(line), "/a/.bashrc") {
			t.Errorf("line = %s", line)
		}
	}
	err := host.Group{}.Apply(context.Background(), step.Plan{Changes: []step.Change{fullDisk{c}}})
	failed := apply.NewEventLine(step.Event{Kind: step.Failed, Step: "Files", Err: err}).Error
	if strings.Contains(failed, "TOKEN") || !strings.Contains(failed, "/a/.bashrc") || !strings.Contains(failed, "no space") {
		t.Errorf("error = %q", failed)
	}
}

type kinded string

func (c kinded) String() string      { return string(c) }
func (kinded) Kind() step.ChangeKind { return step.Install }

// An event comes back from its line as it went, whatever its kind, as a screen reads the fields.
func TestParseEventReadsWhatALineHolds(t *testing.T) {
	events := []step.Event{
		{Kind: step.Planned, Parent: "base", Step: "Packages", Changes: []step.Change{kinded("install git")}},
		{Kind: step.Started, Parent: "base", Step: "Packages", Index: 1, Check: true},
		{Kind: step.Done, Step: "a <b> & c", Index: 2, LeftOut: true},
		{Kind: step.Failed, Step: "Files", Index: 3, Err: errors.New("boom")},
		{Kind: step.Skipped, Step: "Later", Index: 4, After: "Files"},
	}
	for _, want := range events {
		line, err := json.Marshal(apply.NewEventLine(want))
		if err != nil {
			t.Fatal(err)
		}
		got, ok := apply.ParseEvent(string(line))
		if !ok {
			t.Fatalf("%s is no event", line)
		}
		if a, b := apply.NewEventLine(got), apply.NewEventLine(want); !reflect.DeepEqual(a, b) {
			t.Errorf("got %+v, want %+v", a, b)
		}
	}
}

// These are the lines of a run that aren't its events: they stay in its log.
func TestParseEventLeavesOtherLines(t *testing.T) {
	for _, line := range []string{
		"",
		"Packages: applying 2 change(s)",
		"PLAY RECAP",
		`{"level":"info","msg":"started"}`,
		`{"event":"started","step":"s","index":0,"pid":12}`,
		`{"event":"restart","step":"s","index":0}`,
		`{"event":"done","index":0}`,
		`{"event":"done","step":"s","index":0} and more`,
		`{"event":"done","step":"s","in`,
		`[1,2]`,
	} {
		if e, ok := apply.ParseEvent(line); ok {
			t.Errorf("%q reads as %+v", line, e)
		}
	}
}
