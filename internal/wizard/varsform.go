package wizard

import (
	"context"
	"errors"
	"fmt"

	"github.com/zadykian/ws/internal/server/values"
)

// VarsForm is ws's host_vars as a form: the keys that a line holds, with what the file gives each.
// A table stays the file's to edit by hand. The file is root's on ws, so the files script reads it.
type VarsForm struct {
	// Files is the files script on ws.
	Files *Remote
	// file is what host_vars gave at the last load.
	file values.Values
}

// Load reads the file. A machine without one reads as the defaults.
func (f VarsForm) Load(ctx context.Context) (VarsForm, error) {
	text, err := f.Files.Do(ctx, "values")
	if err != nil {
		return f, errors.New("host_vars on ws: " + stderrOf(err))
	}
	if f.file, err = values.Parse(text); err != nil {
		return f, fmt.Errorf("host_vars on ws: %w", err)
	}
	return f, nil
}

// Fields is each key with what it reads as. A field holds no default, as an empty one takes the key
// out of the file, and its help names the default in place of a list's whole length.
func (f VarsForm) Fields() []Field {
	defaults := values.Default()
	keys := values.Keys()
	fields := make([]Field, 0, len(keys))
	for _, k := range keys {
		field := Field{Key: k.Name, Text: k.Kind.String(), Value: f.file.Text(k), Valid: k.Check}
		if value := defaults.Text(k); value != "" && k.Kind != values.List {
			field.Text += ", default " + value
		}
		if k.Kind == values.Switch {
			field.Options = []string{"true", "false"}
		}
		fields = append(fields, field)
	}
	return fields
}
