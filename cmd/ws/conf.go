package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/internal/conf"
)

const confFile = ".config/ws/mac.conf"

// newConfCmd returns ws mac conf.
func newConfCmd(p process) *cobra.Command {
	var lines bool
	c := &cobra.Command{
		Use:   "conf [FILE]",
		Short: "print mac.conf's checked settings as shell assignments",
		Long: "It reads FILE, else $WS_CONF, else ~/.config/ws/mac.conf, and prints the checked settings as " +
			"shell assignments, or the errors of a file it refuses.",
		Args: cobra.MaximumNArgs(1),
		RunE: func(c *cobra.Command, args []string) error { return runConf(c, p, args, lines) },
	}
	c.Flags().BoolVar(&lines, "lines", false, "check each line alone, as a wizard checks an answer before the file holds its key")
	return c
}

// runConf prints the settings of FILE, or of the file that the environment names.
func runConf(c *cobra.Command, p process, args []string, lines bool) error {
	parse := conf.Parse
	if lines {
		parse = conf.Lines
	}
	file := p.getenv("WS_CONF")
	if len(args) == 1 {
		file = args[0]
	}
	if file == "" {
		file = filepath.Join(p.getenv("HOME"), confFile)
	}
	text, err := p.machine.ReadFile(file)
	if errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("%s is missing: docs/mac.md says what goes in it", file)
	}
	if err != nil {
		return err
	}
	settings, errs := parse(string(text))
	if len(errs) > 0 {
		var all []error
		for _, e := range errs {
			all = append(all, errors.New(e.In(file)))
		}
		return errors.Join(all...)
	}
	_, err = io.WriteString(c.OutOrStdout(), settings.Assignments())
	return err
}
