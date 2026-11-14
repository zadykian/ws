package session

import "testing"

func TestSideOfNamesTheServer(t *testing.T) {
	for server, want := range map[string]string{"": "Server (Ubuntu)", "ws": "Server ws (Ubuntu)", "lab-2": "Server lab-2 (Ubuntu)"} {
		if got := string(SideOf(server)); got != want {
			t.Errorf("SideOf(%q) = %q; want %q", server, got, want)
		}
	}
}
