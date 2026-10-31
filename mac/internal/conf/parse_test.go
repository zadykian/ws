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
		{"values", "WS_USER=ada\nWS_TARGET='cloud'\n  WS_PORT=1\n",
			Conf{User: "ada", Target: "cloud", Port: "1"}, nil},
		{"quotes", "WS_GIT_NAME=\"Ada L\"\nWS_GIT_EMAIL='a@b.c'", Conf{User: "root", Target: "lan",
			GitName: "Ada L", GitEmail: "a@b.c"}, nil},
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
