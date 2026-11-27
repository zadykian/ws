package incus

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/ubuntu/values"
)

// Instance is what incus list prints of an instance that ws reads.
type Instance struct {
	Name     string            `json:"name"`
	Status   string            `json:"status"`
	Profiles []string          `json:"profiles"`
	Config   map[string]string `json:"config"`
}

// Instances lists every instance that Incus holds.
func Instances(ctx context.Context, m effect.Machine) ([]Instance, error) {
	out, err := m.Run(ctx, "incus", "list", "--format", "json")
	if err != nil {
		return nil, err
	}
	var all []Instance
	if err := json.NewDecoder(bytes.NewReader(out)).Decode(&all); err != nil {
		return nil, fmt.Errorf("incus list: its output is no JSON array of instances: %w", err)
	}
	return all, nil
}

// Strays are the guests of ws that Incus holds and the list lacks: instances that take the
// profile ws and carry another name than the list's. ws deletes none, and reports them.
func Strays(all []Instance, guests []values.Guest) []string {
	var names []string
	for _, i := range all {
		if slices.Contains(i.Profiles, ProfileName) && !slices.ContainsFunc(guests, func(g values.Guest) bool { return g.Name == i.Name }) {
			names = append(names, i.Name)
		}
	}
	return names
}

// Running starts a guest that is stopped, as a reboot or a crash leaves it. A guest that Incus lacks
// is the instance's to create, and one that runs is left alone.
type Running struct {
	M    effect.Machine
	Name string
}

// Plan plans the start where the guest is stopped.
func (r Running) Plan(ctx context.Context) ([]resource.Change, error) {
	all, err := Instances(ctx, r.M)
	if err != nil {
		return nil, err
	}
	for _, i := range all {
		if i.Name == r.Name && i.Status == "Stopped" {
			return []resource.Change{start{r}}, nil
		}
	}
	return nil, nil
}

type start struct{ r Running }

func (c start) String() string { return "start the guest " + c.r.Name }

func (c start) Apply(ctx context.Context) error {
	_, err := c.r.M.Run(ctx, "incus", "start", c.r.Name)
	return err
}

// Remove deletes the instance of a guest, with its disk and snapshots. The caller has checked that
// the instance is a guest of ws's, which takes the profile ws.
func Remove(ctx context.Context, m effect.Machine, name string) error {
	_, err := m.Run(ctx, "incus", "delete", "--force", name)
	return err
}

// IsGuest reports whether the instances hold a guest of ws's called name: one with the profile ws.
func IsGuest(all []Instance, name string) bool {
	return slices.ContainsFunc(all, func(i Instance) bool { return i.Name == name && slices.Contains(i.Profiles, ProfileName) })
}
