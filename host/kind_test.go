package host_test

import (
	"slices"
	"testing"

	"github.com/zadykian/ws/effect"
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
	m.Dirs["/run/systemd/system"], m.Dirs["/d"] = 0o755, 0o700
	m.Files["/old"] = []byte("old\n")
	m.Files["/home/ann/.config/systemd/user/a.service"] = []byte("[Unit]\n")
	m.Files["/etc/group"] = []byte("docker:x:9:ann\n")
	query := "dpkg-query -W -f ${Package} ${db:Status-Abbrev}\n "
	m.Results[query+"git"] = effecttest.Result{Out: []byte("git ii\n")}
	m.Results[query+"curl"] = effecttest.Result{Out: []byte("curl un\n")}
	m.Results["dpkg-query -W -f ${Version} ${db:Status-Abbrev}\n a"] = effecttest.Result{Out: []byte("0 ii\n")}
	m.Results["dpkg-deb -f /a.deb Package Version"] = effecttest.Result{Out: []byte("Package: a\nVersion: 1\n")}
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
		{"packages", host.Packages{M: m, Names: []string{"curl"}}, []step.ChangeKind{step.Install}},
		{"purged", host.Purged{M: m, Names: []string{"git"}}, []step.ChangeKind{step.Removal}},
		{"reload", host.DaemonReload{M: m, U: u, Watch: watch}, []step.ChangeKind{step.Restart}},
		{"start", host.Unit{M: m, U: u, Out: out, Name: "a.service", Enabled: true, Run: host.Running},
			[]step.ChangeKind{step.Write, step.Restart}},
		{"restart", host.Unit{M: m, U: u, Out: out, Name: "on.service", Run: host.Running, Watch: watch},
			[]step.ChangeKind{step.Write, step.Restart}},
		{"user unit", host.UserUnit{M: m, Account: host.Account{User: ann, UID: 1000}, Out: out, Name: "a.service"},
			[]step.ChangeKind{step.Restart}},
		{"linger", host.Linger{M: m, User: ann, Out: out}, []step.ChangeKind{step.Write}},
		{"deb", host.Deb{M: m, Path: "/a.deb"}, []step.ChangeKind{step.Install}},
	}
	for _, tt := range tests {
		if got := kinds(t, tt.r); !slices.Equal(got, tt.want) {
			t.Errorf("%s: kinds = %v, want %v", tt.name, got, tt.want)
		}
	}
}
