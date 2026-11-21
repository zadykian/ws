package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/spf13/cobra"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/roles/client/macos/secrets"
	"github.com/zadykian/ws/roles/client/values"
)

// dropSecrets takes the host key of the server called name out of the secrets repository, so that a
// removed server opens no value. The key leaves every table of recipients, whose values it encrypts
// again, then the repository's deploy keys. The key is NAME's ed25519 line in known_hosts. Without
// one, remove cannot tell which key is NAME's. add may never have run. Or ssh-keygen -R emptied the
// line, as add's advice after a rebuild says, while the key stays a recipient. byHand is a line for
// the user there, so that such a key stays in sight. byHand also tells the user where this Mac has
// no fnox key to encrypt with, as on a Mac where init never ran. So it does where skip leaves the
// secrets, as on a Mac that cannot reach them. It names the key then, as remove goes on to drop it
// from known_hosts.
func dropSecrets(c *cobra.Command, p enginecli.Env, name string, skip bool) (byHand string, err error) {
	ctx, m := c.Context(), p.Machine
	home := p.Getenv("HOME")
	if home == "" {
		return "", errors.New("HOME is not set: known_hosts has no place")
	}
	key, err := secrets.HostKey(ctx, m, home, name)
	if err != nil {
		return "", err
	}
	if key == "" {
		return fmt.Sprintf("By hand: known_hosts holds no host key of %s: where add ran for it, drop its hostname's "+
			"recipients and deploy key from the secrets", name), nil
	}
	if skip {
		return fmt.Sprintf("By hand: remove skipped the secrets: drop %s's host key, %s, from the secrets' "+
			"recipients and deploy keys", name, key), nil
	}
	keyFile, err := secrets.KeyFile(p.Getenv)
	if err != nil {
		return "", err
	}
	if _, err := m.Stat(keyFile); errors.Is(err, fs.ErrNotExist) {
		return fmt.Sprintf("By hand: this Mac has no fnox key, %s: from a Mac that has one, drop %s's host key, %s, "+
			"from the secrets' recipients and deploy keys", keyFile, name, key), nil
	} else if err != nil {
		return "", fmt.Errorf("cannot stat fnox's key file %s: %w", keyFile, err)
	}
	if err := dropHostKey(c, p, name, key, keyFile); err != nil {
		return "", fmt.Errorf("%w: remove drops %s's host key from the secrets by git, gh and the Nano, or leaves "+
			"it to you by --skip-secrets", err, name)
	}
	return "", nil
}

// dropHostKey takes key out of the recipients in a temporary clone of the secrets repository, lands
// the change, and deletes each deploy key that holds key. name is the server's, for the change's body.
func dropHostKey(c *cobra.Command, p enginecli.Env, name, key, keyFile string) error {
	ctx, m, out := c.Context(), p.Machine, c.OutOrStdout()
	repo, err := secrets.RepoOf(m, secrets.Tree)
	if err != nil {
		return err
	}
	clone, err := secrets.Open(ctx, m, repo)
	if err != nil {
		return err
	}
	defer clone.Close(ctx)
	clone.FnoxEnv = secrets.FnoxEnv(p.Getenv, keyFile)
	// The deploy keys are read first, so that gh's login fails before the Nano asks for a touch.
	keys, err := secrets.DeployKeys(ctx, m, secrets.GH, repo.Slug())
	if err != nil {
		return err
	}
	files, err := secrets.ReadFiles(m, clone.Dir)
	if err != nil {
		return err
	}
	files, changed, labels, err := secrets.DropKey(files, key)
	if err != nil {
		return err
	}
	if err := secrets.WriteFiles(m, clone.Dir, files, changed); err != nil {
		return err
	}
	if err := reencrypt(ctx, clone, files, changed); err != nil {
		return err
	}
	if len(changed) == 0 {
		_, _ = fmt.Fprintf(out, "the secrets' recipients hold no host key of %s\n", name)
	} else if _, err := clone.Land(ctx, out, secrets.Change{Topic: "remove-" + strings.Join(labels, "-"),
		Title: "chore: remove the host key of " + values.List(labels),
		Body:  "Removed by ws client remove " + name + ".", Files: changed}); err != nil {
		return err
	}
	// A key goes whatever its title, as a server whose hostname changed may hold it under another.
	for _, k := range keys {
		if k.Key != key {
			continue
		}
		if err := secrets.DeleteDeployKey(ctx, m, secrets.GH, repo.Slug(), k.ID); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(out, "deleted the deploy key %s of %s\n", k.Title, repo.Slug())
	}
	return nil
}
