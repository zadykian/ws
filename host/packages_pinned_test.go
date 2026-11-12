package host

import (
	"context"
	"errors"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

func TestPackagesPlanPinnedVersion(t *testing.T) {
	tests := []struct {
		name string
		pkgs effecttest.Packages
		err  error
		want string
		fail bool
	}{
		{"the version matches", effecttest.Packages{"nodejs": {Version: "24.1.0-1nodesource1", Status: "ii"}}, nil, "", false},
		{"another major version", effecttest.Packages{"nodejs": {Version: "22.9.0-1nodesource1", Status: "ii"}}, nil,
			"install nodejs=24.*", false},
		{"removed with config left", effecttest.Packages{"nodejs": {Version: "24.1.0", Status: "rc"}}, nil, "install nodejs=24.*", false},
		{"unknown", effecttest.Packages{}, nil, "install nodejs=24.*", false},
		{"no command", nil, errors.New("not found"), "", true},
	}
	for _, tt := range tests {
		m := effecttest.NewMachine()
		m.Packages = tt.pkgs
		if tt.err != nil {
			m.Results["dpkg-query -W -f ${Version} ${db:Status-Abbrev}\n nodejs"] = effecttest.Result{Err: tt.err}
		}
		pl, err := Packages{M: m, Names: []string{"nodejs=24.*"}}.Plan(context.Background())
		got := ""
		if len(pl) > 0 {
			got = pl[0].String()
		}
		if (err != nil) != tt.fail || got != tt.want {
			t.Errorf("%s: got %q, %v; want %q, fail %v", tt.name, got, err, tt.want, tt.fail)
		}
	}
}
