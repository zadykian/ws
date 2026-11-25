package cli

import (
	"bufio"
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/ubuntu/roles/incus"
	"github.com/zadykian/ws/ubuntu/roles/tunnel"
	"github.com/zadykian/ws/ubuntu/values"
)

// newGuestsCmd returns ws ubuntu host guests: what the maintainer does with a guest by hand, past
// what the incus role's units make from the list.
func newGuestsCmd(p enginecli.Env) *cobra.Command {
	return enginecli.Group("guests", "list, set up and remove the guests of WS_INCUS_GUESTS",
		guestsList(p), guestsSetup(p), guestsRemove(p))
}

// declared returns the guest of the host's list called name.
func declared(p enginecli.Env, name string) (values.Guest, error) {
	v, err := values.LoadHost(p.Machine, hostVars)
	if err != nil {
		return values.Guest{}, err
	}
	for _, g := range v.Incus.Guests {
		if g.Name == name {
			return g, nil
		}
	}
	return values.Guest{}, fmt.Errorf("WS_INCUS_GUESTS in %s names no guest %s", hostVars, name)
}

// guestsList returns ws ubuntu host guests list.
func guestsList(p enginecli.Env) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "list each guest of the host's list, and each guest of ws's that the list lacks",
		Long: "Each line holds the guest's name, its address, and its state in Incus: Running, Stopped, or " +
			"missing where Incus holds no instance of that name. A guest of ws's that the list lacks follows, " +
			"marked as not in the list.",
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			v, err := values.LoadHost(p.Machine, hostVars)
			if err != nil {
				return err
			}
			held, err := incus.Instances(c.Context(), p.Machine)
			if err != nil {
				return err
			}
			state := func(name string) string {
				if i := slices.IndexFunc(held, func(i incus.Instance) bool { return i.Name == name }); i >= 0 {
					return held[i].Status
				}
				return "missing"
			}
			for _, g := range v.Incus.Guests {
				_, _ = fmt.Fprintf(c.OutOrStdout(), "%s\t%s\t%s\n", g.Name, g.Address, state(g.Name))
			}
			for _, name := range incus.Strays(held, v.Incus.Guests) {
				_, _ = fmt.Fprintf(c.OutOrStdout(), "%s\t-\t%s\tnot in WS_INCUS_GUESTS\n", name, state(name))
			}
			return nil
		},
	}
}

// guestsSetup returns ws ubuntu host guests setup.
func guestsSetup(p enginecli.Env) *cobra.Command {
	secrets := new(bool)
	c := &cobra.Command{
		Use:   "setup NAME",
		Short: "push ws and the guest's values to a guest, and run its setup",
		Long: "It pushes the host's ws and the guest's own values, /etc/ws/guests/NAME/values.yml where it exists, " +
			"then runs ws ubuntu guest setup in the guest, whether its own plan finds a change or not. " +
			"--secrets adds the task that clones the secrets on its first run: it needs the guest's key among " +
			"the secrets' recipients, which the runbook says how to add. The guest must run.",
		Args: cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			g, err := declared(p, args[0])
			if err != nil {
				return err
			}
			term := effect.NewTerminal(c.ErrOrStderr())
			defer term.Flush()
			out := linesOf(term, c.CommandPath())
			return settle(c.Context(), out, incus.SetUp(p.Machine, g, *secrets, true, out), false)
		},
	}
	secrets = c.Flags().Bool("secrets", false, "run the task secrets in the guest, which clones the secrets on its first run")
	return c
}

