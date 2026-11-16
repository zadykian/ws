package conf

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
		{"empty", "", Conf{User: "zadykian", Target: "lan", Slot: "0"}, nil},
		{"values", "WS_USER=ada\nWS_TARGET='lan'\n  WS_TUNNEL_PORT=1\n",
			Conf{User: "ada", Target: "lan", Slot: "0", Port: "1"}, nil},
		{"quotes", "WS_CABLE_SERVICE=\"USB LAN\"\nWS_CABLE_PORT='en1'", Conf{User: "zadykian", Target: "lan", Slot: "0",
			CableService: "USB LAN", CablePort: "en1"}, nil},
		{"comments", "# a\n\n  \t# b\n", Conf{User: "zadykian", Target: "lan", Slot: "0"}, nil},
		{"later line", "WS_USER=a\nWS_USER=b\nWS_USER=\n", Conf{User: "zadykian", Target: "lan", Slot: "0"}, nil},
		{"empty values", "WS_USER=\nWS_TARGET=\"\"\nWS_TUNNEL_PORT=\n", Conf{User: "zadykian", Target: "lan", Slot: "0"}, nil},
		{"cloud", "WS_TARGET=cloud\nWS_REMOTE=h\nWS_REMOTE_PORT=22\n", Conf{User: "root", Target: "cloud", Slot: "0", Remote: "h", RemotePort: "22"}, nil},
		{"cloud, last line", "WS_USER=\nWS_TARGET=cloud\nWS_REMOTE=h\nWS_REMOTE_PORT=22\n", Conf{User: "root", Target: "cloud", Slot: "0", Remote: "h", RemotePort: "22"}, nil},
		{"slot", "WS_SLOT=15\n", Conf{User: "zadykian", Target: "lan", Slot: "15"}, nil},
		{"empty slot", "WS_SLOT=3\nWS_SLOT=\n", Conf{User: "zadykian", Target: "lan", Slot: "0"}, nil},
		{"slot past 15", "WS_SLOT=16\n", Conf{User: "zadykian", Target: "lan", Slot: "0"},
			[]Error{{1, "WS_SLOT=16 is not " + find(t, "WS_SLOT").form}}},
		{"unknown key", "WS_NOPE=1\n", Conf{User: "zadykian", Target: "lan", Slot: "0"},
			[]Error{{1, "unknown key WS_NOPE"}}},
		{"the removed key", "WS_HOOK=me/tools@v1.0.0\n", Conf{User: "zadykian", Target: "lan", Slot: "0"},
			[]Error{{1, "unknown key WS_HOOK"}}},
		{"the old port key", "WS_PORT=51821\n", Conf{User: "zadykian", Target: "lan", Slot: "0"},
			[]Error{{1, "unknown key WS_PORT"}}},
		{"the removed name", "WS_GIT_NAME=Ada\nWS_GIT_EMAIL=a@b.c\n", Conf{User: "zadykian", Target: "lan", Slot: "0"},
			[]Error{{1, "unknown key WS_GIT_NAME"}, {2, "unknown key WS_GIT_EMAIL"}}},
		{"not a line", "ws_user=a\nWS_USER\n=a\n", Conf{User: "zadykian", Target: "lan", Slot: "0"},
			[]Error{{1, "not a KEY=value line"}, {2, "not a KEY=value line"}, {3, "not a KEY=value line"}}},
		{"bare blank", "WS_CABLE_SERVICE=USB LAN\n", Conf{User: "zadykian", Target: "lan", Slot: "0"},
			[]Error{{1, "WS_CABLE_SERVICE=USB LAN is not " + find(t, "WS_CABLE_SERVICE").form}}},
		{"carriage return", "WS_USER=a\r\n", Conf{User: "zadykian", Target: "lan", Slot: "0"},
			[]Error{{1, "WS_USER=a\r is not " + find(t, "WS_USER").form}}},
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
	if _, errs := Lines("WS_LAN=10.88.0.0/24\n"); len(errs) != 0 {
		t.Errorf("Lines refused a key that goes with another: %v", errs)
	}
	if _, errs := Parse("WS_LAN=10.88.0.0/24\n"); len(errs) != 1 {
		t.Errorf("Parse took a key that goes with another: %v", errs)
	}
}

func TestValue(t *testing.T) {
	c, errs := Lines("WS_GIT_SIGNING_PUB_KEY=\"~/k.pub\"\n")
	if len(errs) > 0 {
		t.Fatal(errs)
	}
	for name, want := range map[string]string{"WS_GIT_SIGNING_PUB_KEY": "~/k.pub", "WS_TARGET": "lan", "WS_LAN": "", "NOT_A_KEY": ""} {
		if got := c.Value(name); got != want {
			t.Errorf("Value(%s) = %q; want %q", name, got, want)
		}
	}
}
