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
		{"empty", "", Conf{User: "zadykian", Target: "lan"}, nil},
		{"values", "WS_USER=ada\nWS_TARGET='lan'\n  WS_PORT=1\n",
			Conf{User: "ada", Target: "lan", Port: "1"}, nil},
		{"quotes", "WS_GIT_NAME=\"Ada L\"\nWS_GIT_EMAIL='a@b.c'", Conf{User: "zadykian", Target: "lan",
			GitName: "Ada L", GitEmail: "a@b.c"}, nil},
		{"comments", "# a\n\n  \t# b\n", Conf{User: "zadykian", Target: "lan"}, nil},
		{"later line", "WS_USER=a\nWS_USER=b\nWS_USER=\n", Conf{User: "zadykian", Target: "lan"}, nil},
		{"empty values", "WS_USER=\nWS_TARGET=\"\"\nWS_PORT=\n", Conf{User: "zadykian", Target: "lan"}, nil},
		{"cloud", "WS_TARGET=cloud\nWS_REMOTE=h\nWS_REMOTE_PORT=22\n", Conf{User: "root", Target: "cloud", Remote: "h", RemotePort: "22"}, nil},
		{"cloud, last line", "WS_USER=\nWS_TARGET=cloud\nWS_REMOTE=h\nWS_REMOTE_PORT=22\n", Conf{User: "root", Target: "cloud", Remote: "h", RemotePort: "22"}, nil},
		{"unknown key", "WS_NOPE=1\n", Conf{User: "zadykian", Target: "lan"},
			[]Error{{1, "unknown key WS_NOPE"}}},
		{"the removed key", "WS_HOOK=me/tools@v1.0.0\n", Conf{User: "zadykian", Target: "lan"},
			[]Error{{1, "unknown key WS_HOOK"}}},
		{"not a line", "ws_user=a\nWS_USER\n=a\n", Conf{User: "zadykian", Target: "lan"},
			[]Error{{1, "not a KEY=value line"}, {2, "not a KEY=value line"}, {3, "not a KEY=value line"}}},
		{"bare blank", "WS_GIT_NAME=Ada L\n", Conf{User: "zadykian", Target: "lan"},
			[]Error{{1, "WS_GIT_NAME=Ada L is not " + find(t, "WS_GIT_NAME").form}}},
		{"carriage return", "WS_USER=a\r\n", Conf{User: "zadykian", Target: "lan"},
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
