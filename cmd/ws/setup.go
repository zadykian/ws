package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/internal/conf"
	"github.com/zadykian/ws/internal/setup"
	"github.com/zadykian/ws/internal/tui"
	"github.com/zadykian/ws/mac"
	"github.com/zadykian/ws/step"
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
// applying, which the step's own lines follow: a plan under --check, what a step is behind, and a
// left-out step's failed plan.
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

func sshStep(c conf.Conf, home string, m effect.Machine, out effect.Out, again bool) step.Step {
	return setup.SSH{Conf: c, Home: home, M: m, Out: out, Now: time.Now, Again: again}
}

// setupBuild returns the steps of a run on the effects it is given. text takes what the run prints
// on stdout.
type setupBuild func(m effect.Machine, out effect.Out, text io.Writer) []step.Step

// setupSteps returns the steps of ws mac apply, in the order ws mac apply runs them.
func setupSteps(c conf.Conf, home, xdg string, m effect.Machine, out effect.Out, text io.Writer, check bool) []step.Step {
	return []step.Step{
		sshStep(c, home, m, out, false),
		setup.Cable{Conf: c, M: m, Out: out},
		&setup.Git{Conf: c, Home: home, XDG: xdg, M: m, Out: out},
		&setup.Ghostty{Home: home, Settings: mac.Ghostty, M: m, Out: out},
		&setup.Zed{Home: home, Settings: mac.Zed, Keymap: mac.ZedKeymap, M: m, Out: out},
		&setup.Cld{Home: home, M: m, Out: out, Check: check},
		&setup.Tunnel{Conf: c, Home: home, Plist: mac.TunnelPlist, M: m, Out: out, Text: text,
			Check: check},
		&setup.Networks{Conf: c, M: m, Out: out, Check: check},
	}
}

// newMacApplyCmd returns ws mac apply.
func newMacApplyCmd(p process) *cobra.Command {
	var check bool
	names := setupStepNames(p)
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

// setupStepNames returns the names of the steps of ws mac apply, in order.
func setupStepNames(p process) []string {
	var names []string
	for _, s := range setupSteps(conf.Conf{}, "", "", p.machine, nil, io.Discard, false) {
		names = append(names, s.Name())
	}
	return names
}

// loadMacConf refuses root and reads mac.conf: the commands that act on the Mac's own files share it.
func loadMacConf(p process) (home string, settings conf.Conf, err error) {
	if p.euid == 0 {
		// As root, with HOME kept, it would leave root's files in the user's ~/.ssh, and run the
		// user's ~/.local/bin/cld.
		return "", conf.Conf{}, errors.New("run it as yourself, not root: it asks sudo where it needs root")
	}
	home = p.getenv("HOME")
	file := p.getenv("WS_CONF")
	if file == "" {
		file = filepath.Join(home, confFile)
	}
	text, err := p.machine.ReadFile(file)
	if errors.Is(err, fs.ErrNotExist) {
		return "", conf.Conf{}, fmt.Errorf("%s is missing: docs/mac.md says what goes in it", file)
	}
	if err != nil {
		return "", conf.Conf{}, err
	}
	var errs []conf.Error
	settings, errs = conf.Parse(string(text))
	if len(errs) > 0 {
		var all []error
		for _, e := range errs {
			all = append(all, errors.New(e.In(file)))
		}
		return "", conf.Conf{}, errors.Join(all...)
	}
	return home, settings, nil
}

// runMacApply runs the steps that args name, or all of them, from mac.conf. Under check it prints
// each change and makes none.
func runMacApply(c *cobra.Command, p process, args []string, check bool) error {
	home, settings, err := loadMacConf(p)
	if err != nil {
		return err
	}
	xdg := p.getenv("XDG_CONFIG_HOME")
	buildFor := func(check bool) setupBuild {
		return func(m effect.Machine, out effect.Out, text io.Writer) []step.Step {
			steps := setupSteps(settings, home, xdg, m, out, text, check)
			switch {
			case len(args) == 1:
				return slices.DeleteFunc(steps, func(s step.Step) bool { return s.Name() != args[0] })
			case check:
				// Nothing is applied, so a second plan would only repeat the first.
				return steps
			}
			// cld's restore tabs change ssh's file, so ssh runs once more.
			return slices.Concat(steps, []step.Step{sshStep(settings, home, m, out, true)})
		}
	}
	build := buildFor(check)
	if opensList(c, p, len(args) == 1) {
		chosen, ok, err := pickSetup(c, p, buildFor)
		if err != nil || !ok {
			return err
		}
		return setupOnScreen(c, p, buildFor(chosen.Check), chosen)
	}
	if drawsScreen(c, p) {
		return setupOnScreen(c, p, build, step.Options{Check: check})
	}
	term := effect.NewTerminal(strings.NewReader(""), c.ErrOrStderr())
	out := progOut{term, c.CommandPath(), map[string]bool{}}
	err = runSetup(c.Context(), build(p.machine, out, c.OutOrStdout()), step.Options{Observer: out, Check: check})
	term.Flush()
	return err
}

// setupOnScreen is a run of ws mac apply under the run screen. The plain effects name the screen's
// rows, and the run builds its steps again on the screen's effects. What a run prints on stdout
// waits for the screen's end, as it would paint over the screen at a terminal.
func setupOnScreen(c *cobra.Command, p process, build setupBuild, o step.Options) error {
	rows := tui.NewGroup("", build(p.machine, nil, io.Discard))
	var text bytes.Buffer
	err := drawRun(c, p, runTitle(c, o.Check), []tui.Group{rows}, func(ctx context.Context, b *tui.Bridge) error {
		steps := build(screenMachine(ctx, b, p.machine), b, &text)
		o.Observer, o.Stop = b, b.Stop(ctx)
		return runSetup(ctx, steps, o)
	})
	if err == nil { // A work left behind may still write to text.
		_, _ = text.WriteTo(c.OutOrStdout())
	}
	return err
}

// runSetup is finishSetup for the command, which names itself in its errors.
func runSetup(ctx context.Context, steps []step.Step, o step.Options) error {
	_, err := finishSetup(ctx, steps, o)
	return unwrapStep(err)
}

// finishSetup runs steps under o, then what each leaves for the end of a run that passed. A step
// left out leaves nothing, as it applied nothing. It reports what step.RunWith does.
func finishSetup(ctx context.Context, steps []step.Step, o step.Options) (bool, error) {
	changed, err := step.RunWith(ctx, steps, o)
	for _, s := range steps {
		left := o.LeftOut != nil && o.LeftOut(o.Parent, s.Name())
		if f, ok := s.(interface{ Finish(context.Context) error }); ok && err == nil && !left {
			err = f.Finish(ctx)
		}
	}
	return changed, err
}

// unwrapStep returns the error of a step, without the step and stage that the runner names before
// it: the command names itself.
func unwrapStep(err error) error {
	if inner := errors.Unwrap(err); inner != nil {
		return inner
	}
	return err
}
