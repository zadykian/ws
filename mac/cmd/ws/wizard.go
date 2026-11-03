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
	"time"

	"github.com/zadykian/ws"
	"github.com/zadykian/ws/mac"
	"github.com/zadykian/ws/mac/internal/effect"
	"github.com/zadykian/ws/mac/internal/step"
	"github.com/zadykian/ws/mac/internal/wizard"
)

const wizardUsage = "usage: ws mac wizard [--check] [--update]"

// wizardEnv is what ws mac wizard takes from the process, so that a test gives it another.
type wizardEnv struct {
	goos string
	uid  int
	pid  int
	m    effect.Machine
	// version is this ws's version, as ws version prints it.
	version string
	// argv is ws's arguments, without the program, which --update runs the new ws with.
	argv []string
	// releases is the address of ws's releases, WS_RELEASES_URL or GitHub's.
	releases string
	getenv   func(string) string
	// lookup tells a variable set to nothing from one that is not set.
	lookup func(string) (string, bool)
	// pwd is the directory a relative WS_CONF is taken from.
	pwd   string
	stdin io.Reader
	// tty tells whether stdin is a terminal, which the questions need.
	tty    bool
	stderr io.Writer
	// stdout gets the usage that -h asks for, as wizard.sh printed it, and the text that the Tunnel
	// step prints once mac.conf names ws's key.
	stdout io.Writer
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

// wizardRefusal returns why ws mac wizard does not run here, or an empty string. It names the system
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

// wizardStages returns the stages of ws mac wizard, in order.
func wizardStages(e wizardEnv, p effect.Prompt, out effect.Out, prog string, o wizardOptions, hand *wizard.Hand) []step.Step {
	check := o.check
	conf := confPath(e.getenv, e.pwd)
	scratch := strings.TrimRight(cmp.Or(e.getenv("TMPDIR"), "/tmp"), "/") + "/" + wizard.LockName + "/mac.conf"
	// Under --check the stages after Answers read the mac.conf that it would write, a file in the lock,
	// as the old script pointed WS_CONF there.
	later := conf
	if check {
		later = scratch
	}
	lock := strings.TrimRight(cmp.Or(e.getenv("TMPDIR"), "/tmp"), "/") + "/" + wizard.LockName
	gh := &wizard.Gh{Conf: later, Home: e.getenv("HOME"), Tty: e.tty, M: e.m, Out: out}
	host := &wizard.HostKey{Conf: later, Lock: lock, Tty: e.tty, Check: check, Env: e.lookup,
		SSH: wizard.SSH{M: e.m, Lock: lock, Home: e.getenv("HOME")}, M: e.m, P: p, Out: out}
	run := &wizard.WsRun{Host: host, Version: e.version, Update: o.update,
		Remote: &wizard.Remote{SSH: host.SSH, Script: mac.RemoteScript}}
	hand.Late(run.Hand)
	install := &wizard.WsInstall{Remote: run.Remote, Version: e.version, URL: e.releases, Out: out}
	setup := wizardSetup(e, out, later)
	hook := &wizard.HookMac{Host: host, Gh: gh, Home: e.getenv("HOME"), M: e.m, Out: out}
	hand.Late(hook.Hand)
	root := &wizard.RootKey{Host: host, Gh: gh}
	hookWs := &wizard.HookWs{Host: host, Run: run, Mac: hook, User: e.getenv("USER"), Tty: e.tty, M: e.m, Out: out,
		Files: &wizard.Remote{SSH: host.SSH, Script: mac.RemoteFilesScript}}
	cable := &wizard.Cable{Host: host, Run: run, Root: root, Files: hookWs.Files, Setup: setup, Env: e.lookup,
		P: p, Prog: prog, Tty: e.tty, M: e.m, Out: out}
	hand.Late(cable.Hand)
	return []step.Step{
		&wizard.Tools{M: e.m, Out: out, Sleep: time.Sleep},
		&wizard.Homebrew{Lock: lock, M: e.m, Out: out},
		&wizard.Answers{Conf: conf, Defaults: ws.ShellDefaults,
			Prog: prog, Tty: e.tty, Check: check, Scratch: scratch, M: e.m, P: p, Out: out, Env: e.lookup},
		&wizard.MacKey{Conf: later, Home: e.getenv("HOME"),
			Lock: lock, Tty: e.tty,
			M: e.m, P: p, Out: out, Sleep: time.Sleep},
		gh,
		setup,
		host,
		root,
		install,
		run,
		&wizard.WsClaude{Run: run, Tty: e.tty},
		&wizard.Signing{Gh: gh, Run: run, Check: check},
		&wizard.Tunnel{Host: host, Run: run, Setup: setup, Hand: hand, Home: e.getenv("HOME"),
			Sleep: time.Sleep, M: e.m, Out: out},
		hook,
		hookWs,
		&wizard.Words{Host: host, Root: root, Hook: hookWs, Files: hookWs.Files, Home: e.getenv("HOME"), M: e.m, Out: out},
		cable,
	}
}

// wizardCommand runs ws mac wizard: its stages under a lock, and what stays by hand after a run in
// which a stage acted. Under --check it prints what each stage would do and changes nothing.
func wizardCommand(ctx context.Context, args []string, e wizardEnv) int {
	prog := cmp.Or(e.getenv("WS_PROG"), "ws mac wizard")
	o, help, err := wizardOptionsOf(args)
	if help {
		_, _ = fmt.Fprintln(e.stdout, strings.Replace(wizardUsage, "ws mac wizard", prog, 1))
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
	if o.update {
		// Its own step.Run, before the lock: the new ws takes the place of this process, which would leave the lock held.
		u := &wizard.MacUpdate{M: e.m, Out: out, Version: e.version, URL: e.releases, Home: e.getenv("HOME"), Args: e.argv}
		if _, err := step.Run(ctx, []step.Step{u}, out, o.check); err != nil {
			_, _ = fmt.Fprintf(e.stderr, "%s: %v\n", prog, errors.Unwrap(err))
			return 1
		}
	}
	hand := &wizard.Hand{}
	closing := func(ctx context.Context) {
		if e.stdout != nil {
			hand.Close(ctx, e.stdout)
		}
		term.Flush()
	}
	_, err = wizard.Run(ctx, e.m, out, wizardStages(e, signalPrompt{term}, out, prog, o, hand), o.check, tmp, e.pid, closing)
	if err != nil {
		// The runner names the stage before the stage's own message, which the old script printed alone.
		if inner := errors.Unwrap(err); inner != nil {
			err = inner
		}
		_, _ = fmt.Fprintf(e.stderr, "%s: %v\n", prog, err)
		return 1
	}
	return 0
}
