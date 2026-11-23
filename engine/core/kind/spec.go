package kind

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"reflect"
	"strings"

	"github.com/pelletier/go-toml/v2"

	"github.com/zadykian/ws/engine/core/resource"
)

// base holds the fields that every kind takes. Each kind embeds it, and Load alone reads it, so
// that a field shared by all kinds is a field here and a line in Load.
type base struct {
	// After names the one unit, KIND/NAME, that this one waits on.
	After string `toml:"after"`
	// When is the machine's facts that the unit needs; without them it plans nothing.
	When when `toml:"when"`
}

func (b base) waits() string { return b.After }

func (b base) condition() when { return b.When }

// unitKind is one unit's spec, of any kind.
type unitKind interface {
	waits() string
	condition() when
	// parts checks the spec and returns what makes the unit's resources as it plans. dir holds the
	// spec's own files, its templates and sources.
	parts(e Env, dir fs.FS) (func(context.Context) ([]resource.Resource, error), error)
}

// spec is one spec file: a table a kind, and a key under it a unit of that kind. A new kind is a
// field here, of a map to a type that implements unitKind, tagged with the kind's name.
type spec struct {
	File map[string]fileSpec `toml:"file"`
	Deb  map[string]debSpec  `toml:"deb"`
	// Binary is a program of a release, pinned by the SHA-256 of each architecture's file.
	Binary map[string]binarySpec `toml:"binary"`
	// Packages is apt's packages, whose names a template gives.
	Packages map[string]packagesSpec `toml:"packages"`
	// Tree is a directory unpacked from archives, each pinned by the SHA-256 of each architecture's
	// file.
	Tree map[string]treeSpec `toml:"tree"`
	// Dkms is a module's pinned source, which DKMS builds for the running kernel.
	Dkms map[string]dkmsSpec `toml:"dkms"`
	// Service is a systemd unit, with the files that one restart covers.
	Service map[string]serviceSpec `toml:"service"`
	// State is one small state with a cheap check.
	State map[string]stateSpec `toml:"state"`
	// Guard is a check that fails where a later unit would lock the machine out.
	Guard map[string]guardSpec `toml:"guard"`
	// Compose is a project of Docker Compose, recreated where its files change.
	Compose map[string]composeSpec `toml:"compose"`
	// Launchd is a family of LaunchDaemons of macOS, restarted where their program changes.
	Launchd map[string]launchdSpec `toml:"launchd"`
}

// parse fills the spec's text, named name in errors, with e.Data, pure's funcs and self, and
// decodes it. A field that no kind has is an error.
func parse(e Env, name string, text []byte) (spec, error) {
	funcs := pure()
	// self is the path of the ws that runs the units, for a command of ws in a spec.
	funcs["self"] = func() (string, error) {
		if e.Self == "" {
			return "", errors.New("self: the path of the running ws is unknown")
		}
		return e.Self, nil
	}
	filled, err := fill(name, text, funcs, e.Data)
	if err != nil {
		return spec{}, err
	}
	var s spec
	d := toml.NewDecoder(bytes.NewReader(filled))
	d.DisallowUnknownFields()
	if err := d.Decode(&s); err != nil {
		return spec{}, fmt.Errorf("%s: %w", name, describe(err))
	}
	return s, nil
}

// describe says where in the spec go-toml's error lies, as its own text names no field.
func describe(err error) error {
	var strict *toml.StrictMissingError
	if errors.As(err, &strict) {
		var keys []string
		for _, e := range strict.Errors {
			row, _ := e.Position()
			keys = append(keys, fmt.Sprintf("%s (line %d)", strings.Join(e.Key(), "."), row))
		}
		return fmt.Errorf("no kind takes the field %s", strings.Join(keys, ", "))
	}
	var bad *toml.DecodeError
	if errors.As(err, &bad) {
		row, col := bad.Position()
		if key := bad.Key(); len(key) > 0 {
			return fmt.Errorf("%s (line %d, column %d): %w", strings.Join(key, "."), row, col, err)
		}
		return fmt.Errorf("line %d, column %d: %w", row, col, err)
	}
	return err
}

// units returns the spec's units by their names, KIND/NAME, each kind read from its field's tag.
func (s spec) units() map[string]unitKind {
	all := map[string]unitKind{}
	v := reflect.ValueOf(s)
	for i := range v.NumField() {
		kind := v.Type().Field(i).Tag.Get("toml")
		for it := v.Field(i).MapRange(); it.Next(); {
			all[kind+"/"+it.Key().String()] = it.Value().Interface().(unitKind)
		}
	}
	return all
}
