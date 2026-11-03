package zed

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/host"
)

const wsDir = "/home/ws/.local/lib/zed-servers"

// npmCommand is the command line that the fake machine answers for the user's npm ci.
func npmCommand(user, home string) string {
	switch user {
	case "root":
		return "env HOME=" + home + " " + npmBin + " ci --prefix " + serversDir(home) +
			" --ignore-scripts --no-audit --no-fund"
	default:
		return "runuser -u " + user + " -- env HOME=" + home + " " + npmBin + " ci --prefix " +
			serversDir(home) + " --ignore-scripts --no-audit --no-fund"
	}
}

func newInstall(m *effecttest.Machine) npmInstall {
	return npmInstall{M: m, User: host.User{Name: "ws", Home: "/home/ws"}, UID: 1000, GID: 1000, Digest: "abc"}
}

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

func TestNpmPlansAChangeForAMissingNpmOrStamp(t *testing.T) {
	cases := []struct {
		name  string
		files map[string]string
		want  int
	}{
		{"no npm", map[string]string{}, 1},
		{"no stamp", map[string]string{npmBin: "x"}, 1},
		{"other digest", map[string]string{npmBin: "x", wsDir + "/node_modules/.ws-sha256": "old\n"}, 1},
		{"same digest", map[string]string{npmBin: "x", wsDir + "/node_modules/.ws-sha256": "abc\n"}, 0},
	}
	for _, c := range cases {
		m := effecttest.NewMachine()
		for p, body := range c.files {
			m.Files[p] = []byte(body)
		}
		got, err := newInstall(m).Plan(context.Background())
		if err != nil || len(got) != c.want {
			t.Errorf("%s: %d changes, %v; want %d", c.name, len(got), err, c.want)
		}
	}
}
