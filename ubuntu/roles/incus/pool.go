// Package incus is the incus role's own steps: the objects that its units make through the
// package object, once Incus's packages are in.
package incus

import (
	"errors"
	"fmt"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/ubuntu/roles/incus/object"
	"github.com/zadykian/ws/ubuntu/values"
)

// PoolName is the one storage pool that the guests take their disks from.
const PoolName = "ws"

// poolKind is incus storage, whose keys that an edit takes are description and config.
var poolKind = object.Kind{
	Noun:   "storage pool",
	List:   []string{"storage", "list"},
	Create: []string{"storage", "create", object.NameArg},
	Edit:   []string{"storage", "edit", object.NameArg},
	Keys:   []string{"description", "config"},
}

// Pool is the ZFS pool on the partition of WS_INCUS_POOL_SOURCE. Incus never moves a pool, and
// ZFS's create wipes the partition, so a pool on another partition is refused, not remade. A
// host that holds its pool needs no value.
func Pool(m effect.Machine, v values.Incus) object.Object {
	return object.Object{
		M: m, Kind: poolKind, Name: PoolName, Args: []string{"zfs"},
		Want:  map[string]any{"description": "ws's pool"},
		First: map[string]any{"config": map[string]any{"source": v.PoolSource}},
		OnCreate: func() error {
			if v.PoolSource == "" {
				return errors.New("WS_INCUS_POOL_SOURCE names no partition: set it in /etc/ws/values.yml to the one the pool takes, as /dev/disk/by-partuuid/ID")
			}
			return nil
		},
		Check: func(have map[string]any) error { return samePartition(have, v.PoolSource) },
	}
}

// samePartition compares the partition that a pool was made on with want. ZFS's create sets the
// pool's source to its name and keeps the partition in volatile.initial_source. So that key is
// the one to read, and source the fallback for a driver that keeps none. An empty want compares
// nothing.
func samePartition(have map[string]any, want string) error {
	if want == "" {
		return nil
	}
	config, _ := have["config"].(map[string]any)
	got, ok := config["volatile.initial_source"].(string)
	if !ok {
		got, _ = config["source"].(string)
	}
	if got != want {
		return fmt.Errorf("it was made on %q, and WS_INCUS_POOL_SOURCE is %q: Incus never moves a pool, so destroy it by hand, or set the key back", got, want)
	}
	return nil
}
