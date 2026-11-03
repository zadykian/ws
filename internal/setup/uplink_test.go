package setup

import "testing"

func TestUplinkRouter(t *testing.T) {
	for info, want := range map[string]string{
		"Router: (null)":      "",
		"Router: 10.77.0.1":   "",
		"Router: 10.0.0.1":    "10.0.0.1",
		"Router: 1.2.3":       "1.2.3",
		"IP address: 1.2.3.4": "",
	} {
		if got := uplinkRouter(info); got != want {
			t.Errorf("uplinkRouter(%q) = %q, want %q", info, got, want)
		}
	}
}
