package main

import "testing"

func TestConfPath(t *testing.T) {
	for conf, want := range map[string]string{"": "/h/.config/ws/mac.conf", "/x/m.conf": "/x/m.conf", "m.conf": "/w/m.conf"} {
		if got := confPath(func(k string) string { return map[string]string{"HOME": "/h", "WS_CONF": conf}[k] }, "/w"); got != want {
			t.Errorf("WS_CONF=%q: confPath = %q, want %q", conf, got, want)
		}
	}
}
