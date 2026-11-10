package main

import (
	"errors"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/internal/secrets"
)

// newMacSecretsCmd returns ws mac secrets, which edits ws's secrets file. Each command runs its own
// of ws server secrets on ws as root, over the wizard's ssh, with a value on ssh's stdin.
func newMacSecretsCmd(p process) *cobra.Command {
	c := group("secrets", "edit ws's secrets file, over ssh", append([]*cobra.Command{{Use: "init",
		Short: "make ws's age identity and an empty secrets file on ws", Args: cobra.NoArgs}},
		secretsEditCmds(p, nil)...)...)
	forward(c, p)
	return c
}

// forward has each command under c, but a group, run its own on ws.
func forward(c *cobra.Command, p process) {
	for _, sub := range c.Commands() {
		if sub.HasSubCommands() {
			forward(sub, p)
			continue
		}
		sub.RunE = func(c *cobra.Command, args []string) error { return runFar(c, p, args) }
	}
}

// runFar runs ws server secrets with the command's path past ws mac secrets, its flags and its
// arguments. The arguments follow a --, as a name may start with a dash. So set env A runs
// ws server secrets set env -- A. set file sends the file at PATH, or stdin, and NAME alone. At a
// terminal, it reaches root on ws before it asks a value, so that a value is asked only where it can
// go. Else it reads stdin first, and sudo's password, where ws's sudo asks one, comes from /dev/tty.
func runFar(c *cobra.Command, p process, args []string) error {
	if why := wizardRefusal(p); why != "" {
		return errors.New(why)
	}
	far := strings.Fields(c.CommandPath())[3:]
	var talk rootTalk = effect.NewTerminal(c.InOrStdin(), c.ErrOrStderr())
	var in []byte
	var err error
	atTerminal := p.tty
	if !atTerminal {
		if in, args, err = farInput(c, p, far, args); err != nil {
			return err
		}
		if p.openTTY != nil {
			if tty, done, ttyErr := p.openTTY(); ttyErr == nil {
				defer done()
				talk, p.tty = struct {
					effect.Hidden
					effect.Out
				}{tty, talk}, true
			}
		}
	}
	r := &varsReach{p: p}
	defer r.close(c.Context())
	ws, err := r.reachOn(c.Context(), p.machine, talk)
	if err != nil {
		return err
	}
	if atTerminal {
		if in, args, err = farInput(c, p, far, args); err != nil {
			return err
		}
	}
	c.LocalNonPersistentFlags().VisitAll(func(f *pflag.Flag) {
		if f.Changed {
			far = append(far, "--"+f.Name+"="+f.Value.String())
		}
	})
	if len(args) > 0 {
		far = append(append(far, "--"), args...)
	}
	out, err := ws.Secrets(c.Context(), in, far...)
	_, _ = c.OutOrStdout().Write(out)
	return err
}

// farInput reads what set sends on ssh's stdin, and returns the arguments that go on to ws.
func farInput(c *cobra.Command, p process, far, args []string) ([]byte, []string, error) {
	switch {
	case far[0] == "set" && far[1] == secrets.KindFile:
		in, err := readProfile(c, p, args)
		return in, args[:1], err
	case far[0] == "set":
		value, err := readValue(c.InOrStdin(), c.ErrOrStderr(), far[1]+" "+args[0])
		return []byte(value + "\n"), args, err
	}
	return nil, args, nil
}
