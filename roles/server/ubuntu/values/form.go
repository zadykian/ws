package values

import (
	"context"
	"errors"
	"fmt"
	"github.com/zadykian/ws/engine/core/form"
	"github.com/zadykian/ws/engine/core/remote"
	"maps"
	"slices"

	"github.com/zadykian/ws/engine/core/effect"
)

// Store is where the form reads and writes host_vars: ws's session, whose operations run on ws.
type Store interface {
	// Values reads the file, empty for a missing file.
	Values(ctx context.Context) ([]byte, error)
	// SetValues writes each KEY=VALUE pair of pairs into the file in one login.
	SetValues(ctx context.Context, pairs []string) error
}

// Form is ws's host_vars as a form: the keys that a line holds, with what the file gives each.
// A table stays the file's to edit by hand. The file is root's on ws, so the files script reads it.
type Form struct {
	// Ws is the session whose operations read and write the file.
	Ws Store
	// file is what host_vars gave at the last load.
	file Values
}

// Load reads the file. A machine without one reads as the defaults.
func (f Form) Load(ctx context.Context) (Form, error) {
	text, err := f.Ws.Values(ctx)
	if err != nil {
		return f, errors.New("host_vars on ws: " + remote.Stderr(err))
	}
	if f.file, err = Parse(text); err != nil {
		return f, fmt.Errorf("host_vars on ws: %w", err)
	}
	return f, nil
}

// Fields is each key with what it reads as. A field holds no default, as an empty one takes the key
// out of the file, and its help names the default in place of a list's whole length.
func (f Form) Fields() []form.Field {
	defaults := Default()
	keys := Keys()
	fields := make([]form.Field, 0, len(keys))
	for _, k := range keys {
		field := form.Field{Key: k.Name, Text: k.Kind.String(), Value: f.file.Text(k), Valid: k.Check}
		if value := defaults.Text(k); value != "" && k.Kind != List {
			field.Text += ", default " + value
		}
		if k.Kind == Switch {
			field.Options = []string{"true", "false"}
		}
		fields = append(fields, field)
	}
	return fields
}

// Save writes the edits in one login, each key's check being ws's, and reads the file back. A failed
// one can leave the first keys written, which a second save writes again as they are. Nothing applies
// them until the next ws server ubuntu apply.
func (f Form) Save(ctx context.Context, out effect.Out, edits map[string]string) (Form, error) {
	if len(edits) == 0 {
		out.Say("host_vars holds these values already")
		return f, nil
	}
	var pairs []string
	for _, key := range slices.Sorted(maps.Keys(edits)) {
		pairs = append(pairs, key+"="+edits[key])
	}
	if err := f.Ws.SetValues(ctx, pairs); err != nil {
		return f, errors.New("host_vars on ws: " + remote.Stderr(err))
	}
	out.Say("wrote host_vars: the next ws server ubuntu apply takes it")
	return f.Load(ctx)
}
