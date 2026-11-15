package values

import "testing"

func TestPath(t *testing.T) {
	for file, want := range map[string]string{"": "/h/.config/ws/mac.conf", "/x/m.conf": "/x/m.conf", "m.conf": "/w/m.conf"} {
		if got := Path(func(k string) string { return map[string]string{"HOME": "/h", "WS_VALUES": file}[k] }, "/w"); got != want {
			t.Errorf("WS_VALUES=%q: Path = %q, want %q", file, got, want)
		}
	}
}
