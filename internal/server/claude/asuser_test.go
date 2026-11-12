package claude

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/host"
)

// serveRead fakes the command that readAsUser runs for the user's file. A nil body is no file.
func serveRead(m *effecttest.Machine, u host.User, file string, body []byte) {
	out := []byte("n")
	if body != nil {
		out = append([]byte("y"), body...)
	}
	line := "env HOME=" + u.Home + " sh -c " + readScript + " sh " + file
	if u.Name != "root" {
		line = "runuser -u " + u.Name + " -- env -C " + u.Home + " " + strings.TrimPrefix(line, "env ")
	}
	m.Results[line] = effecttest.Result{Out: out}
}

// A link at the file or at a directory above it reaches only what the user could read, so the
// reads go as the user and never as root.
func TestReadsGoAsTheUserThroughLinks(t *testing.T) {
	user := host.User{Name: "ws", Home: "/home/ws"}
	settings := user.Home + "/.claude/settings.json"
	script := user.Home + "/.local/share/bash-completion/completions/cld"
	for _, link := range []string{settings, user.Home + "/.claude", script, user.Home + "/.local/share"} {
		m := effecttest.NewMachine()
		m.Links[link] = "/srv/target"
		m.Files["/srv/target"] = []byte("root's data")
		m.Files[user.Home+"/.local/bin/cld"] = nil
		serveRead(m, user, settings, []byte("{}"))
		serveRead(m, user, script, []byte("old\n"))
		m.Results["runuser -u ws -- env -C /home/ws HOME=/home/ws /home/ws/.local/bin/cld completion bash"] =
			effecttest.Result{Out: []byte("new\n")}
		if _, err := (Telemetry{M: m, User: user, Env: map[string]string{"A": "1"}}).Plan(t.Context()); err != nil {
			t.Fatalf("telemetry, link at %s: %v", link, err)
		}
		if got, err := (Completion{M: m, User: user}).Plan(t.Context()); err != nil || len(got) != 1 {
			t.Fatalf("completion, link at %s: %v %v", link, got, err)
		}
		for _, call := range m.Calls {
			if call == "read "+settings || call == "read root "+script {
				t.Errorf("link at %s: %q reads as root or as ws itself", link, call)
			}
		}
	}
}
