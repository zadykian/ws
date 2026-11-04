package main

import (
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

// Say prints the line after the prefix. It drops the runner's line that a step is applying, which
// the step's own lines follow, and the step's name that starts a line of --check.
func (o progOut) Say(format string, a ...any) {
	switch {
	case format == "%s: applying %d change(s)":
		return
	case format == "%s: %s" && len(a) == 2:
		o.sayChange(a[1]) //nolint:gosec // len(a) is 2 in this case
	default:
		o.Out.Say("%s: "+format, append([]any{o.prog}, a...)...)
	}
}

// sayChange prints a change under --check. ws mac apply printed a diff as diff -u does, with no prefix and
// no blank line after it, and said nothing of the backup of a file.
func (o progOut) sayChange(c any) {
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

// setupSteps returns the steps of ws mac apply, in the order ws mac apply runs them.
func setupSteps(c conf.Conf, home, xdg string, m effect.Machine, out effect.Out, text io.Writer, check bool) []step.Step {
	return []step.Step{
		sshStep(c, home, m, out, false),
		setup.Cable{Conf: c, M: m, Out: out},
		&setup.Git{Conf: c, Home: home, XDG: xdg, M: m, Out: out},
		&setup.Gateway{Home: home, M: m, Out: out},
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

// runMacApply runs the steps that args name, or all of them, from mac.conf. Under check it prints
// each change and makes none.
func runMacApply(c *cobra.Command, p process, args []string, check bool) error {
	if p.euid == 0 {
		// As root, with HOME kept, it would leave root's files in the user's ~/.ssh, and run the
		// user's ~/.local/bin/cld.
		return errors.New("run it as yourself, not root: it asks sudo where it needs root")
	}
	home := p.getenv("HOME")
	file := p.getenv("WS_CONF")
	if file == "" {
		file = filepath.Join(home, confFile)
	}
	text, err := p.machine.ReadFile(file)
	if errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("%s is missing: docs/mac.md says what goes in it", file)
	}
	if err != nil {
		return err
	}
	settings, errs := conf.Parse(string(text))
	if len(errs) > 0 {
		var all []error
		for _, e := range errs {
			all = append(all, errors.New(e.In(file)))
		}
		return errors.Join(all...)
	}
	term := effect.NewTerminal(strings.NewReader(""), c.ErrOrStderr())
	out := progOut{term, c.CommandPath(), map[string]bool{}}
	steps := setupSteps(settings, home, p.getenv("XDG_CONFIG_HOME"), p.machine, out, c.OutOrStdout(), check)
	chosen := steps
	if len(args) == 1 {
		chosen = slices.DeleteFunc(slices.Clone(steps), func(s step.Step) bool { return s.Name() != args[0] })
	} else if !check {
		// cld's restore tabs change ssh's file, so ssh runs once more.
		// Under --check nothing is applied, so a second plan would only repeat the first.
		chosen = slices.Concat(steps, []step.Step{sshStep(settings, home, p.machine, out, true)})
	}
	ctx := c.Context()
	_, err = step.Run(ctx, chosen, out, check)
	for _, s := range chosen {
		if f, ok := s.(interface{ Finish(context.Context) error }); ok && err == nil {
			err = f.Finish(ctx)
		}
	}
	term.Flush()
	return unwrapStep(err)
}

// unwrapStep returns the error of a step, without the step and stage that the runner names before
// it: the command names itself.
func unwrapStep(err error) error {
	if inner := errors.Unwrap(err); inner != nil {
		return inner
	}
	return err
}
