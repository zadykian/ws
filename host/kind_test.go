package host_test

import (
	"slices"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/host"
	"github.com/zadykian/ws/step"
)

func kinds(t *testing.T, r host.Resource) []step.ChangeKind {
	t.Helper()
	changes, err := r.Plan(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	var got []step.ChangeKind
	for _, c := range changes {
		got = append(got, step.KindOf(c))
	}
	return got
}

func TestKindsOfTheResourcesChanges(t *testing.T) {
	m := effecttest.NewMachine()
	m.Dirs["/d"] = 0o700
	m.Files["/old"] = []byte("old\n")
	m.Files["/etc/group"] = []byte("docker:x:9:ann\n")
	tests := []struct {
		name string
		r    host.Resource
		want []step.ChangeKind
	}{
		{"file", host.File{M: m, Path: "/new", Mode: 0o644}, []step.ChangeKind{step.Write}},
		{"validated", host.Validated{File: host.File{M: m, Path: "/new", Mode: 0o644}}, []step.ChangeKind{step.Write}},
		{"dir", host.Dir{M: m, Path: "/d", Mode: 0o755}, []step.ChangeKind{step.Write}},
		{"link", host.Link{M: m, Path: "/l", Target: "/t"}, []step.ChangeKind{step.Write}},
		{"download", host.Download{M: m, Path: "/new", SHA256: "0"}, []step.ChangeKind{step.Write}},
		{"absent", host.Absent{M: m, Path: "/old"}, []step.ChangeKind{step.Removal}},
		{"left", host.Left{M: m, User: "ann", Group: "docker"}, []step.ChangeKind{step.Removal}},
	}
	for _, tt := range tests {
		if got := kinds(t, tt.r); !slices.Equal(got, tt.want) {
			t.Errorf("%s: kinds = %v, want %v", tt.name, got, tt.want)
		}
	}
}
