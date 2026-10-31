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

	"github.com/zadykian/ws/mac/internal/conf"
	"github.com/zadykian/ws/mac/internal/effect"
	"github.com/zadykian/ws/mac/internal/setup"
	"github.com/zadykian/ws/mac/internal/step"
)

const setupUsage = "usage: ws setup [--check] [STEP]\n\nWith no STEP, it runs every step in this order, and ssh again, whose plan is empty unless the\ndaemon or cld changed ssh ws's ways.\n\nSteps:\n  ssh    write ~/.ssh/config.d/ws and put its Include first in ~/.ssh/config\n  cable  give the cable's service its address\n  git    set git up with a signing key, and gh\n  gateway  install JetBrains Gateway\n  ghostty  install Ghostty and its font, and write ws.ghostty\n  cld  install cld, and update it\n  tunnel  set up the tunnel: ws, the key, tunnel.conf and the daemon\n  networks  route WS_TUNNEL_NETWORKS and write the resolvers of WS_RESOLVERS\n"

// progOut prefixes a line with the program that prints it, as setup.sh's lines are, and says each
// warning once, as a step plans again after it applies. setup.sh sets the prefix in WS_PROG, so
// that the lines of a step that it runs read as its own.
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

// sayChange prints a change under --check. setup.sh printed a diff as diff -u does, with no prefix and
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

func sshStep(c conf.Conf, home string, m effect.Machine, out effect.Out, check, again bool) step.Step {
	return setup.SSH{Conf: c, Home: home, M: m, Out: out, Now: time.Now, AssumeTunnel: check, Again: again}
}

// setupSteps returns the steps of ws setup, in the order setup.sh runs them.
func setupSteps(c conf.Conf, home, xdg, macDir string, m effect.Machine, out effect.Out, text io.Writer, check bool) []step.Step {
	return []step.Step{
		sshStep(c, home, m, out, check, false),
		setup.Cable{Conf: c, M: m, Out: out},
		&setup.Git{Conf: c, Home: home, XDG: xdg, M: m, Out: out},
		&setup.Gateway{Home: home, M: m, Out: out},
		&setup.Ghostty{Home: home, Settings: macDir + "/ws.ghostty", M: m, Out: out},
		&setup.Cld{Home: home, M: m, Out: out, Check: check},
		&setup.Tunnel{Conf: c, Home: home, Plist: macDir + "/com.github.zadykian.ws-tunnel.plist", M: m, Out: out, Text: text,
			Check: check},
		&setup.Networks{Conf: c, M: m, Out: out, Check: check},
	}
}

// setupCommand runs ws setup: the steps it names, from mac.conf, in the order of setupSteps. Under
// --check it prints each change and makes none.
func setupCommand(ctx context.Context, args []string, uid int, m effect.Machine, getenv func(string) string, stdout,
	stderr io.Writer) int {
	check := len(args) > 0 && args[0] == "--check"
	if check {
		args = args[1:]
	}
	if len(args) > 1 {
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
	out := progOut{term, prog, map[string]bool{}}
	var chosen []step.Step
	steps := setupSteps(c, home, getenv("XDG_CONFIG_HOME"), getenv("WS_MAC_DIR"), m, out, stdout, check)
	if len(args) == 0 {
		chosen = steps
		if !check {
			// The tunnel's daemon and cld's restore tabs change ssh ws's ways, so ssh runs once more.
			// Under --check nothing is applied, so a second plan would only repeat the first.
			chosen = slices.Concat(steps, []step.Step{sshStep(c, home, m, out, check, true)})
		}
	}
	for _, s := range steps {
		if len(args) == 1 && s.Name() == args[0] {
			chosen = append(chosen, s)
		}
	}
	if len(chosen) == 0 {
		_, _ = fmt.Fprintf(stderr, "%s: unknown step %q\n\n%s", prog, args[0], setupUsage)
		return 2
	}
	_, err = step.Run(ctx, chosen, out, check)
	for _, s := range chosen {
		if f, ok := s.(interface{ Finish(context.Context) error }); ok && err == nil {
			err = f.Finish(ctx)
		}
	}
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
