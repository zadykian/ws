// Package zed is the zed role as Go steps: the settings of Zed's server for Go and C#, and the
// open-file limit of ws_user's logins. See docs/zed.md.
package zed

import (
	"embed"
	"encoding/json"
	"fmt"
	"runtime"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
	"github.com/zadykian/ws/internal/secrets"
	"github.com/zadykian/ws/internal/server/devtools"
	"github.com/zadykian/ws/internal/server/values"
	"github.com/zadykian/ws/step"
)

//go:embed settings.json.tmpl
var files embed.FS

//go:embed files/terminal files/tmux.conf
var terminalFiles embed.FS

// terminalDir is where the terminal wrapper and its tmux.conf live: root's, outside any home, as Zed
// runs the wrapper by its path. The wrapper finds its config beside itself.
const terminalDir = "/usr/local/lib/ws/zed"

// terminalPath is the wrapper, which the settings name as the terminal's program.
const terminalPath = terminalDir + "/terminal"

// Steps returns the role's steps in order.
func Steps(v values.Values, _ secrets.Secrets, m effect.Machine, _ effect.Out) []step.Step {
	user := host.User{Name: v.User, Home: v.Home}
	return []step.Step{
		host.Group{Title: "Zed terminal", Parts: terminalParts(m)},
		user.Owned(m, "Zed settings", "Zed terminal", func(uid, gid int) ([]host.Resource, error) {
			settings, err := settingsFile(v)
			if err != nil {
				return nil, err
			}
			dir := v.Home + "/.config/zed"
			return []host.Resource{
				host.Plain{M: m, Home: v.Home, Paths: []string{dir + "/settings.json"}},
				host.Dir{M: m, Path: v.Home + "/.config", Mode: 0o755, UID: uid, GID: gid, CreateOnly: true},
				host.Dir{M: m, Path: dir, Mode: 0o755, UID: uid, GID: gid},
				host.File{M: m, Path: dir + "/settings.json", Content: settings, Mode: 0o644, UID: uid, GID: gid,
					Backup: true},
			}, nil
		}),
		host.Group{Title: "Zed open-file limit", Parts: []host.Resource{
			host.File{M: m, Path: limitsPath, Content: limitsFile(v), Mode: 0o644},
		}},
		download(m, runtime.GOARCH, "postgres-language-server", pgls, pglsURL),
		download(m, runtime.GOARCH, "docker-language-server", dockerLS, dockerLSURL),
		user.Owned(m, "Zed npm servers", "", func(uid, gid int) ([]host.Resource, error) {
			return serverFiles(m, user, uid, gid)
		}),
	}
}

// limitsPath is where the limit for ws_user's logins lives. limits.conf(5) says a `*` line does
// not apply to root, which a cloud VM may use as ws_user, so the user is named.
const limitsPath = "/etc/security/limits.d/ws-zed.conf"

// limitsFile raises ws_user's soft open-file limit: sshd's logins start at 1024, past which Zed's
// server busy-loops (zed#62995).
func limitsFile(v values.Values) []byte {
	return []byte(fmt.Sprintf("# Written by github.com/zadykian/ws, whose next run replaces any change made here.\n"+
		"%s soft nofile %d\n", v.User, v.Zed.Nofile))
}

// quote returns the string as a JSON string.
func quote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// settingsFile renders ~/.config/zed/settings.json: the servers by path, as devtools installs
// them. The server's file wins over the Mac's, which Zed forwards.
func settingsFile(v values.Values) ([]byte, error) {
	return host.Render(files, "settings.json.tmpl", struct{ Terminal, Gopls, Roslyn, Dlv, AnsibleLS, HTMLLS, DockerfileLS string }{
		Terminal:     quote(terminalPath),
		Gopls:        quote(devtools.Gopls(v.Home)),
		Roslyn:       quote(devtools.Roslyn(v.Home)),
		Dlv:          quote(devtools.Dlv(v.Home)),
		AnsibleLS:    quote(serverBin(v.Home, "ansible-language-server")),
		HTMLLS:       quote(serverBin(v.Home, "vscode-html-language-server")),
		DockerfileLS: quote(serverBin(v.Home, "docker-langserver")),
	})
}

// terminalParts installs the wrapper at mode 0755 and its tmux.conf at 0644. Both parents are made
// here: --only zed may run where security has not made /usr/local/lib/ws.
func terminalParts(m effect.Machine) []host.Resource {
	script, _ := terminalFiles.ReadFile("files/terminal")
	conf, _ := terminalFiles.ReadFile("files/tmux.conf")
	return []host.Resource{
		host.Dir{M: m, Path: "/usr/local/lib/ws", Mode: 0o755, CreateOnly: true},
		host.Dir{M: m, Path: terminalDir, Mode: 0o755},
		host.File{M: m, Path: terminalPath, Content: script, Mode: 0o755},
		host.File{M: m, Path: terminalDir + "/tmux.conf", Content: conf, Mode: 0o644},
	}
}
