package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/internal/server/values"
)

// newValuesCmd returns ws server values, which edits the machine's values.
func newValuesCmd(p process) *cobra.Command {
	return group("values", "edit the machine's values", &cobra.Command{
		Use:   "set KEY VALUE",
		Short: "write one key of " + serverVars,
		Long: "An empty VALUE takes the key's line out, which gives its default back. A list's VALUE is split by " +
			"commas. A table, such as fans_curves, stays the file's to edit by hand.\n\n" +
			"The next ws server apply takes the key.",
		Args: cobra.ExactArgs(2),
		RunE: func(c *cobra.Command, args []string) error { return runValuesSet(c, p, args[0], args[1]) },
	})
}

// runValuesSet writes key in the file, by setServerValue.
func runValuesSet(c *cobra.Command, p process, key, value string) error {
	if p.euid != 0 {
		return errors.New("run it as root")
	}
	term := effect.NewTerminal(strings.NewReader(""), c.ErrOrStderr())
	defer term.Flush()
	return setServerValue(p.machine, progOut{term, c.CommandPath(), map[string]bool{}}, key, value)
}

// setServerValue writes key in the file, by rename and with its owner and mode, as the wizard's own
// writes do, and says so on out. It writes nothing where the file holds the value already, and a
// link stays the maintainer's to edit, as a rename would replace it. ws server wizard writes its
// values through it too.
func setServerValue(m effect.Machine, out effect.Out, key, value string) error {
	info, err := m.Stat(serverVars)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if info.Link != "" {
		return fmt.Errorf("%s is a link to %s: name %s in its file by hand", serverVars, info.Link, key)
	}
	data, err := m.ReadFile(serverVars)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	next, err := values.Set(data, key, value)
	if err != nil {
		return fmt.Errorf("%s: %w", serverVars, err)
	}
	if bytes.Equal(next, data) {
		return nil
	}
	if err := writeVars(m, next); err != nil {
		return err
	}
	out.Say("wrote %s", serverVars)
	return nil
}

// writeVars replaces the file. A first one is root's alone, in a directory that anyone reads.
func writeVars(m effect.Machine, data []byte) error {
	if err := makeConfigDir(m); err != nil {
		return err
	}
	attrs, err := m.RootAttrs(serverVars)
	if errors.Is(err, fs.ErrNotExist) {
		attrs, err = effect.Attrs{Mode: 0o600}, nil
	}
	if err != nil {
		return err
	}
	return m.WriteAtomic(serverVars, data, attrs)
}

// makeConfigDir makes /etc/ws where it is missing, a directory that anyone reads: its files are
// root's alone.
func makeConfigDir(m effect.Machine) error {
	dir := path.Dir(serverVars)
	_, err := m.Stat(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return m.Mkdir(dir, 0o755)
	}
	return err
}
