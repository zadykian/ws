package cli

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"path"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine"
	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/ubuntu/values"
)

// valuesChecked is the line of a pass, which the frame's template keeps as the file's text.
// The line never varies, so that the next plan finds the file unchanged.
const valuesChecked = "ws values: checked"

// valuesSide is what ws ubuntu SIDE values needs of a side: the values' file, how the file loads,
// and how a key of the file is set.
type valuesSide struct {
	file string
	load func(m effect.Machine) error
	set  func(data []byte, key, value string) ([]byte, error)
}

// guestValues is the guest's values, with its keys.
var guestValues = valuesSide{
	file: guestVars,
	load: func(m effect.Machine) error { _, err := values.Load(m, guestVars); return err },
	set:  values.Set,
}

// hostValues is the host's values, with its own keys.
var hostValues = valuesSide{
	file: hostVars,
	load: func(m effect.Machine) error { _, err := values.LoadHost(m, hostVars); return err },
	set:  values.SetHost,
}

// newValuesCmd returns ws ubuntu SIDE values, which checks and edits the machine's values.
func newValuesCmd(p enginecli.Env, side valuesSide) *cobra.Command {
	return enginecli.Group("values", "check and edit the machine's values", newValuesCheckCmd(p, side), &cobra.Command{
		Use:   "set KEY VALUE",
		Short: "write one key of " + side.file,
		Long: "An empty VALUE takes the key's line out, which gives its default back. A list's VALUE is split by " +
			"commas. A table stays the file's to edit by hand.\n\n" +
			"The next mise bootstrap takes the key. WS_SELF_CONTAINED waits for the next ws ubuntu guest setup, " +
			"as the units read the mode from the miserc that setup alone writes.",
		Args: cobra.ExactArgs(2),
		RunE: func(c *cobra.Command, args []string) error { return runValuesSet(c, p, side, args[0], args[1]) },
	})
}

// newValuesCheckCmd returns ws ubuntu SIDE values check.
func newValuesCheckCmd(p enginecli.Env, side valuesSide) *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Short: "refuse a bad value of " + side.file,
		Long: "It loads " + side.file + " as each command does, and prints " + valuesChecked + ". A missing file " +
			"passes, with the defaults. A bad value fails, naming its key and its rule. The frame's " +
			"/etc/ws/values.checked runs it as mise renders the config, so that a bad value stops a plan " +
			"or a run before any write.",
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			if err := checkValues(p.Machine, side); err != nil {
				return err
			}
			_, err := fmt.Fprintln(c.OutOrStdout(), valuesChecked)
			return err
		},
	}
}

// checkValues loads the file as each command does. A bad value fails with its key and its rule, and
// the file to edit by hand: values set refuses while any value is bad.
func checkValues(m effect.Machine, side valuesSide) error {
	if err := side.load(m); err != nil {
		return fmt.Errorf("%w: edit %s by hand", err, side.file)
	}
	return nil
}

// runValuesSet writes key in the file, by setServerValue.
func runValuesSet(c *cobra.Command, p enginecli.Env, side valuesSide, key, value string) error {
	if p.Euid != 0 {
		return errors.New("run it as root")
	}
	term := effect.NewTerminal(c.ErrOrStderr())
	defer term.Flush()
	return setServerValue(p.Machine, linesOf(term, c.CommandPath()), side, key, value)
}

// setServerValue writes key in the file, by rename and with its owner and mode, and says so on out.
// It writes nothing where the file holds the value already, and a link stays the maintainer's to
// edit, as a rename would replace it.
func setServerValue(m effect.Machine, out effect.Out, side valuesSide, key, value string) error {
	info, err := m.Stat(side.file)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if info.Link != "" {
		return fmt.Errorf("%s is a link to %s: name %s in its file by hand", side.file, info.Link, key)
	}
	data, err := m.ReadFile(side.file)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	next, err := side.set(data, key, value)
	if err != nil {
		return fmt.Errorf("%s: %w", side.file, err)
	}
	if bytes.Equal(next, data) {
		return nil
	}
	if err := writeVars(m, side.file, next); err != nil {
		return err
	}
	out.Say("wrote %s", side.file)
	return nil
}

// writeVars replaces the file. A first one is root's alone, in a directory that anyone reads.
func writeVars(m effect.Machine, file string, data []byte) error {
	if err := makeConfigDir(m, file); err != nil {
		return err
	}
	attrs, err := m.RootAttrs(file)
	if errors.Is(err, fs.ErrNotExist) {
		attrs, err = effect.Attrs{Mode: 0o600}, nil
	}
	if err != nil {
		return err
	}
	return m.WriteAtomic(file, data, attrs)
}

// makeConfigDir makes a missing /etc/ws, a directory that anyone reads: its files are
// root's alone.
func makeConfigDir(m effect.Machine, file string) error {
	dir := path.Dir(file)
	_, err := m.Stat(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return m.Mkdir(dir, 0o755)
	}
	return err
}

// linesOf returns the lines that the command at path prints on out.
func linesOf(out effect.Out, path string) engine.Lines {
	return engine.Lines{Out: out, Prog: path, Seen: map[string]bool{}}
}
