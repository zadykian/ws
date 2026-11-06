package wizard

import (
	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/internal/conf"
)

// configureStage names the Configure screen in what a save says.
const configureStage = "Configure"

// Field is a key of mac.conf as the Configure screen shows it.
type Field struct {
	Key, Text string
	// Value is what the file holds, and Default what an empty field gives.
	Value, Default string
	// Options are the values of a key that takes one of a few, and nil for a line.
	Options []string
	// Valid says why mac.conf can't take a value. It is pure, as it runs on each edit.
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