// guestsRemove returns ws ubuntu host guests remove.
func guestsRemove(p enginecli.Env) *cobra.Command {
	return &cobra.Command{
		Use:   "remove NAME",
		Short: "delete a guest with its disk and snapshots, after asking",
		Long: "It deletes the instance of a guest of ws's, which takes the profile ws, after it asks. The disk and " +
			"the snapshots go with it, and so does the guest's tunnel on the host: the interface wgN of the slot " +
			"that its values name, its key and its peers' keys, so that the next guest of the slot gets a key of " +
			"its own and none of the old peers. It refuses a guest that WS_INCUS_GUESTS still names, as the next " +
			"run of the guests unit would launch it again, so take it out of the list first. It leaves the guest's " +
			"values in /etc/ws/guests, and prints the command that removes the guest on the Mac.",
		Args: cobra.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			name := args[0]
			v, err := values.LoadHost(p.Machine, hostVars)
			if err != nil {
				return err
			}
			if slices.ContainsFunc(v.Incus.Guests, func(g values.Guest) bool { return g.Name == name }) {
				return fmt.Errorf("WS_INCUS_GUESTS in %s still names %s: take it out first, or the guests unit launches it again", hostVars, name)
			}
			held, err := incus.Instances(c.Context(), p.Machine)
			if err != nil {
				return err
			}
			if !incus.IsGuest(held, name) {
				return fmt.Errorf("no guest of ws's named %s in Incus", name)
			}
			t, noTunnel := removedTunnel(p.Machine, name)
			holder := slices.IndexFunc(v.Incus.Guests, func(g values.Guest) bool { return noTunnel == nil && g.Link() == t.Link() })
			what := "its disk and its snapshots"
			switch {
			case holder >= 0:
				what = fmt.Sprintf("its disk, its snapshots and its tunnel %s on the host, which %s holds now", t.Link(), v.Incus.Guests[holder].Name)
			case noTunnel == nil:
				what = "its disk, its snapshots and its tunnel " + t.Link() + " on the host"
			}
			_, _ = fmt.Fprintf(c.ErrOrStderr(), "Delete the guest %s, with %s? (yes/no) [no]: ", name, what)
			answer, err := bufio.NewReader(c.InOrStdin()).ReadString('\n')
			if err != nil && answer == "" {
				return errors.New("no answer, so nothing changed")
			}
			if strings.TrimSpace(answer) != "yes" {
				return fmt.Errorf("not confirmed, so %s stays", name)
			}
			if err := incus.Remove(c.Context(), p.Machine, name); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(c.OutOrStdout(), "deleted the guest %s\n", name)
			if noTunnel != nil {
				_, _ = fmt.Fprintf(c.ErrOrStderr(), "warning: %v, so no tunnel is deleted. Where %s held a slot N, delete "+
					"/etc/systemd/network/60-wgN.*, /etc/wireguard/wgN and the interface wgN by hand, or the next guest of the slot takes its key and peers\n", noTunnel, name)
			} else {
				if err := tunnel.RemoveGuest(c.Context(), p.Machine, t); err != nil {
					return fmt.Errorf("the guest %s is deleted, but not its tunnel %s: %w", name, t.Link(), err)
				}
				_, _ = fmt.Fprintf(c.OutOrStdout(), "deleted its tunnel %s, with its key and its peers' keys\n", t.Link())
			}
			if holder >= 0 {
				_, _ = fmt.Fprintf(c.OutOrStdout(), "The next ws ubuntu host setup gives %s a new key for %s, and %s's peers' keys go in %s again\n",
					t.Link(), v.Incus.Guests[holder].Name, v.Incus.Guests[holder].Name, t.PeersDir())
			}
			_, _ = fmt.Fprintf(c.OutOrStdout(), "On the Mac, run: ws macos remove %s\n", name)
			return nil
		},
	}
}

// removedTunnel returns the tunnel that the host ended for the guest name, from the values that
// remove leaves in /etc/ws/guests. The error says why the values name none.
func removedTunnel(m effect.Machine, name string) (values.GuestTunnel, error) {
	g := values.Guest{Name: name}
	data, v, err := g.Read(m)
	switch {
	case err != nil:
		return values.GuestTunnel{}, err
	case data == nil:
		return values.GuestTunnel{}, fmt.Errorf("%s is missing", g.File())
	}
	return g.Tunnel(v)
}
