package docker_test

import (
	"slices"
	"testing"

	"github.com/zadykian/ws/internal/server/docker"
)

const (
	pool   = "172.16.0.0/12"
	bridge = "172.17.0.1/16"
)

func conflicts(t *testing.T, routesJSON string) []string {
	routes, err := docker.ParseRoutes([]byte(routesJSON))
	if err != nil {
		t.Fatal(err)
	}
	got, err := docker.Conflicts(routes, pool, bridge)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func TestConflictsCountsTheRoutesDockerDoesNotAvoid(t *testing.T) {
	got := conflicts(t, `[
	  {"dst":"default","dev":"eth0"},
	  {"dst":"172.16.5.0/24","dev":"wg0","table":"51820"},
	  {"dst":"172.20.0.0/16","dev":"br-1","scope":"link"},
	  {"dst":"172.17.4.0/24","dev":"tun0","table":"7"},
	  {"dst":"172.18.0.9","dev":"tun0","table":"7"}]`)
	want := []string{"172.16.5.0/24 dev wg0 table 51820", "172.17.4.0/24 dev tun0 table 7",
		"172.18.0.9/32 dev tun0 table 7"}
	if !slices.Equal(got, want) {
		t.Errorf("Conflicts = %q, want %q", got, want)
	}
}
