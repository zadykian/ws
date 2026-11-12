package effecttest

import (
	"fmt"
	"maps"
	"os/exec"
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

// dpkgQuery answers "dpkg-query -W -f FORMAT NAME..." from the table, and reports whether it did.
// Where a name is unknown it prints the rest and ends with exit 1, as dpkg-query does. The caller
// holds the lock.
func (m *Machine) dpkgQuery(name string, args []string) (Result, bool) {
	if name != "dpkg-query" || m.Packages == nil || len(args) < 4 || args[0] != "-W" || args[1] != "-f" {
		return Result{}, false
	}
	var out strings.Builder
	var err error
	for _, n := range slices.Sorted(maps.Keys(m.Packages)) {
		if !slices.Contains(args[3:], n) {
			continue
		}
		p := m.Packages[n]
		out.WriteString(strings.NewReplacer("${Package}", n, "${Version}", p.Version,
			"${db:Status-Abbrev}", fmt.Sprintf("%-3s", p.Status)).Replace(args[2]))
	}
	for _, n := range args[3:] {
		if _, ok := m.Packages[n]; !ok {
			err = exitOne()
		}
	}
	return Result{Out: []byte(out.String()), Err: err}, true
}
