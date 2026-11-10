package wizard

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/internal/conf"
)

// configureStage names the Configure screen in what a save says.
const configureStage = "Configure"

// targets are WS_TARGET's values, in the order that the form offers them.
var targets = []string{"lan", "cloud", "mac-vm"}

// wayKeys are the keys that an edit can hide. conf refuses one past the edit that hid it, as
// WS_LAN on a cloud VM, and a key that doesn't show has no field to clear it by.
var wayKeys = []string{"WS_LAN", "WS_LAN_ADDRESS", "WS_REMOTE", "WS_REMOTE_PORT", "WS_GIT_EMAIL"}

// Field is a key of mac.conf as the Configure screen shows it.
type Field struct {
	Key, Text string
	// Value is what the file holds, and Default what an empty field gives.
	Value, Default string
	// Options are the values of a key that takes one of a few, and nil for a line.
	Options []string
	// Valid says why mac.conf can't take a value. Valid is pure, as it runs on each edit.
	Valid func(string) error
}

// ConfForm is mac.conf as a form: the keys that the Answers stage would ask where the file lacked
// them all, as the edits so far lead it. A key that the stage never asks stays the file's.
type ConfForm struct {
	// Conf is mac.conf's path.
	Conf string
	// Defaults is the text of the shell role's defaults, whose git identity ws's git takes.
	Defaults []byte
	M        effect.Machine
	// file is what mac.conf held at the last load.
	file conf.Conf
}

// Load reads mac.conf, says each line that conf refuses, and fails on one: a form shows keys, and
// has no field for a line that is none. A refusal of keys that go together opens, as a form mends it.
func (f ConfForm) Load(out effect.Out) (ConfForm, error) {
	_, c, err := loadConf(confFile{f.Conf, configureStage, f.M, out})
	if err != nil {
		return f, err
	}
	f.file = c
	return f, nil
}

// walk runs the Answers stage's sequence, each question getting the value that the form holds, and
// returns the questions it reached and the values it ended on.
func (f ConfForm) walk(edits map[string]string) ([]question, map[string]string) {
	var asked []question
	s := sequence{
		vals:  map[string]string{},
		lacks: func(string) bool { return true },
		ask: func(q question) (string, error) {
			asked = append(asked, q)
			return f.wanted(q, edits), nil
		},
		role: (&Answers{Defaults: f.Defaults}).role,
	}
	_ = s.run() // ask returns no error, which is what would stop the walk.
	return asked, s.vals
}

// wanted is the value that a save would leave for the key: an edit, or the file's, or the default of
// a key that needs a value. An optional key's default stays the wizard's offer, as a form shows the
// file and a value nobody typed would land unseen.
func (f ConfForm) wanted(q question, edits map[string]string) string {
	if raw, ok := edits[q.key]; ok {
		return fieldValue(q, raw)
	}
	if v := f.file.Value(q.key); v != "" || q.need == "" {
		return v
	}
	return q.def
}

// fieldValue reads what a field holds as the wizard reads an answer. But an empty field of an
// optional key is an empty value, which is what the form shows.
func fieldValue(q question, raw string) string {
	if raw == "" && q.need == "" {
		return ""
	}
	return q.reply(raw)
}

// Fields returns the keys that show under the edits, in the order that the stage asks them.
func (f ConfForm) Fields(edits map[string]string) []Field {
	asked, vals := f.walk(edits)
	fields := make([]Field, 0, len(asked))
	for _, q := range asked {
		field := Field{Key: q.key, Text: q.text, Value: f.file.Value(q.key), Valid: func(raw string) error {
			if why := q.refusal(fieldValue(q, raw), vals["WS_LAN"]); why != "" {
				return errors.New(why)
			}
			return nil
		}}
		switch {
		case q.key == "WS_TARGET":
			field.Options = targets
		case q.need != "":
			field.Default = q.def
		case q.def != "":
			field.Text += " (the wizard offers " + q.def + ")"
		}
		fields = append(fields, field)
	}
	return fields
}

// settings lists what a save changes: each key that shows, and an empty value for one that holds a
// value and no longer has a place. The cable's keys never show, and go with the LAN.
func (f ConfForm) settings(edits map[string]string) []setting {
	asked, vals := f.walk(edits)
	var changes []setting
	for _, q := range asked {
		if v := f.wanted(q, edits); v != f.file.Value(q.key) {
			changes = append(changes, setting{q.key, v})
		}
	}
	gone := wayKeys
	if vals["WS_TARGET"] != "lan" {
		gone = slices.Concat(gone, []string{"WS_CABLE_SERVICE", "WS_CABLE_PORT"})
	}
	for _, key := range gone {
		if _, shows := vals[key]; !shows && f.file.Value(key) != "" {
			changes = append(changes, setting{key, ""})
		}
	}
	return changes
}

// Save writes the changes to mac.conf as one and returns the form of what it then holds. It says the
// keys that it emptied, as some of them no longer show.
func (f ConfForm) Save(ctx context.Context, out effect.Out, edits map[string]string) (ConfForm, error) {
	changes := f.settings(edits)
	if len(changes) == 0 {
		out.Say("%s holds these values already", f.Conf)
		return f, nil
	}
	if err := (confFile{f.Conf, configureStage, f.M, out}).setAll(ctx, changes...); err != nil {
		return f, err
	}
	var emptied []string
	for _, c := range changes {
		if c.value == "" {
			emptied = append(emptied, c.key)
		}
	}
	if len(emptied) > 0 {
		out.Say("emptied %s", strings.Join(emptied, " "))
	}
	return f.Load(out)
}
