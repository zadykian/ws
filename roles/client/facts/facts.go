// Package facts reads what the Mac's specs are filled with: the user, the home and each server's
// values, which the Mac's ws unit hands to its units.
package facts

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/roles/client/internal/keys"
	"github.com/zadykian/ws/roles/client/macos/ghostty"
	"github.com/zadykian/ws/roles/client/macos/git"
	"github.com/zadykian/ws/roles/client/macos/ssh"
	"github.com/zadykian/ws/roles/client/values"
	"github.com/zadykian/ws/roles/internal/sshcfg"
)

// Mac is the data of the Mac's specs. A spec names a field as {{ .User }}.
type Mac struct {
	// User is the account that ws runs as, and Home its home.
	User, Home string
	// Servers holds each server of servers/ beside values.yml, by name.
	Servers []Server
	// Ghostty is what Ghostty's units need of the Mac.
	Ghostty Ghostty
	// SSH and Git are the texts of their files in /etc/ws, which their units write.
	SSH SSH
	Git Git
	// Secrets is the secrets' tools that mise installed for the tree.
	Secrets Secrets
}

// SSH is /etc/ws/ssh_config's text, checked as ssh reads it.
type SSH struct {
	Config string
}

// Git is git's files in /etc/ws, and the gh that mise installed for the tree, which its unit copies
// into /usr/local/bin.
type Git struct {
	git.Texts
	// GhFrom is mise's gh, or empty where mise has none: gh's unit alone fails then.
	GhFrom string
}

// Secrets is fnox and age-plugin-yubikey as mise installed them for the tree, which their units copy
// into /usr/local/bin. fnox runs the plugin from PATH, and the user runs fnox by hand. A path is
// empty where mise has none: that tool's unit alone fails then.
type Secrets struct {
	FnoxFrom, PluginFrom string
}

// Ghostty is the config that the line reading /etc/ws/ghostty goes in, which the Mac's files pick.
type Ghostty struct {
	Config string
}

// Server is one server's values, joined with the Mac's own keys.
type Server struct {
	Name string
	values.Conf
}

// Read reads the facts from the process's environment, Ghostty's configs and the values' files:
// values.yml from WS_VALUES or /etc/ws, and each server's file beside it. It runs the whole check
// of the values, reads the keys that values.yml names, and checks ssh's text as ssh reads it. A bad
// value, a bad name, a clash between two servers, a missing key or a config that ssh refuses fails
// every unit's read. mise then stops before a change. A Mac with no server yet reads none.
func Read(ctx context.Context, m effect.Machine, getenv func(string) string, pwd string) (Mac, error) {
	f := Mac{User: getenv("USER"), Home: getenv("HOME")}
	if f.User == "" || !path.IsAbs(f.Home) {
		return Mac{}, fmt.Errorf("USER %q and HOME %q name no user: run it from the user's own shell", f.User, f.Home)
	}
	f.Ghostty.Config = ghostty.ConfigPath(m, f.Home)
	mac := values.Path(getenv, pwd)
	servers, err := Servers(m, mac)
	if err != nil {
		return Mac{}, err
	}
	f.Servers = servers
	own, err := values.LoadMac(m, mac)
	if err != nil {
		return Mac{}, err
	}
	if f.SSH, err = f.readSSH(ctx, m, own); err != nil {
		return Mac{}, err
	}
	signing, err := keys.ReadKey(ctx, m, own, f.Home, keys.SigningKey)
	if err != nil {
		return Mac{}, err
	}
	f.Git = Git{Texts: git.TextsOf(f.Home, signing), GhFrom: toolFrom(ctx, m, "gh")}
	f.Secrets = Secrets{FnoxFrom: toolFrom(ctx, m, "fnox"), PluginFrom: toolFrom(ctx, m, "age-plugin-yubikey")}
	return f, nil
}

// Servers reads each server's file beside values.yml at mac, by name, with the Mac's keys joined
// in. It runs the whole check of the values: a bad value, a bad name or a clash between two servers
// fails it, naming each server it refuses. A Mac with no server yet has none.
func Servers(m effect.Machine, mac string) ([]Server, error) {
	names, err := values.Servers(m, mac)
	if err != nil {
		return nil, err
	}
	var servers []Server
	var all []error
	for _, name := range names {
		c, err := values.Load(m, mac, values.ServerPath(mac, name))
		if err != nil {
			all = append(all, values.RefuseSibling(name, err))
			continue
		}
		servers = append(servers, Server{Name: name, Conf: c})
	}
	return servers, errors.Join(all...)
}

// readSSH reads the login key and returns ssh's text, a block for each server.
func (f Mac) readSSH(ctx context.Context, m effect.Machine, own values.Conf) (SSH, error) {
	login, err := keys.ReadKey(ctx, m, own, f.Home, keys.LoginKey)
	if err != nil {
		return SSH{}, err
	}
	hosts := make([]sshcfg.Host, 0, len(f.Servers))
	for _, s := range f.Servers {
		hosts = append(hosts, sshcfg.Host{Name: s.Name, Addr: s.TunnelSlot().Server()})
	}
	text, err := ssh.Text(ctx, m, f.Home, login, hosts)
	return SSH{Config: text}, err
}

// toolFrom returns where mise installed the tool name for the tree, as the user's mise finds it,
// which is on no shell's PATH. A mise that has none yet gives none.
func toolFrom(ctx context.Context, m effect.Machine, name string) string {
	out, err := m.Run(ctx, "mise", "-C", "/opt/ws/roles/client", "which", name)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
