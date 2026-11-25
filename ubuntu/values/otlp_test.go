package values

import "testing"

// One exporter endpoint on each side. The host's own SigNoz and a self-contained guest's are on
// loopback. A guest under a host reaches the host's by the profile's socket.
func TestTheExportersEndpointFollowsTheSideAndTheMode(t *testing.T) {
	guest, err := Parse(nil)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := guest.OTLPEndpoint(), "unix:///var/lib/ws/otlp/signoz.sock"; got != want {
		t.Errorf("a guest under a host exports to %s, want %s", got, want)
	}
	self, err := Parse([]byte("WS_SELF_CONTAINED: true\n"))
	if err != nil {
		t.Fatal(err)
	}
	host, err := ParseHost(nil)
	if err != nil {
		t.Fatal(err)
	}
	for name, got := range map[string]string{"a self-contained guest": self.OTLPEndpoint(), "the host": host.OTLPEndpoint()} {
		if got != "127.0.0.1:14317" {
			t.Errorf("%s exports to %s, want 127.0.0.1:14317", name, got)
		}
	}
}

// A guest's sysfs shows the host's hwmon, so only the host reads the sensors.
func TestOnlyTheHostScrapesTheSensors(t *testing.T) {
	guest, _ := Parse(nil)
	host, _ := ParseHost(nil)
	if guest.Sensors() || !host.Sensors() {
		t.Errorf("guest %v, host %v: want the host alone", guest.Sensors(), host.Sensors())
	}
}
