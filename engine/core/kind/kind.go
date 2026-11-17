// Package kind holds ws's kinds: resource types written once. A role's spec names their units,
// KIND/NAME. A unit is a step.Step built on the resource packages, so that a run of units plans,
// applies and plans again as a role's steps do. No role's word sits in a kind.
package kind

import (
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/step"
)

// specFile is a role's spec, in the role's own directory one level below the root of the specs.
const specFile = "units.toml"

// unitName is a unit's name within its kind: a plain word, as a tool's package name holds it.
var unitName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// Env is what a kind builds its units on: the machine, systemd's units, the lines, the side's
// values and its operating system. The values fill the specs and their templates.
type Env struct {
	M      effect.Machine
	U      effect.Units
	Out    effect.Out
	Data   any
	System System
}

// Unit is one unit of a spec: its name, KIND/NAME, and the step that sets it up.
type Unit struct {
	Name string
	Step step.Step
}

// UnknownUnit is a name that no spec holds, which a caller takes for a usage error.
type UnknownUnit struct{ Name string }

func (u *UnknownUnit) Error() string { return fmt.Sprintf("no unit %q", u.Name) }

// Load reads each role's spec, DIR/units.toml of specs, fills it with e.Data, and returns its
// units in the order they run. The kinds check their fields here, and make their resources as each
// unit plans, where its when holds. A unit in two specs, an after that no spec holds, or a loop,
// fails the load.
func Load(e Env, specs fs.FS) ([]Unit, error) {
	files, err := fs.Glob(specs, "*/"+specFile)
	if err != nil {
		return nil, err
	}
	all := map[string]unitKind{}
	where := map[string]string{}
	for _, f := range files {
		text, err := fs.ReadFile(specs, f)
		if err != nil {
			return nil, err
		}
		s, err := parse(e, f, text)
		if err != nil {
			return nil, err
		}
		for name, k := range s.units() {
			if _, n, _ := strings.Cut(name, "/"); !unitName.MatchString(n) {
				return nil, fmt.Errorf("%s: unit %s: a name takes letters, digits, '.', '_' and '-'", f, name)
			}
			if other, twice := where[name]; twice {
				return nil, fmt.Errorf("%s: unit %s is in %s too", f, name, other)
			}
			all[name], where[name] = k, f
		}
	}
	order, err := ordered(all)
	if err != nil {
		return nil, err
	}
	units := make([]Unit, 0, len(order))
	for _, name := range order {
		k := all[name]
		dir, err := fs.Sub(specs, path.Dir(where[name]))
		if err != nil {
			return nil, err
		}
		parts, err := k.parts(e, dir)
		if err == nil {
			err = k.condition().check()
		}
		if err != nil {
			return nil, fmt.Errorf("%s: unit %s: %w", where[name], name, err)
		}
		parts = k.condition().gate(e, name, parts)
		units = append(units, Unit{name, resource.Lazy{Title: name, After: k.waits(), Parts: parts}})
	}
	return units, nil
}

// Closure returns the named units and those they wait on, in the order of units. A name that no
// unit has is an *UnknownUnit.
func Closure(units []Unit, names []string) ([]Unit, error) {
	byName := map[string]Unit{}
	for _, u := range units {
		byName[u.Name] = u
	}
	want := map[string]bool{}
	for _, n := range names {
		u, ok := byName[n]
		if !ok {
			return nil, &UnknownUnit{n}
		}
		for !want[u.Name] {
			want[u.Name] = true
			if u, ok = byName[u.Step.WaitsOn()]; !ok {
				break
			}
		}
	}
	var out []Unit
	for _, u := range units {
		if want[u.Name] {
			out = append(out, u)
		}
	}
	return out, nil
}

// Names returns the units' names, in order.
func Names(units []Unit) []string {
	names := make([]string, 0, len(units))
	for _, u := range units {
		names = append(names, u.Name)
	}
	return names
}

// Steps returns the units' steps, in order.
func Steps(units []Unit) []step.Step {
	steps := make([]step.Step, 0, len(units))
	for _, u := range units {
		steps = append(steps, u.Step)
	}
	return steps
}
