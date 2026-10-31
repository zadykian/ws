package setup

import (
	pathpkg "path"
	"strings"
)

// ghosttyIncludes reports whether Ghostty's config, at conf with the text given, has a config-file
// line that names file. The line counts with or without the ? and the quotes Ghostty takes, and
// its path may be relative to conf's directory, under ~/ or full (src/cli/args.zig's
// LineIterator). A path counts by its directory with links resolved, as Ghostty resolves a
// relative one (src/config/path.zig). resolve resolves the links of a directory.
func ghosttyIncludes(conf, text, file, home string, resolve func(string) (string, error)) bool {
	want, err := resolve(pathpkg.Dir(file))
	if err != nil {
		return false
	}
	for _, line := range strings.Split(text, "\n") {
		p, ok := ghosttyConfigFile(line, pathpkg.Dir(conf), home)
		if !ok || pathpkg.Base(p) != pathpkg.Base(file) {
			continue
		}
		if got, err := resolve(pathpkg.Dir(p)); err == nil && got == want {
			return true
		}
	}
	return false
}

// ghosttyConfigFile returns the full path that a config-file line names.
func ghosttyConfigFile(line, dir, home string) (string, bool) {
	line = strings.TrimRight(strings.TrimLeft(line, " \t"), " \t\r")
	key, val, found := strings.Cut(line, "=")
	if strings.HasPrefix(line, "#") || !found || strings.TrimRight(key, " \t") != "config-file" {
		return "", false
	}
	val = unquote(strings.TrimLeft(val, " \t"))
	val = unquote(strings.TrimPrefix(val, "?"))
	switch {
	case strings.HasPrefix(val, "~/"):
		val = home + val[1:]
	case !strings.HasPrefix(val, "/"):
		val = dir + "/" + val
	}
	return val, true
}

// unquote drops one pair of double quotes around s.
func unquote(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}
