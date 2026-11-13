package cli

import (
	"errors"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine"
	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource/file"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos"
	"github.com/zadykian/ws/roles/client/macos/conf"
)

// macChange says the Mac's changes under --check. ws client macos apply printed a diff as diff -u
// does, with no prefix and no blank line after it, and said nothing of the backup of a file.
func macChange(c step.Change) ([]string, bool) {
	switch c := c.(type) {
	case file.UserWrite:
		return []string{strings.TrimSuffix(c.String(), "\n")}, true
	case file.UserBackup:
		return nil, true
	}
	return nil, false
}

// linesOf returns the lines that the command at path prints on out.
func linesOf(out effect.Out, path string) engine.Lines {
	return engine.Lines{Out: out, Prog: path, Seen: map[string]bool{}, Change: macChange}
}

// macOf returns the Mac's steps over the process's environment and the effects given to it.
// No other function builds a Mac. text takes what a run prints on stdout.
func macOf(p enginecli.Env, c conf.Conf, m effect.Machine, out effect.Out, text io.Writer) macos.Mac {
	return macos.Mac{Conf: c, Home: p.Getenv("HOME"), XDG: p.Getenv("XDG_CONFIG_HOME"), M: m, Out: out, Text: text}
}

// newMacApplyCmd returns ws client macos apply.
func newMacApplyCmd(p enginecli.Env) *cobra.Command {
	var only []string
	names := macos.Names()
	c := &cobra.Command{
		Use:   "apply",
		Short: "set the Mac up from mac.conf",
		Long: "It runs the steps it names, from mac.conf, in this order: " + strings.Join(names, ", ") + ". " +
			"With no --only, it runs every step, and ssh again, whose plan is empty unless cld " +
			"changed ssh's file. It asks sudo where it needs root.",
		Args: cobra.NoArgs,
		RunE: enginecli.Apply(macTarget(p, &only), p),
	}
	f := c.Flags()
	f.Bool("check", false, "print each change and make none")
	f.StringSliceVar(&only, "only", nil, "run the named steps alone, as a list or by repeating the flag")
	_ = c.RegisterFlagCompletionFunc("only", cobra.FixedCompletions(names, cobra.ShellCompDirectiveNoFileComp))
	return c
}

// loadMacConf refuses root and reads mac.conf and the server's file: the commands that act on the Mac's
// own files share it.
func loadMacConf(p enginecli.Env) (settings conf.Conf, err error) {
	if p.Euid == 0 {
		// As root, with HOME kept, it would leave root's files in the user's ~/.ssh, and run the
		// user's ~/.local/bin/cld.
		return conf.Conf{}, errors.New("run it as yourself, not root: it asks sudo where it needs root")
	}
	mac := conf.Path(p.Getenv, p.Pwd)
	return conf.Load(p.Machine, mac, conf.ServerPath(mac, conf.FirstServer))
}
