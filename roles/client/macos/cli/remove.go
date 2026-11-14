package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine"
	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/engine/core/wizard"
	"github.com/zadykian/ws/roles/client/macos"
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
	names, err := conf.Servers(p.Machine, mac)
	if err != nil {
		return err
	}
	if len(names) == 1 {
		return fmt.Errorf("%s is the Mac's one server: apply needs one, so remove leaves it", name)
	}
	_, release, err := wizard.Lock(c.Context(), p.Machine, p.LockDir())
	if err != nil {
		return err
	}
	defer release()
	// The steps below skip a server whose file ws refuses, so such a file stops the removal before
	// it starts. NAME's own file is out of the count: a clash with it ends with the removal.
	if err := refuseBadSibling(without{p.Machine, conf.ServerPath(mac, name)}, mac, name); err != nil {
		return err
	}
	// A file that ws refuses goes all the same, as remove is the way out of it, and names no cable.
	var cable string
	if gone, err := conf.Load(p.Machine, mac, conf.ServerPath(mac, name)); err == nil {
		cable = gone.CableService
	}
	if _, err := p.Machine.Run(c.Context(), "rm", "-rf", path.Dir(conf.ServerPath(mac, name))); err != nil {
		return fmt.Errorf("cannot remove the files of %s: %w", name, err)
	}
	if err := forgetHost(c, p, name); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(c.OutOrStdout(), "removed %s\n", name); err != nil {
		return err
	}
	rest := slices.DeleteFunc(names, func(n string) bool { return n == name })
	if err = reapply(c, p, mac, rest[0]); err != nil {
		err = fmt.Errorf("%w: then ws client macos apply --server %s takes %s's daemon, hosts and routes off",
			err, rest[0], name)
	}
	if cable != "" {
		_, _ = fmt.Fprintf(c.OutOrStdout(), "By hand: the cable's service %s keeps 10.77.0.2: name it in another server's "+
			"WS_CABLE_SERVICE, or delete it in System Settings\n", cable)
	}
	return err
}

// reapply runs the Mac's steps that hold a server's traces, for the server that stays. They are ssh, Zed,
// the tunnel and the networks, and the last two prune the gone server's daemon and routes.
func reapply(c *cobra.Command, p enginecli.Env, mac, name string) error {
	settings, err := conf.Load(p.Machine, mac, conf.ServerPath(mac, name))
	if err != nil {
		return fmt.Errorf("server %s: %w", name, err)
	}
	term := effect.NewTerminal(strings.NewReader(""), c.ErrOrStderr())
	defer term.Flush()
	out := linesOf(term, c.CommandPath())
	_, err = macOf(p, name, settings, p.Machine, out, c.OutOrStdout()).Run(c.Context(), macos.Options{
		Options: step.Options{Observer: out}, Only: []string{"ssh", "zed", "tunnel", "networks"}})
	return engine.Cause(err)
}

// without is the Mac's machine with the server's file at file out of sight. conf.Servers leaves
// that server out, so the others load as they will once it goes.
type without struct {
	effect.Machine
	file string
}

// Stat reports the hidden file as missing.
func (w without) Stat(file string) (effect.Info, error) {
	if file == w.file {
		return effect.Info{}, fs.ErrNotExist
	}
	return w.Machine.Stat(file)
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
