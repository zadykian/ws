package main

import (
	"cmp"
	"context"
	"errors"
	"io"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/internal/server/values"
	"github.com/zadykian/ws/internal/wizard"
	"github.com/zadykian/ws/mac"
	"github.com/zadykian/ws/step"
)

type wizardOptions struct{ check, update bool }

// newWizardCmd returns ws mac wizard.
func newWizardCmd(p process) *cobra.Command {
	var o wizardOptions
	c := &cobra.Command{
		Use:   "wizard",
		Short: "set the Mac up in stages",
		Long: "At a terminal the stages open as two lists, by the machine each runs on, and the ones chosen run. " +
			"The stages run under a lock, and what stays by hand is listed after a run in which a stage " +
			"acted. Under --check each stage prints what it would do and changes nothing.",
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error { return runWizard(c, p, o) },
	}
	f := c.Flags()
	f.BoolVar(&o.check, "check", false, "print what each stage would do and change nothing")
	f.BoolVar(&o.update, "update", false, "install the latest ws first, and run it in place of this one")
	return c
}

// wizardRefusal returns why ws mac wizard does not run here, or an empty string. It names the system
// as uname -s does.
func wizardRefusal(p process) string {
	switch {
	case p.goos != "darwin":
		return "it sets up a Mac, not " + strings.ToUpper(p.goos[:1]) + p.goos[1:]
	case p.euid == 0:
		return "run it as yourself, not root: it asks sudo where it needs root"
	}
	return ""
}

// wizardEffects is what the stages act through: a plain run's terminal, or a screen's Bridge.
type wizardEffects struct {
	// p holds the machine that the stages act on.
	p      process
	prog   string
	talks  effect.Foreground
	ask    effect.Prompt
	hidden effect.Hidden
	out    effect.Out
	// text takes what a run prints on stdout.
	text io.Writer
	obs  step.Observer
	hand *wizard.Hand
	// sudo is the stage that finds how to reach root on ws. A run takes it from the scan before it, so
	// that its password is asked once.
	sudo *wizard.Sudo
	// stop and left are the run's Stop and LeftOut, which reach the steps of ws mac apply too. Nil ones
	// run all.
	stop func() error
	left func(parent, name string) bool
	// events hears ws's run, and only is what it takes where it runs in part. A plain run has neither:
	// it follows ws's run by its lines.
	events step.Observer
	only   []string
}

// wizardStages returns the stages of ws mac wizard, in order.
func wizardStages(e wizardEffects, o wizardOptions) []step.Step {
	p, ask, out, prog, hand := e.p, e.ask, e.out, e.prog, e.hand
	check := o.check
	conf := confPath(p.getenv, p.pwd)
	lock := wizard.LockPath(cmp.Or(p.getenv("TMPDIR"), "/tmp"))
	scratch := lock + "/mac.conf"
	// Under --check the stages after Answers read the mac.conf that it would write, a file in the lock,
	// as the old script pointed WS_CONF there.
	later := conf
	if check {
		later = scratch
	}
	gh := &wizard.Gh{GhCLI: wizard.GhCLI{Conf: later, Home: p.getenv("HOME"), M: p.machine, Out: out}, Tty: p.tty, F: e.talks}
	host := &wizard.HostKey{Conf: later, Lock: lock, Tty: p.tty, Check: check, Env: p.lookup,
		SSH: wizard.SSH{M: p.machine, F: e.talks, Lock: lock, Home: p.getenv("HOME"), Conf: later}, M: p.machine, P: ask, Out: out}
	sudo := e.sudo
	sudo.Host, sudo.P = host, e.hidden
	ws := wizard.NewSession(wizard.MacReach{SSH: host.SSH, Sudo: sudo, Out: out},
		wizard.Scripts{Remote: mac.RemoteScript, Files: mac.RemoteFilesScript, Events: e.events})
	run := &wizard.WsRun{Ws: ws, Check: check, Out: out, Version: p.version, Update: o.update, Only: e.only,
		Default: values.Default().User}
	hand.Late(ws.Hand)
	hand.Late(gh.Hand)
	install := &wizard.WsInstall{Ws: ws, Version: p.version, URL: p.releases, Out: out}
	setup := wizardSetup(e, later)
	login := &wizard.Login{Ws: ws, Host: host, Gh: gh.GhCLI}
	cable := &wizard.Cable{Ws: ws, Conf: later, Check: check, Version: p.version, Default: run.Default, Update: o.update,
		Setup: setup, Env: p.lookup, P: ask, Prog: prog, Tty: p.tty, M: p.machine, Out: out}
	hand.Late(cable.Hand)
	return []step.Step{
		&wizard.Tools{M: p.machine, Out: out, Sleep: time.Sleep},
		&wizard.Homebrew{Lock: lock, M: p.machine, Out: out},
		&wizard.Answers{Conf: conf, Defaults: shellDefaults(),
			Prog: prog, Tty: p.tty, Check: check, Scratch: scratch, M: p.machine, P: ask, Out: out, Env: p.lookup},
		&wizard.MacKey{Conf: later, Home: p.getenv("HOME"),
			Lock: lock, Tty: p.tty,
			M: p.machine, F: e.talks, P: ask, Out: out, Sleep: time.Sleep},
		gh,
		setup,
		host,
		login,
		sudo,
		install,
		run,
		&wizard.WsClaude{Ws: ws, Tty: p.tty},
		&wizard.Signing{Gh: gh.GhCLI, Ws: ws, Lock: lock, Check: check},
		&wizard.Tunnel{Ws: ws, Conf: later, Check: check, Setup: setup, Hand: hand, Home: p.getenv("HOME"),
			Sleep: time.Sleep, M: p.machine, Out: out},
		cable,
	}
}

