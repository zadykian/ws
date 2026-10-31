package main

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"

	"github.com/zadykian/ws/mac/internal/conf"
	"github.com/zadykian/ws/mac/internal/effect"
)

const confUsage = "usage: ws conf [FILE]\n"

const confFile = ".config/ws/mac.conf"

// confCommand runs ws conf: it reads FILE, else $WS_CONF, else ~/.config/ws/mac.conf, and prints
// the checked settings as shell assignments, or the errors of a file it refuses.
func confCommand(args []string, m effect.Machine, getenv func(string) string, stdout, stderr io.Writer) int {
	if len(args) > 1 {
		_, _ = io.WriteString(stderr, confUsage)
		return 2
	}
	file := getenv("WS_CONF")
	if len(args) == 1 {
		file = args[0]
	}
	if file == "" {
		file = filepath.Join(getenv("HOME"), confFile)
	}
	text, err := m.ReadFile(file)
	if errors.Is(err, fs.ErrNotExist) {
		_, _ = fmt.Fprintf(stderr, "ws conf: %s is missing: docs/mac.md says what goes in it\n", file)
		return 1
	}
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "ws conf: %v\n", err)
		return 1
	}
	c, errs := conf.Parse(string(text))
	for _, e := range errs {
		_, _ = fmt.Fprintf(stderr, "ws conf: %s\n", e.In(file))
	}
	if len(errs) > 0 {
		return 1
	}
	_, _ = io.WriteString(stdout, c.Assignments())
	return 0
}
