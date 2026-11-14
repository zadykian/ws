package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"path"

	"github.com/spf13/cobra"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/wizard"
	"github.com/zadykian/ws/roles/client/macos/conf"
)

// newRemoveCmd returns ws client macos remove.
func newRemoveCmd(p enginecli.Env, server *string) *cobra.Command {
	return &cobra.Command{
		Use:   "remove NAME",
		Short: "remove a server from the Mac",
		Long: "It deletes servers/NAME beside mac.conf and NAME's entry in known_hosts, and leaves the server as it " +
			"is. It refuses the Mac's last server, as apply needs one, and takes no --server: the argument is the consent.",
		Args: cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error { return runRemove(c, p, *server, args[0]) },
	}
}

// runRemove drops the server called name from the Mac's files, under the wizard's lock, so that no run
// writes its file meanwhile.
func runRemove(c *cobra.Command, p enginecli.Env, server, name string) error {
	if p.Euid == 0 {
		return errors.New("run it as yourself, not root: it asks sudo where it needs root")
	}
	if server != "" {
		return errors.New("remove takes the server's name alone")
	}
	mac := conf.Path(p.Getenv, p.Pwd)
	if _, err := conf.Pick(p.Machine, mac, name); err != nil {
		return err
	}
	if names, err := conf.Servers(p.Machine, mac); err != nil {
		return err
	} else if len(names) == 1 {
		return fmt.Errorf("%s is the Mac's one server: apply needs one, so remove leaves it", name)
	}
	_, release, err := wizard.Lock(c.Context(), p.Machine, p.LockDir())
	if err != nil {
		return err
	}
	defer release()
	if _, err := p.Machine.Run(c.Context(), "rm", "-rf", path.Dir(conf.ServerPath(mac, name))); err != nil {
		return fmt.Errorf("cannot remove the files of %s: %w", name, err)
	}
	if err := forgetHost(c, p, name); err != nil {
		return err
	}
	_, err = fmt.Fprintf(c.OutOrStdout(), "removed %s\n", name)
	return err
}

// forgetHost drops name's entry from known_hosts, where ssh keeps the key under the server's name.
// A file that is missing has none, and a link is left to the maintainer, as the wizard leaves it.
func forgetHost(c *cobra.Command, p enginecli.Env, name string) error {
	file := p.Getenv("HOME") + "/.ssh/known_hosts"
	info, err := p.Machine.Stat(file)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Link != "" {
		_, err = fmt.Fprintf(c.OutOrStdout(), "By hand: %s is a link to %s: delete the lines of %s there\n", file, info.Link, name)
		return err
	}
	if _, err := p.Machine.Run(c.Context(), "ssh-keygen", "-R", name); err != nil {
		return fmt.Errorf("cannot drop %s from %s", name, file)
	}
	return nil
}
