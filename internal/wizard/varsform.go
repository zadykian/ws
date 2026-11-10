package wizard

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/zadykian/ws/effect"
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
	text, err := f.Files.query(ctx, "values")
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

// Save writes the edits in one login, each key's check being ws's, and reads the file back. A failed
// one can leave the first keys written, which a second save writes again as they are. A pair goes as
// base64, as the files script says. Nothing applies them until the next ws server apply.
func (f VarsForm) Save(ctx context.Context, out effect.Out, edits map[string]string) (VarsForm, error) {
	if len(edits) == 0 {
		out.Say("host_vars holds these values already")
		return f, nil
	}
	op := []string{"values"}
	for _, key := range slices.Sorted(maps.Keys(edits)) {
		op = append(op, base64.StdEncoding.EncodeToString([]byte(key+"="+edits[key])))
	}
	if _, err := f.Files.Do(ctx, op...); err != nil {
		return f, errors.New("host_vars on ws: " + stderrOf(err))
	}
	out.Say("wrote host_vars: the next ws server apply takes it")
	return f.Load(ctx)
}
