package cli

import (
	"errors"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect"
	ubuntucli "github.com/zadykian/ws/roles/server/ubuntu/cli"
	"github.com/zadykian/ws/roles/server/ubuntu/secrets"
)

// newMacSecretsCmd returns ws client macos secrets, which edits ws's secrets file. Each command
// runs its own of ws server ubuntu secrets on ws as root, over the wizard's ssh, with a value on
// ssh's stdin.
func newMacSecretsCmd(p enginecli.Env) *cobra.Command {
	c := enginecli.Group("secrets", "edit ws's secrets file, over ssh", append([]*cobra.Command{{Use: "init",
		Short: "make ws's age identity and an empty secrets file on ws", Args: cobra.NoArgs}},
		ubuntucli.SecretsEdit(p, nil)...)...)
	forward(c, p)
	return c
}

// forward has each command under c, but a group, run its own on ws.
func forward(c *cobra.Command, p enginecli.Env) {
	for _, sub := range c.Commands() {
		if sub.HasSubCommands() {
			forward(sub, p)
			continue
		}
		sub.RunE = func(c *cobra.Command, args []string) error { return runFar(c, p, args) }
	}
}

// runFar runs ws server ubuntu secrets with the command's path past ws client macos secrets, its
// flags and its arguments. The arguments follow a --, as a name may start with a dash. So set env A
// runs ws server ubuntu secrets set env -- A. set file sends the file at PATH, or stdin, and NAME
// alone. At a terminal, it reaches root on ws before it asks a value, so that a value is asked only
// where it can go. Else it reads stdin first, and sudo's password, where ws's sudo asks one, comes
// from /dev/tty.
func runFar(c *cobra.Command, p enginecli.Env, args []string) error {
	if why := wizardRefusal(p); why != "" {
		return errors.New(why)
	}
	far := strings.Fields(c.CommandPath())[4:]
	var talk rootTalk = effect.NewTerminal(c.InOrStdin(), c.ErrOrStderr())
	var in []byte
	var err error
	atTerminal := p.TTY
	if !atTerminal {
		if in, args, err = farInput(c, p, far, args); err != nil {
			return err
		}
		if p.OpenTTY != nil {
			if tty, done, ttyErr := p.OpenTTY(); ttyErr == nil {
				defer done()
				talk, p.TTY = struct {
					effect.Hidden
					effect.Out
				}{tty, talk}, true
			}
		}
	}
	r := &varsReach{p: p}
	defer r.close(c.Context())
	ws, err := r.reachOn(c.Context(), p.Machine, talk)
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
func farInput(c *cobra.Command, p enginecli.Env, far, args []string) ([]byte, []string, error) {
	switch {
	case far[0] == "set" && far[1] == secrets.KindFile:
		in, err := ubuntucli.ReadProfile(c, p, args)
		return in, args[:1], err
	case far[0] == "set":
		value, err := ubuntucli.ReadValue(c.InOrStdin(), c.ErrOrStderr(), far[1]+" "+args[0])
		return []byte(value + "\n"), args, err
	}
	return nil, args, nil
}
