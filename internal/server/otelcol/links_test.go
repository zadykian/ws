package otelcol

import (
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

func TestRemoteEndpointRefusesALinkUnderTheHome(t *testing.T) {
	for _, link := range []string{"/home/ws/.secrets", "/home/ws/.secrets/env"} {
		m := effecttest.NewMachine()
		m.Links[link] = "/srv/target"
		if _, err := remoteEndpoint(m, "/home/ws"); err == nil {
			t.Errorf("a link at %s: no error", link)
		}
	}
}
