package otelcol

import (
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/host"
)

func TestRemoteEndpointRefusesALinkAtTheSecretEnv(t *testing.T) {
	for _, link := range []string{"/home/ws/.secrets/env"} {
		m := effecttest.NewMachine()
		m.Links[link] = "/srv/target"
		if _, err := remoteEndpoint(m, host.User{Name: "ws", Home: "/home/ws"}); err == nil {
			t.Errorf("a link at %s: no error", link)
		}
	}
}
