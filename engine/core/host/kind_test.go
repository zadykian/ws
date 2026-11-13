package host_test

import (
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/host"
	"github.com/zadykian/ws/engine/core/host/apt"
	"github.com/zadykian/ws/engine/core/host/systemd"
	"github.com/zadykian/ws/engine/core/step"
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
	m.Dirs["/run/systemd/system"], m.Dirs["/d"] = 0o755, 0o700
	m.Files["/old"] = []byte("old\n")
	m.Files["/home/ann/.config/systemd/user/a.service"] = []byte("[Unit]\n")
	m.Files["/etc/group"] = []byte("docker:x:9:ann\n")
	m.Packages = effecttest.Packages{"git": {Version: "1", Status: "ii"}, "curl": {Status: "un"}, "a": {Version: "0", Status: "ii"}}
	m.Results["dpkg --print-architecture"] = effecttest.Result{Out: []byte("amd64\n")}
	u := &effecttest.Units{States: map[string]effect.UnitState{"on.service": {Active: true}}}
	out := &effecttest.Out{}
	ann := host.User{Name: "ann", Home: "/home/ann"}
	watch := []host.Resource{host.Absent{M: m, Path: "/old"}}
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
		{"packages", apt.Packages{M: m, Names: []string{"curl"}}, []step.ChangeKind{step.Install}},
		{"purged", apt.Purged{M: m, Names: []string{"git"}}, []step.ChangeKind{step.Removal}},
		{"reload", systemd.DaemonReload{M: m, U: u, Watch: watch}, []step.ChangeKind{step.Restart}},
		{"start", systemd.Unit{M: m, U: u, Out: out, Name: "a.service", Enabled: true, Run: systemd.Running},
			[]step.ChangeKind{step.Write, step.Restart}},
		{"restart", systemd.Unit{M: m, U: u, Out: out, Name: "on.service", Run: systemd.Running, Watch: watch},
			[]step.ChangeKind{step.Write, step.Restart}},
		{"user unit", systemd.UserUnit{M: m, Account: host.Account{User: ann, UID: 1000}, Out: out, Name: "a.service"},
			[]step.ChangeKind{step.Restart}},
		{"linger", systemd.Linger{M: m, User: ann, Out: out}, []step.ChangeKind{step.Write}},
		{"package", apt.Package{M: m, Name: "a", Source: host.PinnedFile{M: m, Version: "1", SHA256: map[string]string{"amd64": "s"},
			URL: func(version, arch string) string { return "https://x/a_" + version + arch + ".deb" }}},
			[]step.ChangeKind{step.Write, step.Write, step.Install}},
	}
	for _, tt := range tests {
		if got := kinds(t, tt.r); !slices.Equal(got, tt.want) {
			t.Errorf("%s: kinds = %v, want %v", tt.name, got, tt.want)
		}
	}
}
