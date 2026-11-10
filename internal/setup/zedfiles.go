package setup

import (
	"bytes"
	"strings"

	"github.com/zadykian/ws/step"
)

const (
	// zedHeader is the first line of global_settings.json, by which a run knows a file for its own.
	zedHeader = "// Written by ws mac apply: its next run replaces changes made here."
	// zedKeymapMark starts the first line of keymap.json, which carries its own header. It does not
	// name the command, so the file stays ws's if the command changes.
	zedKeymapMark = "// Written by github.com/zadykian/ws"
	// zedSeed is settings.json where the file is missing. Zed writes that file from its UI and
	// merges it over global_settings.json. The connection to ws can sit in no other file, as an
	// opened project is recorded by its index in this list.
	zedSeed = `// Zed's own settings, which its UI writes. ws wrote this once, where it was missing; ws's
// settings are in global_settings.json, and a key set here wins over them.
{ "ssh_connections": [{ "host": "ws" }] }
`
)

// files plans Zed's directory and its three files. global_settings.json is Settings, replaced on
// each run, and left with a warning where its first line is not ws's. keymap.json is Keymap,
// replaced and left on the same terms with its own mark. settings.json is written where nothing is there, and never read.
func (z *Zed) files() []step.Change {
	conf := z.Home + "/.config"
	dir := conf + "/zed"
	global := dir + "/global_settings.json"
	var cs []step.Change
	dirInfo, dirErr := z.M.Stat(dir)
	if dirErr != nil {
		if _, err := z.M.Stat(conf); err != nil {
			cs = append(cs, MakeDir{conf, 0o755})
		}
		cs = append(cs, MakeDir{dir, 0o755})
	}
	cs = append(cs, z.owned(global, z.Settings, zedOwn(zedHeader, true))...)
	cs = append(cs, z.owned(dir+"/keymap.json", z.Keymap, zedOwn(zedKeymapMark, false))...)
	if _, err := z.M.Stat(dir + "/settings.json"); err != nil {
		cs = append(cs, Write{dir + "/settings.json", nil, []byte(zedSeed), false, 0o644})
	}
	if dirErr == nil && dirInfo.Link != "" && len(cs) > 0 {
		z.Out.Warn("%s is a link to %s: Zed's files go there", dir, dirInfo.Link)
	}
	return cs
}

// zedOwn returns the test of a file's first line: equal to mark, or starting with it where exact is
// false.
func zedOwn(mark string, exact bool) func([]byte) bool {
	return func(have []byte) bool {
		first, _, _ := strings.Cut(string(have), "\n")
		return first == mark || (!exact && strings.HasPrefix(first, mark))
	}
}

// owned plans a file that ws replaces while own passes it. A file with another first line is left,
// with a warning.
func (z *Zed) owned(file string, want []byte, own func([]byte) bool) []step.Change {
	info, err := z.M.Stat(file)
	if err != nil {
		return []step.Change{Write{file, nil, want, false, 0o644}}
	}
	old, _ := z.M.ReadFile(file)
	if !own(old) {
		z.Out.Warn("%s is not ws's: move it away, and the next run writes ws's", file)
		return nil
	}
	switch {
	case !bytes.Equal(old, want):
		return []step.Change{Write{file, old, want, true, 0o644}}
	case info.Mode.Perm() != 0o644:
		return []step.Change{SetMode{file, 0o644}}
	}
	return nil
}
