package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os/signal"
	"path"
	"slices"
	"syscall"

	"github.com/spf13/cobra"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/internal/etcws"
	"github.com/zadykian/ws/macos/values"
)

// newRemoveCmd returns ws macos remove.
func newRemoveCmd(p enginecli.Env, server *string) *cobra.Command {
	var skip bool
	c := &cobra.Command{
		Use:   "remove NAME",
		Short: "remove a server from the Mac",
		Long: "It first takes NAME's host key, from known_hosts, out of the secrets repository by git and gh: from " +
			"the recipients of fnox.toml and of every profile file, whose values it encrypts again with the Nano, by " +
			"a pull request, and from the repository's deploy keys. Where that fails, nothing else changes. Where " +
			"known_hosts holds no key of NAME, where this Mac has no fnox key, or with --skip-secrets, as where the " +
			"secrets are out of reach, it leaves them and prints what to do there by hand. It then deletes " +
			"servers/NAME beside values.yml and NAME's entry in known_hosts, and leaves the server as it is. " +
			"mise's bootstrap then takes its daemon, routes and host off. It refuses the Mac's last server, as " +
			"the bootstrap takes a Mac with none for a stray WS_VALUES and prunes no daemon, and takes no --server: " +
			"the argument is the consent.",
		Args: cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			// A Ctrl-C or a SIGTERM ends the context, not ws, so that remove still removes its clone.
			ctx, stop := signal.NotifyContext(c.Context(), syscall.SIGINT, syscall.SIGTERM)
			defer stop()
			c.SetContext(ctx)
			return runRemove(c, p, *server, args[0], skip)
		},
	}
	c.Flags().BoolVar(&skip, "skip-secrets", false, "leave NAME's host key in the secrets, and print it to drop by hand")
	return c
}

// runRemove drops the server called name from the Mac's files, and its host key from the secrets
// where skip is false.
func runRemove(c *cobra.Command, p enginecli.Env, server, name string, skip bool) error {
	if p.Euid == 0 {
		return errors.New("run it as yourself, not root: it asks sudo where it needs root")
	}
	if server != "" {
		return errors.New("remove takes the server's name alone")
	}
	mac := values.Path(p.Getenv, p.Pwd)
	if !values.IsName(name) {
		return fmt.Errorf("%s is not a server's name: %s", name, values.NameForm)
	}
	fleet := values.ReadFleet(p.Machine, mac)
	if fleet.NamesErr != nil {
		return fleet.NamesErr
	}
	names := fleet.Names()
	if !slices.Contains(names, name) {
		return fmt.Errorf("no server named %s in %s, which holds %s", name,
			path.Dir(path.Dir(values.ServerPath(mac, name))), values.List(names))
	}
	if len(names) == 1 {
		return fmt.Errorf("%s is the Mac's one server: with none, the bootstrap prunes no daemon, so remove leaves it", name)
	}
	if guests := fleet.GuestsOf(name); len(guests) > 0 {
		return fmt.Errorf("%s is the host of %s: remove them first, as a guest names its host in WS_HOST", name, values.List(guests))
	}
	// The units below skip a server whose file ws refuses, so such a file stops the removal before
	// it starts. NAME's own file is out of the count: a clash with it ends with the removal. A bad
	// values.yml is refused first and alone: no server is at fault.
	if fleet.MacErr != nil {
		return fleet.MacErr
	}
	for _, s := range fleet.Without(name).Servers {
		if s.Err != nil {
			return values.RefuseSibling(s.Name, s.Err)
		}
	}
	// A file that ws refuses goes all the same, as remove is the way out of it, and names no cable.
	var cable string
	if gone, _ := fleet.Server(name); gone.Err == nil {
		cable = gone.Conf.CableService
	}
	// The secrets go first, as they read the host's key, then the host: a failure of either leaves
	// the server whole, and remove runs again.
	byHand, err := dropSecrets(c, p, name, skip)
	if err != nil {
		return fmt.Errorf("%w: %s stays on the Mac: run remove again", err, name)
	}
	if err := forgetHost(c, p, name); err != nil {
		return err
	}
	dir := path.Dir(values.ServerPath(mac, name))
	rm := p.Machine.Run
	if etcws.Holds(dir) {
		rm = p.Machine.RunRoot
	}
	if _, err := rm(c.Context(), "rm", "-rf", dir); err != nil {
		return fmt.Errorf("cannot remove the files of %s: %w", name, err)
	}
	if _, err := fmt.Fprintf(c.OutOrStdout(), "removed %s\n", name); err != nil {
		return err
	}
	// The bootstrap takes the rest off. The unit of the tunnel's files boots the daemon out and
	// removes its files. The networks' unit takes its resolvers, launchd's its plist, ssh's its host.
	if err = p.Machine.Pass(c.Context(), false, "mise", "-C", "/opt/ws/macos", "bootstrap", "--yes"); err != nil {
		err = fmt.Errorf("mise -C /opt/ws/macos bootstrap: %w: run it again to take %s's daemon, routes and host off", err, name)
	}
	if byHand != "" {
		_, _ = fmt.Fprintln(c.OutOrStdout(), byHand)
	}
	if cable != "" {
		_, _ = fmt.Fprintf(c.OutOrStdout(), "By hand: the cable's service %s keeps 10.77.0.2: name it in another server's "+
			"WS_CABLE_SERVICE, or delete it in System Settings\n", cable)
	}
	return err
}

// forgetHost drops name's entry from known_hosts, where ssh keeps the key under the server's name.
// A file that is missing has none, and a link is left to the maintainer, who chose where it points.
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
		return fmt.Errorf("cannot drop %s from %s: %w", name, file, err)
	}
	return nil
}
