package role_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/role"
	"github.com/zadykian/ws/engine/core/step"
)

// userFile is a change to a file of the user's: String quotes the old text, and Excerpt leaves it out.
type userFile struct{ path, before, after string }

func (c userFile) String() string  { return c.path + "\n-" + c.before + "+" + c.after }
func (c userFile) Excerpt() string { return c.path + "\n+" + c.after }
func (c userFile) Title() string   { return "would write " + c.path }

// fullDisk is a write that fails.
type fullDisk struct{ userFile }

func (fullDisk) Apply(context.Context) error { return errors.New("no space left on device") }

// A file's text stays off the line: a token that a person added by hand must not reach a journal.
func TestEventLineHoldsNoFilesText(t *testing.T) {
	c := userFile{path: "/a/.bashrc", before: "TOKEN=old\n", after: "new\n"}
	if !strings.Contains(c.String(), "TOKEN") {
		t.Fatalf("the change holds no text to keep off: %q", c)
	}
	for _, kind := range []step.Kind{step.Planned, step.Started, step.Done} {
		line, err := json.Marshal(role.NewEventLine(step.Event{Kind: kind, Step: "Files", Changes: []step.Change{c}}))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(line), "TOKEN") || !strings.Contains(string(line), "/a/.bashrc") {
			t.Errorf("line = %s", line)
		}
	}
	err := resource.Group{}.Apply(context.Background(), step.Plan{Changes: []step.Change{fullDisk{c}}})
	failed := role.NewEventLine(step.Event{Kind: step.Failed, Step: "Files", Err: err}).Error
	if strings.Contains(failed, "TOKEN") || !strings.Contains(failed, "/a/.bashrc") || !strings.Contains(failed, "no space") {
		t.Errorf("error = %q", failed)
	}
}
