package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/engine/tui"
	"github.com/zadykian/ws/internal/conf"
	"github.com/zadykian/ws/roles/client/macos/setup"
)

// progOut prefixes a line with the path of the command that prints it, and says each warning once,
// as a step plans again after it applies.
type progOut struct {
	effect.Out
	prog string
	seen map[string]bool
}

// Say prints the line after the prefix.
func (o progOut) Say(format string, a ...any) {
	o.Out.Say("%s: "+format, append([]any{o.prog}, a...)...)
}

// Observe prints a run's events as the runner's lines once did, but for the line that a step is
// applying. The step's own lines follow that line: a plan under --check, what a step is behind,
// and a left-out step's failed plan.
func (o progOut) Observe(e step.Event) {
	switch {
	case e.Kind == step.Skipped && e.Err != nil:
		o.Warn("left out: %v", e.Err)
	case e.LeftOut:
	case e.Kind == step.Planned && e.Check:
		for _, c := range e.Changes {
			o.sayChange(c)
		}
	case e.Kind == step.Skipped && e.After != "":
		o.Say("%s: after %s", e.Step, e.After)
	}
}

// sayChange prints a change under --check. ws mac apply printed a diff as diff -u does, with no prefix and
// no blank line after it, and said nothing of the backup of a file.
func (o progOut) sayChange(c step.Change) {
	switch c := c.(type) {
	case setup.Write:
		o.Out.Say("%s", strings.TrimSuffix(c.String(), "\n"))
	case setup.Backup:
	default:
		o.Out.Say("%s: %v", o.prog, c)
	}
}

func (o progOut) Warn(format string, a ...any) {
	line := fmt.Sprintf(format, a...)
	if !o.seen[line] {
		o.seen[line] = true
		o.Out.Say("%s: warning: %s", o.prog, line)
	}
}

// macOf returns the Mac's steps over the process's environment and the effects given to it.
// No other function builds a Mac. text takes what a run prints on stdout.
func macOf(p process, c conf.Conf, m effect.Machine, out effect.Out, text io.Writer) setup.Mac {
	return setup.Mac{Conf: c, Home: p.getenv("HOME"), XDG: p.getenv("XDG_CONFIG_HOME"), M: m, Out: out, Text: text}
}

// newMacApplyCmd returns ws mac apply.
func newMacApplyCmd(p process) *cobra.Command {
	var check bool
	names := setup.Names()
	c := &cobra.Command{
		Use:   "apply [STEP]",
		Short: "set the Mac up from mac.conf",
		Long: "It runs the steps it names, from mac.conf, in this order: " + strings.Join(names, ", ") + ". " +
			"With no STEP, it runs every step, and ssh again, whose plan is empty unless cld " +
			"changed ssh's file. It asks sudo where it needs root.",
		Args:      cobra.MatchAll(cobra.MaximumNArgs(1), cobra.OnlyValidArgs),
		ValidArgs: names,
		RunE:      func(c *cobra.Command, args []string) error { return runMacApply(c, p, args, check) },
	}
	c.Flags().BoolVar(&check, "check", false, "print each change and make none")
	return c
}

// loadMacConf refuses root and reads mac.conf: the commands that act on the Mac's own files share it.
func loadMacConf(p process) (settings conf.Conf, err error) {
	if p.euid == 0 {
		// As root, with HOME kept, it would leave root's files in the user's ~/.ssh, and run the
		// user's ~/.local/bin/cld.
		return conf.Conf{}, errors.New("run it as yourself, not root: it asks sudo where it needs root")
	}
	file := p.getenv("WS_CONF")
	if file == "" {
		file = filepath.Join(p.getenv("HOME"), confFile)
	}
	text, err := p.machine.ReadFile(file)
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

// setupRun runs ws mac apply's steps on the effects given to it. It names no step in its errors.
type setupRun func(ctx context.Context, m effect.Machine, out effect.Out, text io.Writer, o step.Options) error

// runMacApply runs the steps that args name, or every step, from mac.conf. Under check it prints
// each change and makes none.
func runMacApply(c *cobra.Command, p process, args []string, check bool) error {
	settings, err := loadMacConf(p)
	if err != nil {
		return err
	}
	var only string
	if len(args) == 1 {
		only = args[0]
	}
	var run setupRun = func(ctx context.Context, m effect.Machine, out effect.Out, text io.Writer, o step.Options) error {
		_, err := macOf(p, settings, m, out, text).Run(ctx, setup.Options{Options: o, Only: only})
		return unwrapStep(err)
	}
	rowsOf := func(check bool) tui.Group {
		o := setup.Options{Options: step.Options{Check: check}, Only: only}
		return tui.NewGroup("", macOf(p, settings, nil, nil, nil).Rows(o))
	}
	if opensList(c, p, len(args) == 1) {
		chosen, ok, err := pickSetup(c, p, rowsOf(true), run)
		if err != nil || !ok {
			return err
		}
		return setupOnScreen(c, p, rowsOf(chosen.Check), run, chosen)
	}
	if drawsScreen(c, p) {
		return setupOnScreen(c, p, rowsOf(check), run, step.Options{Check: check})
	}
	term := effect.NewTerminal(strings.NewReader(""), c.ErrOrStderr())
	out := progOut{term, c.CommandPath(), map[string]bool{}}
	err = run(c.Context(), p.machine, out, c.OutOrStdout(), step.Options{Observer: out, Check: check})
	term.Flush()
	return err
}

// setupOnScreen is a run of ws mac apply under the run screen. The rows name the screen's steps,
// and the run acts on the screen's effects. What a run prints on stdout waits for the screen's end,
// as it would paint over the screen at a terminal.
func setupOnScreen(c *cobra.Command, p process, rows tui.Group, run setupRun, o step.Options) error {
	return drawRun(c, p, runTitle(c, o.Check), []tui.Group{rows}, func(ctx context.Context, b *tui.Bridge) error {
		e := b.Effects()
		return run(ctx, e.M, b, e.Text, b.Options(o))
	})
}

// unwrapStep returns the error of a step, without the step and stage that the runner names before
// it: the command names itself.
func unwrapStep(err error) error {
	if inner := errors.Unwrap(err); inner != nil {
		return inner
	}
	return err
}
