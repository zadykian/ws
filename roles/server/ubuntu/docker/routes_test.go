package docker_test

import (
	"testing"

	"github.com/zadykian/ws/roles/server/ubuntu/docker"
)

func TestParseRoutesReadsIPsJSON(t *testing.T) {
	got, err := docker.ParseRoutes([]byte(`[{"dst":"172.18.0.0/16","dev":"br-1","scope":"link"}]`))
	if err != nil {
		t.Fatal(err)
	}
	if want := (docker.Route{Dst: "172.18.0.0/16", Dev: "br-1", Scope: "link"}); len(got) != 1 || got[0] != want {
		t.Errorf("ParseRoutes = %+v, want %+v", got, want)
	}
	if _, err := docker.ParseRoutes([]byte("no json")); err == nil {
		t.Error("text that is no JSON passed")
	}
}
