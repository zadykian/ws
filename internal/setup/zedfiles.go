package setup

import (
	"bytes"
	"strings"

	"github.com/zadykian/ws/step"
)

// zedHeader is the first line of global_settings.json, by which a run knows a file for its own.
const zedHeader = "// Written by ws mac apply: its next run replaces changes made here."

// files plans Zed's directory and ws's settings. global_settings.json is the header and
// mac/zed.json, replaced on each run, and left with a warning where its first line is not ws's.
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
	cs = append(cs, z.global(global)...)
	if dirErr == nil && dirInfo.Link != "" && len(cs) > 0 {
		z.Out.Warn("%s is a link to %s: Zed's files go there", dir, dirInfo.Link)
	}
	return cs
}

// global plans global_settings.json.
func (z *Zed) global(file string) []step.Change {
	want := append([]byte(zedHeader+"\n"), z.Settings...)
	info, err := z.M.Stat(file)
	if err != nil {
		return []step.Change{Write{file, nil, want, false, 0o644}}
	}
	old, _ := z.M.ReadFile(file)
	if first, _, _ := strings.Cut(string(old), "\n"); first != zedHeader {
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
