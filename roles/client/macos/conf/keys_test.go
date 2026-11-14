package conf

import "testing"

// find returns the key named name.
func find(t *testing.T, name string) key {
	for _, k := range keys {
		if k.name == name {
			return k
		}
	}
	t.Fatalf("no key %s", name)
	return key{}
}

func TestTunnelSlot(t *testing.T) {
	for slot, want := range map[string]string{"": "10.99.0.1", "0": "10.99.0.1", "1": "10.99.1.1", "15": "10.99.15.1"} {
		if got := (Conf{Slot: slot}).TunnelSlot().Server().String(); got != want {
			t.Errorf("slot %q: server %s, want %s", slot, got, want)
		}
	}
}

func TestKeys(t *testing.T) {
	tests := []struct {
		key, value string
		want       bool
	}{
		{"WS_USER", "ada", true},
		{"WS_TARGET", "mac-vm", true},
		{"WS_TARGET", "vm", false},
		{"WS_SLOT", "0", true},
		{"WS_SLOT", "15", true},
		{"WS_SLOT", "16", false},
		{"WS_SLOT", "07", false},
		{"WS_SLOT", "-1", false},
		{"WS_SLOT", "+1", false},
		{"WS_PUBLIC_KEY", "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=", true},
		{"WS_LAN", "10.88.0.0/24", true},
		{"WS_LAN", "10.99.0.0/16", false},
		{"WS_LAN", "10.99.8.0/24", false},
		{"WS_LAN", "10.77.0.0/30", false},
		{"WS_LAN", "10.88.0.1/24", false},
		{"WS_LAN_ADDRESS", "10.88.0.2", true},
		{"WS_REMOTE", "vm.example.com", true},
		{"WS_REMOTE_PORT", "22", true},
		{"WS_PORT", "51820", true},
		{"WS_TUNNEL_NETWORKS", "10.20.0.0/16 10.30.0.0/16", true},
		{"WS_TUNNEL_NETWORKS", "10.99.0.0/24", false},
		{"WS_TUNNEL_NETWORKS", "10.99.15.0/24", false},
		{"WS_TUNNEL_NETWORKS", "10.99.16.0/24", true},
		{"WS_RESOLVERS", "example.internal=10.20.0.53", true},
		{"WS_RESOLVERS", "Example.internal=10.20.0.53", false},
		{"WS_CABLE_PORT", "enp5s0", true},
		{"WS_CABLE_PORT", "-enp5s0", false},
		{"WS_GIT_EMAIL", "ws@example.com", true},
		{"WS_GIT_EMAIL", "ws", false},
		{"WS_GIT_SIGNING_PUB_KEY", "~/.ssh/ws.pub", true},
		{"WS_IDENTITY_PUB_KEY", "~/.ssh/ws", false},
	}
	for _, tt := range tests {
		if got := find(t, tt.key).ok(tt.value, false); got != tt.want {
			t.Errorf("%s=%q: %v, want %v", tt.key, tt.value, got, tt.want)
		}
	}
}

func TestTunnelPortAndForwardPortGiveTheirDefaults(t *testing.T) {
	for _, tc := range []struct {
		c             Conf
		port, forward string
	}{
		{Conf{}, "51820", "22022"},
		{Conf{Port: "51821", RemotePort: "22023"}, "51821", "22023"},
	} {
		if got := tc.c.TunnelPort(); got != tc.port {
			t.Errorf("%+v: TunnelPort = %s, want %s", tc.c, got, tc.port)
		}
		if got := tc.c.ForwardPort(); got != tc.forward {
			t.Errorf("%+v: ForwardPort = %s, want %s", tc.c, got, tc.forward)
		}
	}
}
