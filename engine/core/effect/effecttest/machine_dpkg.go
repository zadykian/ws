package effecttest

import (
	"fmt"
	"maps"
	"os/exec"
	"path"
	"slices"
	"strings"
	"sync"
)

// Package is a package as dpkg's database holds it. Status is ${db:Status-Abbrev} without its
// padding, such as "ii", "rc", "un", "iU" or "hi"; the fake pads it to three columns.
type Package struct{ Version, Status string }

// Packages is dpkg's database by package name.
type Packages map[string]Package

// exitOne is dpkg-query's error where a pattern matches no package. An ExitError needs a process.
var exitOne = sync.OnceValue(func() error {
	return exec.Command("sh", "-c", "exit 1").Run()
})

var knownFields = strings.NewReplacer("${Package}", "", "${Version}", "", "${db:Status-Abbrev}", "")

// dpkgQuery answers "dpkg-query -W -f FORMAT PATTERN..." from the table, and reports whether it
// did. A pattern is a name or a glob. Each package that one matches prints once, in name order.
// Where a pattern matches none, the rest still prints and the error is exit 1, as dpkg-query
// does. A field of the format other than apt.PackageStates' three fails the command. The
// caller holds the lock.
func (m *Machine) dpkgQuery(name string, args []string) (Result, bool) {
	if name != "dpkg-query" || m.Packages == nil || len(args) < 4 || args[0] != "-W" || args[1] != "-f" {
		return Result{}, false
	}
	if rest := knownFields.Replace(args[2]); strings.Contains(rest, "${") {
		return Result{Err: fmt.Errorf("effecttest: dpkg-query field in %q", args[2])}, true
	}
	var out strings.Builder
	var err error
	names := slices.Sorted(maps.Keys(m.Packages))
	for _, pattern := range args[3:] {
		if !slices.ContainsFunc(names, func(n string) bool { return matches(pattern, n) }) {
			err = exitOne()
		}
	}
	for _, n := range names {
		if !slices.ContainsFunc(args[3:], func(pattern string) bool { return matches(pattern, n) }) {
			continue
		}
		p := m.Packages[n]
		out.WriteString(strings.NewReplacer("${Package}", n, "${Version}", p.Version,
			"${db:Status-Abbrev}", fmt.Sprintf("%-3s", p.Status)).Replace(args[2]))
	}
	return Result{Out: []byte(out.String()), Err: err}, true
}

func matches(pattern, name string) bool {
	ok, _ := path.Match(pattern, name)
	return ok
}
