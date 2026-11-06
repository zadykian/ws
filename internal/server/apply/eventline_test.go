package apply_test

import (
	"context"
	"encoding/json"
	"errors"
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
