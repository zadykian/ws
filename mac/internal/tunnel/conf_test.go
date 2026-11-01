package tunnel

import (
	"net/netip"
	"strings"
	"testing"
)

func TestParseConf(t *testing.T) {
	key := strings.Repeat("A", 43) + "="
	full := "WS_PUBLIC_KEY=" + key + "\nWS_LAN=192.168.1.0/24\nWS_LAN_ADDRESS=192.168.1.10\n" +
		"WS_REMOTE=ws.example.com\nWS_PORT=51000\n"
	tests := []struct {
		name string
		in   string
		want Conf
		err  string
	}{
		{"key alone", "WS_PUBLIC_KEY=" + key, Conf{Key: key, Port: 51820}, ""},
		{"all", full, Conf{
			Key: key, LAN: netip.MustParsePrefix("192.168.1.0/24"),
			LANServer: netip.MustParseAddr("192.168.1.10"), Remote: "ws.example.com", Port: 51000,
		}, ""},
		{"comments, blanks and spaces", "# a\n\n \t WS_PORT=80 \t\nWS_PUBLIC_KEY=" + key + "\n",
			Conf{Key: key, Port: 80}, ""},
		{"a later line wins", "WS_PORT=80\nWS_PORT=81\nWS_PUBLIC_KEY=" + key, Conf{Key: key, Port: 81}, ""},
		{"no key", "WS_PORT=80\n", Conf{}, "tunnel.conf has no WS_PUBLIC_KEY"},
		{"empty", "", Conf{}, "tunnel.conf has no WS_PUBLIC_KEY"},
		{"bad key", "WS_PUBLIC_KEY=abc\n", Conf{}, "tunnel.conf: cannot take 'WS_PUBLIC_KEY=abc'"},
		{"bad network", "WS_LAN=192.168.1.5/24\n", Conf{}, "tunnel.conf: cannot take 'WS_LAN=192.168.1.5/24'"},
		{"bad port", "WS_PORT=0\n", Conf{}, "tunnel.conf: cannot take 'WS_PORT=0'"},
		{"unknown name", "WS_OTHER=1\n", Conf{}, "tunnel.conf: cannot take 'WS_OTHER=1'"},
		{"no =", "WS_PORT\n", Conf{}, "tunnel.conf: cannot take 'WS_PORT'"},
		{"empty value", "WS_REMOTE=\n", Conf{}, "tunnel.conf: cannot take 'WS_REMOTE='"},
		{"last line without a newline", "WS_PUBLIC_KEY=" + key + "\nWS_PORT=82", Conf{Key: key, Port: 82}, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseConf(tc.in)
			if tc.err != "" {
				if err == nil || err.Error() != tc.err {
					t.Fatalf("error %v, want %q", err, tc.err)
				}
				return
			}
			if err != nil || got != tc.want {
				t.Errorf("got %+v, %v; want %+v", got, err, tc.want)
			}
		})
	}
}
