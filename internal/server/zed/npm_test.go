package zed

import (
	"strings"
	"testing"
)

const wsDir = "/home/ws/.local/lib/zed-servers"

func TestTheDirectoriesPlanInOrderForAHomeWithNoLocal(t *testing.T) {
	m := run(t, "ws", "/home/ws")
	var made []string
	for _, c := range m.Calls {
		if p, ok := strings.CutPrefix(c, "mkdir "); ok && strings.Contains(p, "/.local") {
			made = append(made, p)
		}
	}
	want := []string{"/home/ws/.local", "/home/ws/.local/lib", wsDir}
	if len(made) != 3 {
		t.Fatalf("made %v", made)
	}
	for i, p := range want {
		if !strings.HasPrefix(made[i], p) || m.Owners[p] != [2]int{1000, 1000} || m.Dirs[p] != 0o755 {
			t.Errorf("%s: made %q, owner %v, mode %v", p, made[i], m.Owners[p], m.Dirs[p])
		}
	}
	for _, name := range []string{"package.json", "package-lock.json"} {
		if len(m.Files[wsDir+"/"+name]) == 0 || m.Owners[wsDir+"/"+name] != [2]int{1000, 1000} {
			t.Errorf("%s missing or not the user's", name)
		}
	}
}
