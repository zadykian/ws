package values

import (
	"strings"
	"testing"
)

func TestLine(t *testing.T) {
	for _, tt := range []struct{ key, value, want string }{
		{"WS_SLOT", "0", "WS_SLOT: 0"},
		{"WS_REMOTE_PORT", "22022", "WS_REMOTE_PORT: 22022"},
		{"WS_CABLE_SERVICE", "USB LAN", "WS_CABLE_SERVICE: USB LAN"},
		{"WS_CABLE_SERVICE", "USB 10/100/1G/2.5G LAN", "WS_CABLE_SERVICE: USB 10/100/1G/2.5G LAN"},
		{"WS_RESOLVERS", "example.internal=10.20.0.53", "WS_RESOLVERS: example.internal=10.20.0.53"},
		{"WS_PUBLIC_KEY", "ab+/cd=", "WS_PUBLIC_KEY: ab+/cd="},
		{"WS_LAN", "", `WS_LAN: ""`},
		{"WS_TUNNEL_PORT", "a: b", `WS_TUNNEL_PORT: "a: b"`},
		{"WS_TUNNEL_PORT", "#x", `WS_TUNNEL_PORT: "#x"`},
		{"WS_TUNNEL_PORT", "~", `WS_TUNNEL_PORT: "~"`},
		{"WS_TUNNEL_PORT", "null", `WS_TUNNEL_PORT: "null"`},
		{"WS_TUNNEL_PORT", " x", `WS_TUNNEL_PORT: " x"`},
		{"WS_TUNNEL_PORT", "x ", `WS_TUNNEL_PORT: "x "`},
		{"WS_TUNNEL_PORT", "@x", `WS_TUNNEL_PORT: "@x"`},
		{"WS_TUNNEL_PORT", "%x", `WS_TUNNEL_PORT: "%x"`},
		{"WS_TUNNEL_PORT", "-x", `WS_TUNNEL_PORT: "-x"`},
		{"WS_TUNNEL_PORT", "a\"b", `WS_TUNNEL_PORT: "a\"b"`},
	} {
		if got := Line(tt.key, tt.value); got != tt.want {
			t.Errorf("Line(%q, %q) = %q, want %q", tt.key, tt.value, got, tt.want)
		}
	}
}

// TestLineReadsBackAsItself holds the writer to the reader: each value comes back from entries as
// it went in.
func TestLineReadsBackAsItself(t *testing.T) {
	for _, value := range []string{
		"0", "22022", "USB 10/100/1G/2.5G LAN", "~/.ssh/k.pub", "example.internal=10.20.0.53", "",
		"a: b", "#x", "~", "null", "Null", " x", "x ", "@x", "%x", "-x", "a, b", "yes", "1e3", "a\tb",
		"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA+/=", "a'b", `a"b`,
	} {
		got, errs := entries(Line("WS_TUNNEL_PORT", value) + "\n")
		if len(errs) > 0 || len(got) != 1 || got[0].err != "" || got[0].value != value {
			t.Errorf("%q came back as %+v, %v", value, got, errs)
		}
	}
}

func TestText(t *testing.T) {
	tests := []struct {
		name  string
		pairs []string
		want  string
	}{
		{"no pairs", nil, ""},
		{"one", []string{"WS_TARGET", "lan"}, Line("WS_TARGET", "lan") + "\n"},
		{"several in order", []string{"WS_SLOT", "0", "WS_LAN", "10.0.0.0/24"},
			Line("WS_SLOT", "0") + "\n" + Line("WS_LAN", "10.0.0.0/24") + "\n"},
		{"an empty value", []string{"WS_LAN", ""}, Line("WS_LAN", "") + "\n"},
		{"a value with a blank", []string{"WS_CABLE_SERVICE", "USB LAN"}, Line("WS_CABLE_SERVICE", "USB LAN") + "\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Text(tt.pairs...); got != tt.want {
				t.Errorf("Text = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestTextOddCount(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("Text(\"WS_LAN\") did not panic")
		}
	}()
	Text(strings.Fields("WS_LAN")...)
}
