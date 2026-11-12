package effecttest

import (
	"io/fs"
	"slices"
	"strings"

	"github.com/zadykian/ws/effect"
)

// tree handles two commands that the table holds no result for, and reports whether it did. Root's
// "rm -rf -- PATH" drops PATH and everything under it. "mv -T -- FROM TO" moves FROM and everything
// under it to TO. The caller holds the lock.
func (m *Machine) tree(name string, args []string) bool {
	paths := slices.DeleteFunc(slices.Clone(args), func(a string) bool { return strings.HasPrefix(a, "-") })
	switch {
	case name == "rm" && slices.Contains(args, "-rf"):
		for _, p := range paths {
			m.each(p, func(string) string { return "" })
		}
		return true
	case name == "mv" && slices.Contains(args, "-T") && len(paths) == 2:
		from, to := paths[0], paths[1]
		m.each(from, func(key string) string { return to + strings.TrimPrefix(key, from) })
		return true
	}
	return false
}

// each applies move to every key at or under p in the maps: it drops the key where move returns
// "", else it renames it.
func (m *Machine) each(p string, move func(key string) string) {
	under := func(key string) bool { return key == p || strings.HasPrefix(key, p+"/") }
	rename(m.Files, under, move)
	rename(m.Root, under, move)
	rename(m.Dirs, under, move)
	rename(m.Links, under, move)
	rename(m.Modes, under, move)
	rename(m.Owners, under, move)
}

func rename[V string | []byte | effect.Attrs | fs.FileMode | [2]int](
	mp map[string]V, under func(string) bool, move func(string) string,
) {
	moved := map[string]V{}
	for key, v := range mp {
		if !under(key) {
			continue
		}
		delete(mp, key)
		if to := move(key); to != "" {
			moved[to] = v
		}
	}
	for key, v := range moved {
		mp[key] = v
	}
}
