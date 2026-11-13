package cmd

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine"
	"github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/engine/tui"
	"github.com/zadykian/ws/roles/client/macos"
	"github.com/zadykian/ws/roles/client/macos/conf"
)

// macChange says the Mac's changes under --check. ws client macos apply printed a diff as diff -u
// does, with no prefix and no blank line after it, and said nothing of the backup of a file.
func macChange(c step.Change) (string, bool) {
	switch c := c.(type) {
	case macos.Write:
		return strings.TrimSuffix(c.String(), "\n"), true
	case macos.Backup:
		return "", true
	}
	return "", false
}

// linesOf returns the lines that the command at path prints on out.
func linesOf(out effect.Out, path string) engine.Lines {
	return engine.Lines{Out: out, Prog: path, Seen: map[string]bool{}, Change: macChange}
}

// macOf returns the Mac's steps over the process's environment and the effects given to it.
// No other function builds a Mac. text takes what a run prints on stdout.
func macOf(p cli.Env, c conf.Conf, m effect.Machine, out effect.Out, text io.Writer) macos.Mac {
	return macos.Mac{Conf: c, Home: p.Getenv("HOME"), XDG: p.Getenv("XDG_CONFIG_HOME"), M: m, Out: out, Text: text}
}

// newMacApplyCmd returns ws client macos apply.
func newMacApplyCmd(p cli.Env) *cobra.Command {
	var check bool
	var only []string
	names := macos.Names()
	c := &cobra.Command{
		Use:   "apply",
		Short: "set the Mac up from mac.conf",
		Long: "It runs the steps it names, from mac.conf, in this order: " + strings.Join(names, ", ") + ". " +
			"With no --only, it runs every step, and ssh again, whose plan is empty unless cld " +
			"changed ssh's file. It asks sudo where it needs root.",
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error { return runMacApply(c, p, only, check) },
	}
	f := c.Flags()
	f.BoolVar(&check, "check", false, "print each change and make none")
	f.StringSliceVar(&only, "only", nil, "run the named steps alone, as a list or by repeating the flag")
	_ = c.RegisterFlagCompletionFunc("only", cobra.FixedCompletions(names, cobra.ShellCompDirectiveNoFileComp))
	return c
}

// loadMacConf refuses root and reads mac.conf: the commands that act on the Mac's own files share it.
func loadMacConf(p cli.Env) (settings conf.Conf, err error) {
	if p.Euid == 0 {
		// As root, with HOME kept, it would leave root's files in the user's ~/.ssh, and run the
		// user's ~/.local/bin/cld.
		return conf.Conf{}, errors.New("run it as yourself, not root: it asks sudo where it needs root")
	}
	file := p.Getenv("WS_CONF")
	if file == "" {
		file = filepath.Join(p.Getenv("HOME"), confFile)
	}
	text, err := p.Machine.ReadFile(file)
	if errors.Is(err, fs.ErrNotExist) {
		return conf.Conf{}, fmt.Errorf("%s is missing: docs/mac.md says what goes in it", file)
	}
	if err != nil {
		return conf.Conf{}, err
	}
	var errs []conf.Error
	settings, errs = conf.Parse(string(text))
	if len(errs) > 0 {
		var all []error
		for _, e := range errs {
			all = append(all, errors.New(e.In(file)))
		}
		return conf.Conf{}, errors.Join(all...)
	}
	return settings, nil
}

// setupRun runs ws client macos apply's steps on the effects given to it. It names no step in its
// errors.
type setupRun func(ctx context.Context, m effect.Machine, out effect.Out, text io.Writer, o step.Options) error

// runMacApply runs the steps that only names, or every step, from mac.conf. Under check it prints
// each change and makes none.
func runMacApply(c *cobra.Command, p cli.Env, only []string, check bool) error {
	if c.Flags().Changed("only") && len(only) == 0 {
		return engine.UsageError{Err: errors.New("--only names no step")}
	}
	for _, name := range only {
		if !slices.Contains(macos.Names(), name) {
			return engine.UsageError{Err: fmt.Errorf("no step is named %q", name)}
		}
	}
	settings, err := loadMacConf(p)
	if err != nil {
		return err
	}
	var run setupRun = func(ctx context.Context, m effect.Machine, out effect.Out, text io.Writer, o step.Options) error {
		_, err := macOf(p, settings, m, out, text).Run(ctx, macos.Options{Options: o, Only: only})
		return engine.Cause(err)
	}
	rowsOf := func(check bool) tui.Group {
		o := macos.Options{Options: step.Options{Check: check}, Only: only}
		return tui.NewGroup("", macOf(p, settings, nil, nil, nil).Rows(o))
	}
	if opensList(c, p, len(only) > 0) {
		chosen, ok, err := pickSetup(c, p, rowsOf(true), run)
		if err != nil || !ok {
			return err
		}
		return setupOnScreen(c, p, rowsOf(chosen.Check), run, chosen)
	}
	if engine.DrawsScreen(cli.CallOf(c, p)) {
		return setupOnScreen(c, p, rowsOf(check), run, step.Options{Check: check})
	}
	term := effect.NewTerminal(strings.NewReader(""), c.ErrOrStderr())
	out := linesOf(term, c.CommandPath())
	err = run(c.Context(), p.Machine, out, c.OutOrStdout(), step.Options{Observer: out, Check: check})
	term.Flush()
	return err
}

// setupOnScreen is a run of ws client macos apply under the run screen. The rows name the screen's
// steps, and the run acts on the screen's effects. What a run prints on stdout waits for the
// screen's end, as it would paint over the screen at a terminal.
func setupOnScreen(c *cobra.Command, p cli.Env, rows tui.Group, run setupRun, o step.Options) error {
	call := cli.CallOf(c, p)
	return engine.DrawRun(c.Context(), call, engine.RunTitle(call, o.Check), []tui.Group{rows}, func(ctx context.Context, b *tui.Bridge) error {
		e := b.Effects()
		return run(ctx, e.M, b, e.Text, b.Options(o))
	})
}
