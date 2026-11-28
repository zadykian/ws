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
	"github.com/zadykian/ws/macos/roles/ghostty"
	"github.com/zadykian/ws/macos/roles/git"
	"github.com/zadykian/ws/macos/roles/ssh"
	"github.com/zadykian/ws/macos/values"
)

// Mac is the data of the Mac's specs. A spec names a field as {{ .User }}.
type Mac struct {
	// User is the account that ws runs as, and Home its home.
	User, Home string
	// VM is true on a virtual machine, which has no Secure Enclave: seward's keys go in a file there.
	VM bool
	// Servers holds each server of servers/, by name.
	Servers []Server
	// Ghostty is what Ghostty's units need of the Mac.
	Ghostty Ghostty
	// SSH and Git are the texts of their files in /etc/ws, which their units write.
	SSH SSH
	Git Git
	// Secrets is the secrets' tools that mise installed for the tree.
	Secrets Secrets
}

// SSH is /etc/ws/ssh_config's text, checked as ssh reads it, and the seward that mise installed for
// the tree, which its unit copies into /usr/local/bin.
type SSH struct {
	Config string
	// SewardFrom is mise's seward, or empty where mise has none: seward's unit alone fails then.
	SewardFrom string
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

// Server is one server's values.
type Server struct {
	Name string
	values.Conf
}

// Read reads the facts from the environment, Ghostty's configs and the values' files. It reads
// each server's file in servers/ of WS_VALUES or /etc/ws. A refusal of one, or a config that ssh
// refuses, fails the read, and mise stops before a change. A Mac with no server yet reads none.
func Read(ctx context.Context, m effect.Machine, getenv func(string) string, pwd string) (Mac, error) {
	f := Mac{User: getenv("USER"), Home: getenv("HOME")}
	if f.User == "" || !path.IsAbs(f.Home) {
		return Mac{}, fmt.Errorf("USER %q and HOME %q name no user: run it from the user's own shell", f.User, f.Home)
	}
	f.VM = onVM(ctx, m)
	f.Ghostty.Config = ghostty.ConfigPath(m, f.Home)
	fleet := values.ReadFleet(m, values.Dir(getenv, pwd))
	var err error
	if f.Servers, err = serversOf(fleet); err != nil {
		return Mac{}, err
	}
	if f.SSH, err = f.readSSH(ctx, m); err != nil {
		return Mac{}, err
	}
	f.SSH.SewardFrom = toolFrom(ctx, m, "seward")
	f.Git = Git{Texts: git.TextsOf(f.Home), GhFrom: toolFrom(ctx, m, "gh")}
	f.Secrets = Secrets{FnoxFrom: toolFrom(ctx, m, "fnox"), PluginFrom: toolFrom(ctx, m, "age-plugin-yubikey")}
	return f, nil
}

// Servers reads each server's file in servers/ of dir, by name. It runs the whole check of the
// values: a bad value, a bad name or a clash between two servers fails it, naming each server it
// refuses. A Mac with no server yet has none.
func Servers(m effect.Machine, dir string) ([]Server, error) {
	return serversOf(values.ReadFleet(m, dir))
}

// serversOf returns the servers that nothing refuses, with the refusal of each that something does.
func serversOf(fleet values.Fleet) ([]Server, error) {
	if fleet.NamesErr != nil {
		return nil, fleet.NamesErr
	}
	var servers []Server
	for _, s := range fleet.Accepted() {
		servers = append(servers, Server{Name: s.Name, Conf: s.Conf})
	}
	return servers, fleet.Refused()
}

// readSSH returns ssh's text, a block for each server.
func (f Mac) readSSH(ctx context.Context, m effect.Machine) (SSH, error) {
	hosts := make([]sshcfg.Host, 0, len(f.Servers))
	for _, s := range f.Servers {
		hosts = append(hosts, sshcfg.Host{Name: s.Name, Addr: s.TunnelSlot().Server()})
	}
	text, err := ssh.Text(ctx, m, f.Home, hosts)
	return SSH{Config: text}, err
}

// onVM reports whether the hypervisor bit is set, as sysctl kern.hv_vmm_present prints 1. A sysctl
// that fails or prints anything else reads as a real Mac.
func onVM(ctx context.Context, m effect.Machine) bool {
	out, err := m.Run(ctx, "sysctl", "-n", "kern.hv_vmm_present")
	return err == nil && strings.TrimSpace(string(out)) == "1"
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
