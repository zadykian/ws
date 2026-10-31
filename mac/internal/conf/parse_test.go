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
		{"empty", "", Conf{User: "root", Target: "lan"}, nil},
		{"values", "WS_USER=ada\nWS_TARGET='lan'\n  WS_PORT=1\n",
			Conf{User: "ada", Target: "lan", Port: "1"}, nil},
		{"quotes", "WS_GIT_NAME=\"Ada L\"\nWS_GIT_EMAIL='a@b.c'", Conf{User: "root", Target: "lan",
			GitName: "Ada L", GitEmail: "a@b.c"}, nil},
		{"comments", "# a\n\n  \t# b\n", Conf{User: "root", Target: "lan"}, nil},
		{"later line", "WS_USER=a\nWS_USER=b\nWS_USER=\n", Conf{User: "root", Target: "lan"}, nil},
		{"empty values", "WS_USER=\nWS_TARGET=\"\"\nWS_PORT=\n", Conf{User: "root", Target: "lan"}, nil},
		{"unknown key", "WS_NOPE=1\n", Conf{User: "root", Target: "lan"},
			[]Error{{1, "unknown key WS_NOPE"}}},
		{"not a line", "ws_user=a\nWS_USER\n=a\n", Conf{User: "root", Target: "lan"},
			[]Error{{1, "not a KEY=value line"}, {2, "not a KEY=value line"}, {3, "not a KEY=value line"}}},
		{"bare blank", "WS_GIT_NAME=Ada L\n", Conf{User: "root", Target: "lan"},
			[]Error{{1, "WS_GIT_NAME=Ada L is not " + find(t, "WS_GIT_NAME").form}}},
		{"carriage return", "WS_USER=a\r\n", Conf{User: "root", Target: "lan"},
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