// runWizard runs ws mac wizard: its stages under a lock, and what stays by hand after a run in
// which a stage acted. Under --check it prints what each stage would do and changes nothing.
func runWizard(c *cobra.Command, p process, o wizardOptions) error {
	if why := wizardRefusal(p); why != "" {
		return errors.New(why)
	}
	// Homebrew's bin comes first in PATH, as a shell started before its install lacks it, and WS_CONF
	// is set for the scripts that stages run.
	_ = p.setenv("PATH", "/opt/homebrew/bin:"+p.getenv("PATH"))
	_ = p.setenv("WS_CONF", confPath(p.getenv, p.pwd))
	prog := c.CommandPath()
	tmp := cmp.Or(p.getenv("TMPDIR"), "/tmp")
	// The run's context ends with a signal, and its lock goes with it.
	ctx, stop := signal.NotifyContext(c.Context(), syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	term := effect.NewTerminal(c.InOrStdin(), c.ErrOrStderr())
	out := progOut{term, prog, map[string]bool{}}
	obs := out
	if o.update {
		// Its own run, before the lock: the new ws takes the place of this process, which would leave the
		// lock held. It runs before the screen too, on lines, or the new ws would find a terminal left raw.
		u := &wizard.MacUpdate{M: p.machine, Out: out, Version: p.version, URL: p.releases, Home: p.getenv("HOME"), Args: p.argv}
		if _, err := step.RunWith(ctx, []step.Step{u}, step.Options{Observer: obs, Check: o.check}); err != nil {
			return unwrapStep(err)
		}
	}
	if drawsScreen(c, p) {
		stop() // The screen takes the signals, and asks before a signal ends a run.
		return macScreen{c, p, o}.draw()
	}
	hand := &wizard.Hand{}
	closing := func(ctx context.Context) {
		hand.Close(ctx, c.OutOrStdout())
		term.Flush()
	}
	talks, _ := p.machine.(effect.Foreground)
	stages := wizardStages(wizardEffects{p: p, prog: prog, talks: talks, ask: signalPrompt{term}, hidden: term, out: out,
		text: c.OutOrStdout(), obs: obs, hand: hand, sudo: &wizard.Sudo{}}, o)
	_, err := wizard.Run(ctx, p.machine, step.Options{Observer: obs, Check: o.check}, stages, tmp, p.pid, closing)
	// The runner names the stage before the stage's own message.
	return unwrapStep(err)
}

// shellDefaults is the shell role's git identity as lines of key: value, which ws mac wizard
// offers as its answers.
func shellDefaults() []byte {
	d := values.Default().Shell
	return []byte("shell_git_name: " + d.GitName + "\nshell_git_email: " + d.GitEmail + "\n")
}
