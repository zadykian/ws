// Package object reads, compares and writes the objects that Incus keeps: an ACL, a pool, a network
// and a profile, and later the guests. ws declares the keys it cares for, and Incus fills the rest,
// such as volatile.* keys. So ws's YAML never equals a whole object, and an edit that replaced one
// could drop those. A missing object is created from the declared keys. A differing one is edited
// by patching them into what Incus holds. Both send YAML on the standard input of incus. Incus has
// no dry run, so the comparison is the plan.
package object

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
)

// NameArg stands for an object's name in the words of a Kind, where incus takes it.
const NameArg = "{name}"

// Kind says how the incus command reads and writes one sort of object. Each slice holds words of
// the command, after incus: `network acl list`, say.
type Kind struct {
	// Noun names the sort in a message: "network ACL".
	Noun string
	// List prints every object of the sort as JSON, as incus's --format json does, each with its
	// name. The words leave out the format.
	List []string
	// Create and Edit take the object's YAML on standard input. Each holds NameArg once, where the
	// command takes the object's name.
	Create, Edit []string
	// Keys are the keys of an object, at its top, that Edit takes: the object's own, less what
	// Incus fills, such as used_by.
	Keys []string
}

// Object is one object of Incus that ws declares, and a resource.Resource that makes it so. An
// object declares no secret, as a plan shows its YAML.
type Object struct {
	M    effect.Machine
	Kind Kind
	Name string
	// Want is the keys that ws declares, as YAML decodes them. Each run compares them with the
	// object's, sends them whole on a create, and patches them into the object on an edit. A list
	// in Want replaces the object's whole, and an empty one matches a key Incus leaves out.
	Want map[string]any
	// First is the keys that a create alone takes, such as a pool's source, which Incus never
	// edits. They are never compared.
	First map[string]any
	// Args follow the name on a create, as a pool's driver does.
	Args []string
	// Check refuses an object that already exists, by what it reads: a key that an edit could not
	// change. Nil refuses none.
	Check func(have map[string]any) error
}

// Plan creates the object where Incus lacks it, and edits it where a declared key differs.
func (o Object) Plan(ctx context.Context) ([]resource.Change, error) {
	if err := o.valid(); err != nil {
		return nil, err
	}
	have, found, err := o.read(ctx)
	if err != nil {
		return nil, err
	}
	if !found {
		body, err := yaml.Marshal(merge(o.First, o.Want))
		if err != nil {
			return nil, err
		}
		return []resource.Change{create{o: o, body: body}}, nil
	}
	if o.Check != nil {
		if err := o.Check(have); err != nil {
			return nil, fmt.Errorf("the %s %s: %w", o.Kind.Noun, o.Name, err)
		}
	}
	keys := differs(o.Want, have, "")
	if len(keys) == 0 {
		return nil, nil
	}
	now := pick(have, o.Kind.Keys)
	before, err := yaml.Marshal(now)
	if err != nil {
		return nil, err
	}
	after, err := yaml.Marshal(merge(now, o.Want))
	if err != nil {
		return nil, err
	}
	return []resource.Change{edit{o: o, keys: keys, before: before, after: after}}, nil
}

// valid refuses a Kind whose commands do not name the object.
func (o Object) valid() error {
	for _, words := range [][]string{o.Kind.Create, o.Kind.Edit} {
		if slices.Index(words, NameArg) < 0 {
			return fmt.Errorf("the %s kind's words %q lack %s, where incus takes the name", o.Kind.Noun, words, NameArg)
		}
	}
	return nil
}

// read finds the object in incus's list. A name that the list lacks is no object, so that a daemon
// that fails to answer is never taken for a missing one.
func (o Object) read(ctx context.Context) (map[string]any, bool, error) {
	args := append(slices.Clone(o.Kind.List), "--format", "json")
	out, err := o.M.Run(ctx, "incus", args...)
	if err != nil {
		return nil, false, err
	}
	dec := json.NewDecoder(bytes.NewReader(out))
	dec.UseNumber()
	var all []map[string]any
	if err := dec.Decode(&all); err != nil {
		return nil, false, fmt.Errorf("incus %s: its list is no JSON array of objects: %w", strings.Join(o.Kind.List, " "), err)
	}
	for _, one := range all {
		if one["name"] == o.Name {
			return plain(one).(map[string]any), true, nil
		}
	}
	return nil, false, nil
}

// argv returns the arguments of incus for words, with the object's name where NameArg stands.
func (o Object) argv(words []string) []string {
	return slices.Collect(func(yield func(string) bool) {
		for _, w := range words {
			if w == NameArg {
				w = o.Name
			}
			if !yield(w) {
				return
			}
		}
	})
}

// create makes the object from its declared keys, and First's.
type create struct {
	o    Object
	body []byte
}

func (c create) String() string {
	return fmt.Sprintf("create the %s %s\n%s", c.o.Kind.Noun, c.o.Name, c.body)
}

func (c create) Apply(ctx context.Context) error {
	_, err := c.o.M.RunInput(ctx, c.body, "incus", append(c.o.argv(c.o.Kind.Create), c.o.Args...)...)
	return err
}

// edit patches the declared keys into the object that Incus holds.
type edit struct {
	o             Object
	keys          []string
	before, after []byte
}

// maxNamed bounds the keys that a change's first line names.
const maxNamed = 4

// String names the keys that differ on its first line, and shows the object's YAML before and after.
func (c edit) String() string {
	named := c.keys
	if len(named) > maxNamed {
		named = append(slices.Clone(named[:maxNamed]), "…")
	}
	head := fmt.Sprintf("edit the %s %s: %s", c.o.Kind.Noun, c.o.Name, strings.Join(named, ", "))
	return head + "\n" + effect.Diff(c.o.Name, c.before, c.after)
}

func (c edit) Apply(ctx context.Context) error {
	_, err := c.o.M.RunInput(ctx, c.after, "incus", c.o.argv(c.o.Kind.Edit)...)
	return err
}
