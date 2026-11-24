package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/macos/roles/secrets"
	"github.com/zadykian/ws/macos/values"
)

// guestKeyFile is the ed25519 host key that a guest's sshd offers, which its host reads.
const guestKeyFile = "/etc/ssh/ssh_host_ed25519_key.pub"

// reach is how ws macos secrets add gets to a server before the tunnel is up. A guest has no
// address of its own, so its host answers. Add runs its commands in the guest by incus exec, and
// the host's own ws runs the guest's setup.
type reach struct {
	// name is the server that add acts on.
	name string
	// ssh is the server ssh logs in to: the server itself, or the guest's host.
	ssh string
	// target is ssh's arguments to ssh, with the first contact's address and port where given.
	target []string
	// host is the guest's host, and empty for a server that is no guest.
	host string
}

// reachOf returns how add reaches the server called name, from the host that its file names.
func reachOf(p enginecli.Env, name, address string, port int) (reach, error) {
	c, err := values.Read(p.Machine, values.ServerPath(values.Path(p.Getenv, p.Pwd), name))
	if err != nil {
		return reach{}, err
	}
	login := name
	if c.Host != "" {
		login = c.Host
	}
	return reach{name: name, ssh: login, target: sshTarget(login, address, port), host: c.Host}, nil
}

// command returns ssh's arguments that run args at the server: in the guest by incus exec, where
// the server is one.
func (r reach) command(args ...string) []string {
	if r.host == "" {
		return append(append([]string(nil), r.target...), args...)
	}
	return append(append(append([]string(nil), r.target...), "incus", "exec", r.name, "--"), args...)
}

// setup returns the command that runs the server's setup with the secrets, as root: the server's
// own ws, or the host's ws pushing the guest's.
func (r reach) setup() []string {
	if r.host == "" {
		return serverSecretsRun
	}
	return guestSecretsRun(r.name)
}

// hostTrusted refuses a guest whose host known_hosts holds no key of. Each ssh to the guest goes to
// the host, so that ssh's first-use prompt would decide the host and the console's proof is lost.
func hostTrusted(c *cobra.Command, p enginecli.Env, r reach) error {
	key, err := secrets.HostKey(c.Context(), p.Machine, p.Getenv("HOME"), r.host)
	if err != nil {
		return err
	}
	if key == "" {
		return fmt.Errorf("%s holds no host key of %s, the host of %s: run ws macos secrets add %s first",
			secrets.KnownHosts(p.Getenv("HOME")), r.host, r.name, r.host)
	}
	return nil
}

// guestHostKey reads the guest's ed25519 host key through its host, and records it in known_hosts
// under the guest's name. No fingerprint is compared: the host that known_hosts trusts runs the
// guest.
func guestHostKey(c *cobra.Command, p enginecli.Env, r reach) (string, error) {
	out, err := p.Machine.Run(c.Context(), "ssh", r.command("cat", guestKeyFile)...)
	if err != nil {
		return "", fmt.Errorf("cannot read the host key of %s through %s by ssh: %w", r.name, r.host, err)
	}
	key, err := secrets.PubKey(out)
	if err != nil {
		return "", fmt.Errorf("the host key of %s: %w", r.name, err)
	}
	if key == "" {
		return "", fmt.Errorf("the guest %s offers no ed25519 host key", r.name)
	}
	if err := secrets.RecordHostKey(p.Machine, p.Getenv("HOME"), r.name, key); err != nil {
		return "", err
	}
	_, _ = fmt.Fprintf(c.ErrOrStderr(), "recorded the host key of %s, which %s read, in %s\n", r.name, r.host,
		secrets.KnownHosts(p.Getenv("HOME")))
	return key, nil
}
