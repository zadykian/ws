package cli

import (
	"context"
	"fmt"
	"path"

	"github.com/spf13/cobra"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/roles/client/macos/cld"
	"github.com/zadykian/ws/roles/client/macos/internal/keys"
	"github.com/zadykian/ws/roles/client/macos/values"
)

// The Mac's parts that are no file: each a command of ws that a run state's unit runs, as the
// user, with --check as its check. Each reads as the user, and a write of root's goes through sudo.

// newCldCmd returns ws client macos cld, which cld's unit runs.
func newCldCmd(p enginecli.Env) *cobra.Command {
	return enginecli.Procedure("cld", "install cld into ~/.local/bin, and keep it at its latest release",
		"It installs cld with the install.sh of its latest release where it is missing, and runs cld "+
			"update where a newer release is out. A cld built from a clone is left as it is. A failed "+
			"install or update, or a latest release that it cannot read, is a warning.",
		func(_ context.Context, out effect.Out) ([]resource.Resource, error) {
			home, err := homeOf(p)
			if err != nil {
				return nil, err
			}
			return []resource.Resource{(&cld.Cld{Home: home, M: p.Machine, Out: out}).Part()}, nil
		})
}

// newKeysCmd returns ws client macos keys, which the keys' unit runs.
func newKeysCmd(p enginecli.Env) *cobra.Command {
	return enginecli.Procedure("keys", "copy the .pub of each key of Secretive's that values.yml names into /etc/ws",
		"ssh and git name a key that Secretive's agent holds by its .pub, which macOS keeps ws from "+
			"reading in Secretive's container. So ws keeps a copy, from the agent, in /etc/ws/secretive. "+
			"A key in a file has none.",
		func(ctx context.Context, out effect.Out) ([]resource.Resource, error) {
			home, err := homeOf(p)
			if err != nil {
				return nil, err
			}
			c, err := values.LoadMac(p.Machine, values.Path(p.Getenv, p.Pwd))
			if err != nil {
				return nil, err
			}
			return keys.Copies(ctx, p.Machine, out, c, home)
		})
}

// homeOf returns the user's home from HOME.
func homeOf(p enginecli.Env) (string, error) {
	home := p.Getenv("HOME")
	if !path.IsAbs(home) {
		return "", fmt.Errorf("HOME %q names no home: run it from the user's own shell", home)
	}
	return home, nil
}
