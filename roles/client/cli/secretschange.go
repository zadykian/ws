package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/roles/client/macos/secrets"
)

// addHostKey makes key the recipient of hostname in a temporary clone of the secrets repository,
// keeps it as the repository's deploy key titled hostname, and lands the change. The deploy keys
// are read first, so that gh's login fails before the Nano asks for a touch. A change of no
// recipient lands nothing.
func addHostKey(c *cobra.Command, p enginecli.Env, name, hostname, key, keyFile string) error {
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
	keys, err := secrets.DeployKeys(ctx, m, secrets.GH, repo.Slug())
	if err != nil {
		return err
	}
	files, err := secrets.ReadFiles(m, clone.Dir)
	if err != nil {
		return err
	}
	files, changed, err := secrets.SetHostKey(files, hostname, key)
	if err != nil {
		return err
	}
	if err := secrets.WriteFiles(m, clone.Dir, files, changed); err != nil {
		return err
	}
	if err := reencrypt(ctx, clone, files, changed); err != nil {
		return err
	}
	// GitHub changes no deploy key in place, so an older key goes before the new one comes.
	drop, add := secrets.PlanDeployKeys(keys, hostname, key)
	for _, k := range drop {
		if err := secrets.DeleteDeployKey(ctx, m, secrets.GH, repo.Slug(), k.ID); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(out, "deleted the deploy key %s of %s\n", k.Title, repo.Slug())
	}
	if add {
		if err := secrets.AddDeployKey(ctx, m, secrets.GH, repo.Slug(), hostname, key); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(out, "added the read-only deploy key %s to %s\n", hostname, repo.Slug())
	}
	if len(changed) == 0 {
		_, _ = fmt.Fprintf(out, "the secrets already hold the host key of %s\n", hostname)
		return nil
	}
	_, err = clone.Land(ctx, out, secrets.Change{Topic: "add-" + hostname, Title: "chore: add the host key of " + hostname,
		Body: "Added by ws client macos secrets add " + name + ".", Files: changed})
	return err
}

// reencrypt encrypts the values of each changed fnox file again, to the recipients its table holds
// now: fnox.toml's by the shared provider, a profile's under its profile. A file of no value has
// nothing to encrypt, so fnox runs on none. fnox asks for the Nano's PIN and touch on the terminal.
func reencrypt(ctx context.Context, c *secrets.Clone, f secrets.Files, changed []string) error {
	for _, name := range changed {
		held, err := secrets.HasSecrets(f[name])
		if err != nil {
			return fmt.Errorf("cannot read %s: %w", name, err)
		}
		if !held {
			continue
		}
		args := []string{"reencrypt", "-p", secrets.SharedProvider, "-f"}
		if name != secrets.SharedFile {
			profile := strings.TrimSuffix(strings.TrimPrefix(name, "fnox."), ".toml")
			args = []string{"-P", profile, "reencrypt", "-p", secrets.ProfileProvider, "-f"}
		}
		if err := c.FnoxPass(ctx, args...); err != nil {
			return fmt.Errorf("cannot encrypt the values of %s again: %w", name, err)
		}
	}
	return nil
}
