// Package facts reads what the Mac's specs are filled with: the user, the home and each server's
// values, which the Mac's ws unit hands to its units.
package facts

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/internal/sshcfg"
	"github.com/zadykian/ws/macos/internal/keys"
	"github.com/zadykian/ws/macos/roles/ghostty"
	"github.com/zadykian/ws/macos/roles/git"
	"github.com/zadykian/ws/macos/roles/ssh"
	"github.com/zadykian/ws/macos/values"
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
	// Ikeward is ikeward as mise installed it for the tree.
	Ikeward Ikeward
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

// Ikeward is ikeward as mise installed it for the tree, which its unit copies into /usr/local/sbin.
// From is empty where mise has none: that unit alone fails then.
type Ikeward struct {
	From string
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

// Read reads the facts from the environment, Ghostty's configs and the values' files. It reads
// values.yml from WS_VALUES or /etc/ws first, then each server's file beside it. A bad values.yml
// fails the read alone. Any other refusal, a missing key or a config that ssh refuses fails it
// too, and mise stops before a change. A Mac with no server yet reads none.
func Read(ctx context.Context, m effect.Machine, getenv func(string) string, pwd string) (Mac, error) {
	f := Mac{User: getenv("USER"), Home: getenv("HOME")}
	if f.User == "" || !path.IsAbs(f.Home) {
		return Mac{}, fmt.Errorf("USER %q and HOME %q name no user: run it from the user's own shell", f.User, f.Home)
	}
	f.Ghostty.Config = ghostty.ConfigPath(m, f.Home)
	mac := values.Path(getenv, pwd)
	fleet := values.ReadFleet(m, mac)
	var err error
	if f.Servers, err = serversOf(fleet); err != nil {
		return Mac{}, err
	}
	own := fleet.Mac
	if f.SSH, err = f.readSSH(ctx, m, own); err != nil {
		return Mac{}, err
	}
	signing, err := keys.ReadKey(ctx, m, own, f.Home, keys.SigningKey)
	if err != nil {
		return Mac{}, err
	}
	f.Git = Git{Texts: git.TextsOf(f.Home, signing), GhFrom: toolFrom(ctx, m, "gh")}
	f.Secrets = Secrets{FnoxFrom: toolFrom(ctx, m, "fnox"), PluginFrom: toolFrom(ctx, m, "age-plugin-yubikey")}
	f.Ikeward = Ikeward{From: toolFrom(ctx, m, "ikeward")}
	return f, nil
}

// Servers reads each server's file beside values.yml at mac, by name, with the Mac's keys joined
// in. It reads values.yml first, and a refusal of it ends the read in that file's words. Then it
// runs the whole check of the values: a bad value, a bad name or a clash between two servers fails
// it, naming each server it refuses. A Mac with no server yet has none.
func Servers(m effect.Machine, mac string) ([]Server, error) {
	return serversOf(values.ReadFleet(m, mac))
}

// serversOf returns the servers that nothing refuses, with the refusal of each that something does.
func serversOf(fleet values.Fleet) ([]Server, error) {
	if fleet.MacErr != nil {
		return nil, fleet.MacErr
	}
	if fleet.NamesErr != nil {
		return nil, fleet.NamesErr
	}
	var servers []Server
	for _, s := range fleet.Accepted() {
		servers = append(servers, Server{Name: s.Name, Conf: s.Conf})
	}
	return servers, fleet.Refused()
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
	out, err := m.Run(ctx, "mise", "-C", "/opt/ws/macos", "which", name)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
