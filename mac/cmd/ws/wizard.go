package main

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"io"
	"os/signal"
	"strings"
	"syscall"

	"github.com/zadykian/ws/mac/internal/effect"
	"github.com/zadykian/ws/mac/internal/step"
	"github.com/zadykian/ws/mac/internal/wizard"
)

const wizardUsage = "usage: ws wizard [--check] [--update]"

// wizardEnv is what ws wizard takes from the process, so that a test gives it another.
type wizardEnv struct {
	goos   string
	uid    int
	pid    int
	m      effect.Machine
	getenv func(string) string
	// lookup tells a variable set to nothing from one that is not set.
	lookup func(string) (string, bool)
	// pwd is the directory a relative WS_CONF is taken from.
	pwd   string
	stdin io.Reader
	// tty tells whether stdin is a terminal, which the questions need.
	tty    bool
	stderr io.Writer
}

type wizardOptions struct{ check, update bool }

// wizardOptionsOf reads the arguments, or says why it can't. Help is no error.
func wizardOptionsOf(args []string) (o wizardOptions, help bool, err error) {
	for _, a := range args {
		switch a {
		case "--check":
			o.check = true
		case "--update":
			o.update = true
		case "-h", "--help":
			return o, true, nil
		default:
			return o, false, fmt.Errorf("unknown argument '%s'; %s", a, wizardUsage)
		}
	}
	return o, false, nil
}

// wizardRefusal returns why ws wizard does not run here, or an empty string. It names the system
// as uname -s does.
func wizardRefusal(e wizardEnv) string {
	switch {
	case e.goos != "darwin":
		return "it sets up a Mac, not " + strings.ToUpper(e.goos[:1]) + e.goos[1:]
	case e.uid == 0:
		return "run it as yourself, not root: it asks sudo where it needs root"
	}
	return ""
}

// repoDir returns the clone that holds the roles, which wizard.sh found from its own path: WS_REPO,
// else where bootstrap.sh clones it.
func repoDir(getenv func(string) string) string {
	return cmp.Or(getenv("WS_REPO"), getenv("HOME")+"/repository/ws")
}

// wizardStages returns the stages of ws wizard, in order.
func wizardStages(e wizardEnv, p effect.Prompt, out effect.Out, prog string, check bool) []step.Step {
	conf := confPath(e.getenv, e.pwd)
	scratch := strings.TrimRight(cmp.Or(e.getenv("TMPDIR"), "/tmp"), "/") + "/" + wizard.LockName + "/mac.conf"
	return []step.Step{
		&wizard.Answers{Conf: conf, Roles: repoDir(e.getenv) + "/roles/shell/defaults/main.yml",
			Prog: prog, Tty: e.tty, Check: check, Scratch: scratch, M: e.m, P: p, Out: out, Env: e.lookup},
	}
}

// wizardCommand runs ws wizard: its stages under a lock, and what stays by hand after a run in
// which a stage acted. Under --check it prints what each stage would do and changes nothing.
func wizardCommand(ctx context.Context, args []string, e wizardEnv) int {
	prog := cmp.Or(e.getenv("WS_PROG"), "ws wizard")
	o, help, err := wizardOptionsOf(args)
	if help {
		_, _ = fmt.Fprintln(e.stderr, strings.Replace(wizardUsage, "ws wizard", prog, 1))
		return 0
	}
	if err == nil && wizardRefusal(e) != "" {
		err = errors.New(wizardRefusal(e))
	}
	if err != nil {
		_, _ = fmt.Fprintf(e.stderr, "%s: %v\n", prog, err)
		return 1
	}
	tmp := cmp.Or(e.getenv("TMPDIR"), "/tmp")
	// The run's context ends with a signal, and its lock goes with it.
	ctx, stop := signal.NotifyContext(ctx, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	term := effect.NewTerminal(e.stdin, e.stderr)
	out := progOut{term, prog, map[string]bool{}}
	changed, err := wizard.Run(ctx, e.m, out, wizardStages(e, signalPrompt{term}, out, prog, o.check), o.check, tmp, e.pid)
	if err != nil {
		// The runner names the stage before the stage's own message, which wizard.sh printed alone.
		if inner := errors.Unwrap(err); inner != nil {
			err = inner
		}
		_, _ = fmt.Fprintf(e.stderr, "%s: %v\n", prog, err)
		return 1
	}
	if changed && !o.check {
		term.Flush()
	}
	return 0
}
