package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"strings"
	"time"

	"github.com/zadykian/ws/mac/internal/conf"
	"github.com/zadykian/ws/mac/internal/effect"
	"github.com/zadykian/ws/mac/internal/setup"
	"github.com/zadykian/ws/mac/internal/step"
)

const setupUsage = "usage: ws setup [--check] STEP\n\nSteps:\n  ssh  write ~/.ssh/config.d/ws and put its Include first in ~/.ssh/config\n"

// setupSteps returns the steps of ws setup, in the order setup.sh runs them.
func setupSteps(c conf.Conf, home string, m effect.Machine, out effect.Out, check bool) []step.Step {
	return []step.Step{
		setup.SSH{Conf: c, Home: home, M: m, Out: out, Now: time.Now, AssumeTunnel: check},
	}
}

// setupCommand runs ws setup: the steps it names, from mac.conf, in the order of setupSteps. Under
// --check it prints each change and makes none.
func setupCommand(ctx context.Context, args []string, uid int, m effect.Machine, getenv func(string) string,
	stderr io.Writer) int {
	check := len(args) > 0 && args[0] == "--check"
	if check {
		args = args[1:]
	}
	if len(args) != 1 {
		_, _ = io.WriteString(stderr, setupUsage)
		return 2
	}
	prog := getenv("WS_PROG")
	if prog == "" {
		prog = "ws setup"
	}
	if uid == 0 {
		// As root, with HOME kept, it would leave root's files in the user's ~/.ssh, and run the
		// user's ~/.local/bin/cld.
		_, _ = fmt.Fprintf(stderr, "%s: run it as yourself, not root: it asks sudo where it needs root\n", prog)
		return 1
	}
	home := getenv("HOME")
	file := getenv("WS_CONF")
	if file == "" {
		file = filepath.Join(home, confFile)
	}
	text, err := m.ReadFile(file)
	if errors.Is(err, fs.ErrNotExist) {
		_, _ = fmt.Fprintf(stderr, "%s: %s is missing: docs/mac.md says what goes in it\n", prog, file)
		return 1
	}
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "%s: %v\n", prog, err)
		return 1
	}
	c, errs := conf.Parse(string(text))
	for _, e := range errs {
		_, _ = fmt.Fprintf(stderr, "%s: %s\n", prog, e.In(file))
	}
	if len(errs) > 0 {
		return 1
	}
	term := effect.NewTerminal(strings.NewReader(""), stderr)
	out := effect.Out(term)
	var chosen []step.Step
	for _, s := range setupSteps(c, home, m, out, check) {
		if s.Name() == args[0] {
			chosen = append(chosen, s)
		}
	}
	if len(chosen) == 0 {
		_, _ = fmt.Fprintf(stderr, "%s: unknown step %q\n\n%s", prog, args[0], setupUsage)
		return 2
	}
	_, err = step.Run(ctx, chosen, out, check)
	term.Flush()
	if err != nil {
		// The runner names the step and its stage before the step's own error, which setup.sh
		// printed alone.
		if inner := errors.Unwrap(err); inner != nil {
			err = inner
		}
		_, _ = fmt.Fprintf(stderr, "%s: %v\n", prog, err)
		return 1
	}
	return 0
}
