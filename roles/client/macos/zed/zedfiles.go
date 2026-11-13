package zed

import (
	"context"
	"strings"

	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/file"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/conf"
)

const (
	// zedHeader is the first line of global_settings.json, by which a run knows a file for its own.
	zedHeader = "// Written by ws client macos apply: its next run replaces changes made here."
	// zedKeymapMark starts the first line of keymap.json, which carries its own header. It does not
	// name the command, so the file stays ws's if the command changes.
	zedKeymapMark = "// Written by github.com/zadykian/ws"
	// zedSeedHead is the start of settings.json where the file is missing. Zed writes that file from
	// its UI and merges it over global_settings.json. The connections to the servers can sit in no
	// other file, as an opened project is recorded by its index in this list.
	zedSeedHead = `// Zed's own settings, which its UI writes. ws wrote this once, where it was missing; ws's
// settings are in global_settings.json, and a key set here wins over them.
`
)

// files plans Zed's directory and its three files. global_settings.json is Settings, replaced on
// each run, and left with a warning where its first line is not ws's. keymap.json is Keymap,
// replaced and left on the same terms with its own mark. settings.json is written where nothing is there, and never read.
func (z *Zed) files(ctx context.Context) ([]step.Change, error) {
	dir := z.Home + "/.config/zed"
	cs, link, err := z.dirs().Plan()
	if err != nil {
		return nil, err
	}
	for _, f := range z.whole() {
		if cs, err = resource.Append(ctx, cs, f); err != nil {
			return nil, err
		}
	}
	if _, err := z.M.Stat(dir + "/settings.json"); err != nil {
		names, err := conf.Known(z.M, z.MacConf, z.Server)
		if err != nil {
			return nil, err
		}
		cs = append(cs, file.UserWrite{M: z.M, Out: z.Out, Path: dir + "/settings.json", Before: nil, After: seed(names), Exists: false, Mode: 0o644})
	}
	link.Warn(z.Out, len(cs), "Zed's files go there")
	return cs, nil
}

// seed is settings.json where the file is missing: a connection to each server, in the order given.
func seed(names []string) []byte {
	conns := make([]string, len(names))
	for i, n := range names {
		conns[i] = `{ "host": "` + n + `" }`
	}
	return []byte(zedSeedHead + `{ "ssh_connections": [` + strings.Join(conns, ", ") + "] }\n")
}

// dirs returns ~/.config and Zed's directory in it.
func (z *Zed) dirs() file.UserDirs {
	return file.UserDirs{{M: z.M, Out: z.Out, Path: z.Home + "/.config", Mode: 0o755}, {M: z.M, Out: z.Out, Path: z.Home + "/.config/zed", Mode: 0o755}}
}

// whole returns global_settings.json and keymap.json, the files that ws replaces whole.
func (z *Zed) whole() []file.UserFile {
	dir := z.Home + "/.config/zed"
	return []file.UserFile{
		{M: z.M, Out: z.Out, Path: dir + "/global_settings.json", Text: z.Settings, Mode: 0o644, Own: zedOwn(zedHeader, true), Warn: true},
		{M: z.M, Out: z.Out, Path: dir + "/keymap.json", Text: z.Keymap, Mode: 0o644, Own: zedOwn(zedKeymapMark, false), Warn: true},
	}
}

// zedOwn returns the test of a file's first line: equal to mark, or starting with it where exact is
// false.
func zedOwn(mark string, exact bool) func([]byte) bool {
	return func(have []byte) bool {
		first, _, _ := strings.Cut(string(have), "\n")
		return first == mark || (!exact && strings.HasPrefix(first, mark))
	}
}
