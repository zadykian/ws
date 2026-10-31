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

func TestKeys(t *testing.T) {
	tests := []struct {
		key, value string
		want       bool
	}{
		{"WS_USER", "ada", true},
		{"WS_TARGET", "mac-vm", true},
		{"WS_TARGET", "vm", false},
		{"WS_PUBLIC_KEY", "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=", true},
		{"WS_LAN", "10.88.0.0/24", true},
		{"WS_LAN", "10.99.0.0/16", false},
		{"WS_LAN", "10.77.0.0/30", false},
		{"WS_LAN", "10.88.0.1/24", false},
		{"WS_LAN_ADDRESS", "10.88.0.2", true},
		{"WS_REMOTE", "vm.example.com", true},
		{"WS_REMOTE_PORT", "22", true},
		{"WS_PORT", "51820", true},
	}
	for _, tt := range tests {
		if got := find(t, tt.key).ok(tt.value, false); got != tt.want {
			t.Errorf("%s=%q: %v, want %v", tt.key, tt.value, got, tt.want)
		}
	}
}
