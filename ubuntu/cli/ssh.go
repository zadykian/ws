package cli

import (
	"bytes"
	"errors"
	"io/fs"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine"
	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/internal/sshkey"
)

// sshDir is root's directory of keys, below its home.
const sshDir = ".ssh"

// privateKeyHead opens an OpenSSH private key.
var privateKeyHead = []byte("-----BEGIN OPENSSH PRIVATE KEY-----")

// newSSHCmd returns ws ubuntu ssh: the keys in root's ~/.ssh, for a host and a guest alike.
func newSSHCmd(p enginecli.Env) *cobra.Command {
	return enginecli.Group("ssh", "the SSH keys in root's ~/.ssh, of a host and of a guest",
		newSSHKeygenCmd(p))
}

// sshNameArg checks that a command takes one name that internal/sshkey allows, before any path is
// built from it.
func sshNameArg(c *cobra.Command, args []string) error {
	if err := cobra.ExactArgs(1)(c, args); err != nil {
		return err
	}
	return sshkey.Check(args[0])
}

// sshFile reads a file of root's ~/.ssh by the account's guard, which refuses a link at the file or
// on the way. A missing file is not found, with no error.
func sshFile(m effect.Machine, rel string) (body []byte, found bool, err error) {
	body, err = rootUser.ReadHome(m, sshDir+"/"+rel)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, false, nil
	}
	return body, err == nil, err
}

// isPrivateKey tells an OpenSSH private key by its first line.
func isPrivateKey(body []byte) bool { return bytes.HasPrefix(body, privateKeyHead) }

// sshOut returns the lines a command says on its standard error, and the call that flushes them.
func sshOut(c *cobra.Command) (effect.Out, func()) {
	term := effect.NewTerminal(c.ErrOrStderr())
	return engine.Lines{Out: term, Prog: c.CommandPath(), Seen: map[string]bool{}}, term.Flush
}
