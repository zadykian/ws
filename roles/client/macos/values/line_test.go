package values

import (
	"strings"
	"testing"
)

func TestLine(t *testing.T) {
	if got := Line("WS_CABLE_SERVICE", "USB LAN"); got != `WS_CABLE_SERVICE="USB LAN"` {
		t.Errorf("Line = %q", got)
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
