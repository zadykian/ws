package incus

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/ubuntu/values"
)

const (
	poolList = "incus storage list --format json"
	// zfsPool is what incus lists for a ZFS pool made on a partition: source holds the pool's
	// name after the create, and volatile.initial_source the partition (docs: storage_zfs).
	zfsPool = `[{"name":"default","driver":"dir","config":{"source":"/var/lib/incus/storage-pools/default"}},
{"name":"ws","description":"ws's pool","driver":"zfs","status":"Created","used_by":["/1.0/profiles/default"],
"config":{"source":"ws","volatile.initial_source":"/dev/disk/by-partuuid/0a1b","volume.zfs.remove_snapshots":"true","volume.zfs.reserve_space":"true","volume.zfs.use_refquota":"true","zfs.pool_name":"ws"},"locations":["none"]}]`
)

func poolOn(m *effecttest.Machine, source string) []string {
	r, err := Pool(m, values.Incus{PoolSource: source}).Plan(context.Background())
	if err != nil {
		return []string{"error: " + err.Error()}
	}
	var heads []string
	for _, c := range r {
		head, _, _ := strings.Cut(c.String(), "\n")
		heads = append(heads, head)
	}
	return heads
}

func TestAMissingPoolIsCreatedOnTheHostsPartition(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[poolList] = effecttest.Result{Out: []byte("[]")}
	m.Results["incus storage create ws zfs"] = effecttest.Result{}
	changes, err := Pool(m, values.Incus{PoolSource: "/dev/disk/by-partuuid/0a1b"}).Plan(t.Context())
	if err != nil || len(changes) != 1 {
		t.Fatalf("changes = %v, err = %v", changes, err)
	}
	if err := changes[0].Apply(t.Context()); err != nil {
		t.Fatal(err)
	}
	want := "config:\n    source: /dev/disk/by-partuuid/0a1b\n    volume.zfs.remove_snapshots: \"true\"\n    volume.zfs.reserve_space: \"true\"\n    volume.zfs.use_refquota: \"true\"\ndescription: ws's pool\n"
	if m.Inputs[0] != want {
		t.Errorf("stdin = %q, want %q", m.Inputs[0], want)
	}
}

func TestAMissingPoolWithNoPartitionNamesTheKey(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[poolList] = effecttest.Result{Out: []byte("[]")}
	_, err := Pool(m, values.Incus{}).Plan(t.Context())
	if err == nil || !strings.Contains(err.Error(), "WS_INCUS_POOL_SOURCE") {
		t.Errorf("err = %v, want the key named", err)
	}
}

// ZFS's create sets source to the pool's name, so the plan compares the partition with
// volatile.initial_source, which a pool in sync holds.
func TestAPoolOnItsPartitionIsLeftAlone(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[poolList] = effecttest.Result{Out: []byte(zfsPool)}
	if got := poolOn(m, "/dev/disk/by-partuuid/0a1b"); len(got) != 0 {
		t.Errorf("changes = %v, want none", got)
	}
}

func TestAPoolWithNoPartitionNamedIsLeftAlone(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[poolList] = effecttest.Result{Out: []byte(zfsPool)}
	if got := poolOn(m, ""); len(got) != 0 {
		t.Errorf("changes = %v, want none: a host that made its pool needs no value", got)
	}
}

func TestAPoolOnAnotherPartitionIsRefused(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[poolList] = effecttest.Result{Out: []byte(zfsPool)}
	got := poolOn(m, "/dev/disk/by-partuuid/ffff")
	if len(got) != 1 || !strings.Contains(got[0], "/dev/disk/by-partuuid/0a1b") || !strings.Contains(got[0], "/dev/disk/by-partuuid/ffff") {
		t.Errorf("result = %v, want a refusal that names both partitions", got)
	}
}

func TestAPoolWithNoInitialSourceHoldsItsPartitionInSource(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[poolList] = effecttest.Result{Out: []byte(`[{"name":"ws","description":"ws's pool","config":{"source":"/dev/vdb1","volume.zfs.remove_snapshots":"true","volume.zfs.reserve_space":"true","volume.zfs.use_refquota":"true"}}]`)}
	if got := poolOn(m, "/dev/vdb1"); len(got) != 0 {
		t.Errorf("changes = %v, want none: without volatile.initial_source, source is the partition", got)
	}
}

func TestADifferingDescriptionIsEdited(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[poolList] = effecttest.Result{Out: []byte(strings.Replace(zfsPool, "ws's pool", "mine", 1))}
	got := poolOn(m, "/dev/disk/by-partuuid/0a1b")
	if len(got) != 1 || got[0] != "edit the storage pool ws: description" {
		t.Errorf("changes = %v", got)
	}
}

// A quota that counts the snapshots fills with them, and ZFS restores the latest snapshot alone.
func TestAPoolWithoutRefquotaOrSnapshotRemovalIsEdited(t *testing.T) {
	m := effecttest.NewMachine()
	old := strings.NewReplacer(`"volume.zfs.remove_snapshots":"true",`, "", `"volume.zfs.use_refquota":"true",`, "").Replace(zfsPool)
	m.Results[poolList] = effecttest.Result{Out: []byte(old)}
	got := poolOn(m, "/dev/disk/by-partuuid/0a1b")
	if len(got) != 1 || got[0] != "edit the storage pool ws: config.volume.zfs.remove_snapshots, config.volume.zfs.use_refquota" {
		t.Errorf("changes = %v", got)
	}
}

// A quota that no reservation holds leaves a guest's space to the others' snapshots.
func TestAPoolThatReservesNoQuotaIsEdited(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[poolList] = effecttest.Result{Out: []byte(strings.Replace(zfsPool, `"volume.zfs.reserve_space":"true",`, "", 1))}
	got := poolOn(m, "/dev/disk/by-partuuid/0a1b")
	if len(got) != 1 || got[0] != "edit the storage pool ws: config.volume.zfs.reserve_space" {
		t.Errorf("changes = %v", got)
	}
}
