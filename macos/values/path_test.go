package values

import "testing"

func TestDir(t *testing.T) {
	for dir, want := range map[string]string{"": "/etc/ws", "/x/ws": "/x/ws", "ws": "/w/ws"} {
		if got := Dir(func(k string) string { return map[string]string{"HOME": "/h", "WS_VALUES": dir}[k] }, "/w"); got != want {
			t.Errorf("WS_VALUES=%q: Dir = %q, want %q", dir, got, want)
		}
	}
}
