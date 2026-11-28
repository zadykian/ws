package cli

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
)

// newSSHKeygenCmd returns ws ubuntu ssh keygen.
func newSSHKeygenCmd(p enginecli.Env) *cobra.Command {
	return &cobra.Command{
		Use:   "keygen NAME",
		Short: "make root's ed25519 key NAME, with no passphrase, or its .pub where the key alone is there",
		Long: "It makes /root/.ssh/NAME at 0600 and NAME.pub at 0644, and /root/.ssh at 0700 where it is missing. " +
			"Where the key is there and its .pub is not, it writes the .pub again by ssh-keygen -y and exits 0. " +
			"Where both are there, it changes nothing and exits 1. So it does where the .pub alone is there: run " +
			"remove first. It refuses a link at either file or at /root/.ssh.",
		Args: sshNameArg,
		RunE: func(c *cobra.Command, args []string) error {
			out, flush := sshOut(c)
			defer flush()
			return sshKeygen(c.Context(), p.Machine, out, args[0])
		},
	}
}

// sshKeygen makes the key NAME and its .pub, or its .pub alone, as the files in ~/.ssh stand. Each
// file is made with no replace, so that a keygen of the same name at the same time loses and fails.
func sshKeygen(ctx context.Context, m effect.Machine, out effect.Out, name string) error {
	key, hasKey, err := sshFile(m, name)
	if err != nil {
		return err
	}
	_, hasPub, err := sshFile(m, name+".pub")
	if err != nil {
		return err
	}
	a, err := rootUser.Lookup(ctx, m)
	if err != nil {
		return err
	}
	dir, keyPath, pubPath := a.Dir(m, sshDir, 0o700), sshDir+"/"+name, sshDir+"/"+name+".pub"
	dir.CreateOnly = true
	switch {
	case hasKey && hasPub:
		return fmt.Errorf("%s and %s.pub exist already, so nothing changes: run remove first to make a new key", name, name)
	case hasPub:
		return fmt.Errorf("%s.pub exists with no key %s: run remove %s, then keygen again", name, name, name)
	case hasKey:
		if !isPrivateKey(key) {
			return fmt.Errorf("%s/%s is no OpenSSH private key", a.Home, keyPath)
		}
		line, err := publicOf(ctx, m, a.Home+"/"+keyPath, name)
		if err != nil {
			return err
		}
		pub := a.File(m, pubPath, line, 0o644)
		pub.Create = true
		return settle(ctx, out, []resource.Resource{pub}, false)
	}
	private, line, err := newKey(name)
	if err != nil {
		return err
	}
	keyFile := a.File(m, keyPath, private, 0o600)
	keyFile.Secret, keyFile.Create = true, true
	pub := a.File(m, pubPath, line, 0o644)
	pub.Create = true
	return settle(ctx, out, []resource.Resource{dir, keyFile, pub}, false)
}

// publicOf returns the .pub's line of the key at path, as ssh-keygen -y prints its key and name.
func publicOf(ctx context.Context, m effect.Machine, path, name string) ([]byte, error) {
	got, err := m.Run(ctx, "ssh-keygen", "-y", "-P", "", "-f", path)
	if err != nil {
		return nil, fmt.Errorf("ssh-keygen -y cannot read the public key of %s: %w", path, err)
	}
	fields := strings.Fields(string(got))
	if len(fields) < 2 {
		return nil, fmt.Errorf("ssh-keygen -y printed no public key for %s", path)
	}
	return []byte(fields[0] + " " + fields[1] + " " + name + "\n"), nil
}

// newKey makes an ed25519 key with no passphrase, and returns its OpenSSH private file and the
// line of its .pub, which ends in the name.
func newKey(name string) (private, line []byte, err error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	block, err := ssh.MarshalPrivateKey(priv, name)
	if err != nil {
		return nil, nil, err
	}
	public, err := ssh.NewPublicKey(pub)
	if err != nil {
		return nil, nil, errors.Join(errors.New("make the public key"), err)
	}
	text := strings.TrimSuffix(string(ssh.MarshalAuthorizedKey(public)), "\n")
	return pem.EncodeToMemory(block), []byte(text + " " + name + "\n"), nil
}
