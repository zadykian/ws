package cli

import (
	"reflect"
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

// A host's ufw lets Docker's networks reach the collector's socket on the bridge address, and a
// guest, which holds no filter, has no such rule.
func TestHostLetsDockersContainersSendToTheCollector(t *testing.T) {
	got, err := planned(t, hostSpecs(t, effecttest.NewMachine()), "state/otelcol-docker-ufw")
	want := []string{
		"ufw allow in on docker0 from any to 172.17.0.1 port 4317 proto tcp",
		"ufw allow in on br-+ from any to 172.17.0.1 port 4317 proto tcp",
	}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Errorf("plan = %q, %v, want %q", got, err, want)
	}
	if slices.Contains(specLines(t, "guest", defaults(t)), "state/otelcol-docker-ufw") {
		t.Error("the guest holds the ufw rule")
	}
}
