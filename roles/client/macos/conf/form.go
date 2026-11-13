package conf

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/form"
)

// configureStage names the Configure screen in what a save says.
const configureStage = "Configure"

// wayKeys are the keys that an edit can hide. conf refuses one past the edit that hid it, as
// WS_LAN on a cloud VM, and a key that doesn't show has no field to clear it by.
var wayKeys = []string{"WS_LAN", "WS_LAN_ADDRESS", "WS_REMOTE", "WS_REMOTE_PORT", "WS_GIT_EMAIL"}

// Form is mac.conf and a server's file as a form. It shows the keys that the Answers stage would ask
// where the files lacked them all, as the edits so far lead it. A key that the stage never asks
// stays the file's. Each key is saved to its own file.
type Form struct {
	// Conf is mac.conf's path, and Server the server's file's.
	Conf   string
	Server string
	// Defaults is the text of the shell role's defaults, whose git identity ws's git takes.
	Defaults []byte
	M        effect.Machine
	// file is what the files held at the last load, joined.
	file Conf
}

// of returns the file of the scope, which says its refusals to out.
func (f Form) of(s Scope, out effect.Out) File {
	path := f.Conf
	if s == ServerFile {
		path = f.Server
	}
	return File{Path: path, Stage: configureStage, M: f.M, Out: out, Scope: s}
}

// Load reads both files, says each line that conf refuses, and fails on one: a form shows keys, and
// has no field for a line that is none. A refusal of keys that go together opens, as a form mends it.
func (f Form) Load(out effect.Out) (Form, error) {
	_, mac, macErr := f.of(MacFile, out).Load()
	_, server, serverErr := f.of(ServerFile, out).Load()
	if err := errors.Join(macErr, serverErr); err != nil {
		return f, err
	}
	f.file = join(mac, server)
	return f, nil
}

// walk runs the Answers stage's sequence, each question getting the value that the form holds, and
// returns the questions it reached and the values it ended on.
func (f Form) walk(edits map[string]string) ([]Question, map[string]string) {
	var asked []Question
	s := Sequence{
		Vals:  map[string]string{},
		Lacks: func(string) bool { return true },
		Ask: func(q Question) (string, error) {
			asked = append(asked, q)
			return f.wanted(q, edits), nil
		},
		Role: func(name string) string { return Default(f.Defaults, name) },
	}
	_ = s.Run() // ask returns no error, which is what would stop the walk.
	return asked, s.Vals
}

// wanted is the value that a save would leave for the key: an edit, or the file's, or the default of
// a key that needs a value. An optional key's default stays the wizard's offer, as a form shows the
// file and a value nobody typed would land unseen.
func (f Form) wanted(q Question, edits map[string]string) string {
	if raw, ok := edits[q.Key]; ok {
		return fieldValue(q, raw)
	}
	if v := f.file.Value(q.Key); v != "" || q.Need == "" {
		return v
	}
	return q.Def
}

// fieldValue reads what a field holds as the wizard reads an answer. But an empty field of an
// optional key is an empty value, which is what the form shows.
func fieldValue(q Question, raw string) string {
	if raw == "" && q.Need == "" {
		return ""
	}
	return q.Reply(raw)
}

// Fields returns the keys that show under the edits, in the order that the stage asks them.
func (f Form) Fields(edits map[string]string) []form.Field {
	asked, vals := f.walk(edits)
	fields := make([]form.Field, 0, len(asked))
	for _, q := range asked {
		field := form.Field{Key: q.Key, Text: q.Text, Value: f.file.Value(q.Key), Valid: func(raw string) error {
			if why := q.Refusal(fieldValue(q, raw), vals["WS_LAN"]); why != "" {
				return errors.New(why)
			}
			return nil
		}}
		switch {
		case q.Options() != nil:
			field.Options = q.Options()
		case q.Need != "":
			field.Default = q.Def
		case q.Def != "":
			field.Text += " (the wizard offers " + q.Def + ")"
		}
		fields = append(fields, field)
	}
	return fields
}

// settings lists what a save changes: each key that shows, and an empty value for one that holds a
// value and no longer has a place. The cable's keys never show, and go with the LAN.
func (f Form) settings(edits map[string]string) []Setting {
	asked, vals := f.walk(edits)
	var changes []Setting
	for _, q := range asked {
		if v := f.wanted(q, edits); v != f.file.Value(q.Key) {
			changes = append(changes, Setting{Key: q.Key, Value: v})
		}
	}
	gone := wayKeys
	if vals["WS_TARGET"] != "lan" {
		gone = slices.Concat(gone, []string{"WS_CABLE_SERVICE", "WS_CABLE_PORT"})
	}
	for _, key := range gone {
		if _, shows := vals[key]; !shows && f.file.Value(key) != "" {
			changes = append(changes, Setting{Key: key})
		}
	}
	return changes
}

// Save writes the changes, each to the file of its key, mac.conf first, and returns the form of what
// they then hold. It says the keys that it emptied, as some of them no longer show.
func (f Form) Save(ctx context.Context, out effect.Out, edits map[string]string) (Form, error) {
	changes := f.settings(edits)
	if len(changes) == 0 {
		out.Say("%s and %s hold these values already", f.Conf, f.Server)
		return f, nil
	}
	for _, s := range []Scope{MacFile, ServerFile} {
		var own []Setting
		for _, c := range changes {
			if KeyScope(c.Key) == s {
				own = append(own, c)
			}
		}
		if len(own) == 0 {
			continue
		}
		if err := f.of(s, out).SetAll(ctx, own...); err != nil {
			return f, err
		}
	}
	var emptied []string
	for _, c := range changes {
		if c.Value == "" {
			emptied = append(emptied, c.Key)
		}
	}
	if len(emptied) > 0 {
		out.Say("emptied %s", strings.Join(emptied, " "))
	}
	return f.Load(out)
}
