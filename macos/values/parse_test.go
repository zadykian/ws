package values

import (
	"reflect"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name, text string
		want       Conf
		errs       []Error
	}{
		{"empty", "", Conf{Target: "lan", Slot: "0"}, nil},
		{"values", "WS_TARGET: 'lan'\nWS_TUNNEL_PORT: 1\n",
			Conf{Target: "lan", Slot: "0", Port: "1"}, nil},
		{"quotes", "WS_CABLE_SERVICE: \"USB LAN\"\nWS_CABLE_PORT: 'en1'", Conf{Target: "lan", Slot: "0",
			CableService: "USB LAN", CablePort: "en1"}, nil},
		{"comments", "# a\n\n  \t# b\n", Conf{Target: "lan", Slot: "0"}, nil},
		{"later line", "WS_TUNNEL_PORT: 1\nWS_TUNNEL_PORT: 2\n", Conf{Target: "lan", Slot: "0"},
			[]Error{{2, "WS_TUNNEL_PORT is set already, on line 1"}}},
		{"empty values", "WS_SLOT:\nWS_TARGET: \"\"\nWS_TUNNEL_PORT: ~\n", Conf{Target: "lan", Slot: "0"}, nil},
		{"cloud", Text("WS_TARGET", "cloud", "WS_REMOTE", "h", "WS_REMOTE_PORT", "22"), Conf{Target: "cloud", Slot: "0", Remote: "h", RemotePort: "22"}, nil},
		{"slot", Text("WS_SLOT", "15"), Conf{Target: "lan", Slot: "15"}, nil},
		{"empty slot", "WS_SLOT:\n", Conf{Target: "lan", Slot: "0"}, nil},
		{"slot past 15", Text("WS_SLOT", "16"), Conf{Target: "lan", Slot: "0"},
			[]Error{{1, "WS_SLOT=16 is not " + find(t, "WS_SLOT").form}}},
		{"unknown key", Text("WS_NOPE", "1"), Conf{Target: "lan", Slot: "0"},
			[]Error{{1, "unknown key WS_NOPE"}}},
		{"the removed key", Text("WS_HOOK", "me/tools@v1.0.0"), Conf{Target: "lan", Slot: "0"},
			[]Error{{1, "unknown key WS_HOOK"}}},
		{"the removed user", Text("WS_USER", "root"), Conf{Target: "lan", Slot: "0"},
			[]Error{{1, "unknown key WS_USER"}}},
		{"the old port key", Text("WS_PORT", "51821"), Conf{Target: "lan", Slot: "0"},
			[]Error{{1, "unknown key WS_PORT"}}},
		{"the removed name", Text("WS_GIT_NAME", "Ada", "WS_GIT_EMAIL", "a@b.c"), Conf{Target: "lan", Slot: "0"},
			[]Error{{1, "unknown key WS_GIT_NAME"}, {2, "unknown key WS_GIT_EMAIL"}}},
		{"not a line", "WS_TUNNEL_PORT\n", Conf{Target: "lan", Slot: "0"},
			[]Error{{1, oneLine}}},
		{"a lower-case key", "ws_slot: 1\n", Conf{Target: "lan", Slot: "0"},
			[]Error{{1, "unknown key ws_slot"}}},
		{"not YAML", "WS_TUNNEL_PORT: a: b\n", Conf{Target: "lan", Slot: "0"},
			[]Error{{1, "not YAML: mapping values are not allowed in this context"}}},
		{"a list", "WS_TUNNEL_PORT: [1]\n", Conf{Target: "lan", Slot: "0"},
			[]Error{{1, "WS_TUNNEL_PORT takes one value, not a list or a table"}}},
		{"the old line", "WS_TUNNEL_PORT=1\n", Conf{Target: "lan", Slot: "0"},
			[]Error{{1, oneLine}}},
		{"a bare blank", "WS_CABLE_SERVICE: USB LAN\n", Conf{Target: "lan", Slot: "0",
			CableService: "USB LAN"}, nil},
		{"CRLF line ends", "WS_TUNNEL_PORT: 2\r\nWS_SLOT: 1\r\n", Conf{Target: "lan", Slot: "1", Port: "2"}, nil},
	}
	for _, tt := range tests {
		got, errs := Parse(tt.text)
		if !reflect.DeepEqual(errs, tt.errs) {
			t.Errorf("%s: errors %v, want %v", tt.name, errs, tt.errs)
		}
		if len(errs) == 0 && got != tt.want {
			t.Errorf("%s: %+v, want %+v", tt.name, got, tt.want)
		}
	}
}

func TestLines(t *testing.T) {
	if _, errs := Lines(Text("WS_LAN", "10.88.0.0/24")); len(errs) != 0 {
		t.Errorf("Lines refused a key that goes with another: %v", errs)
	}
	if _, errs := Parse(Text("WS_LAN", "10.88.0.0/24")); len(errs) != 1 {
		t.Errorf("Parse took a key that goes with another: %v", errs)
	}
}

func TestValue(t *testing.T) {
	c, errs := Lines(Text("WS_REMOTE", "r.example"))
	if len(errs) > 0 {
		t.Fatal(errs)
	}
	for name, want := range map[string]string{"WS_REMOTE": "r.example", "WS_TARGET": "lan", "WS_LAN": "", "NOT_A_KEY": ""} {
		if got := c.Value(name); got != want {
			t.Errorf("Value(%s) = %q; want %q", name, got, want)
		}
	}
}
