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
		Long: "The stages run under a lock, and what stays by hand is listed after a run in which a stage " +
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

// wizardStages returns the stages of ws mac wizard, in order.
func wizardStages(p process, stdout io.Writer, ask effect.Prompt, out effect.Out, prog string, o wizardOptions, hand *wizard.Hand) []step.Step {
	check := o.check
	conf := confPath(p.getenv, p.pwd)
	scratch := strings.TrimRight(cmp.Or(p.getenv("TMPDIR"), "/tmp"), "/") + "/" + wizard.LockName + "/mac.conf"
	// Under --check the stages after Answers read the mac.conf that it would write, a file in the lock,
	// as the old script pointed WS_CONF there.
	later := conf
	if check {
		later = scratch
	}
	lock := strings.TrimRight(cmp.Or(p.getenv("TMPDIR"), "/tmp"), "/") + "/" + wizard.LockName
	gh := &wizard.Gh{Conf: later, Home: p.getenv("HOME"), Tty: p.tty, M: p.machine, Out: out}
	host := &wizard.HostKey{Conf: later, Lock: lock, Tty: p.tty, Check: check, Env: p.lookup,
		SSH: wizard.SSH{M: p.machine, Lock: lock, Home: p.getenv("HOME"), Conf: later}, M: p.machine, P: ask, Out: out}
	run := &wizard.WsRun{Host: host, Version: p.version, Update: o.update,
		Secrets: p.getenv("HOME") + "/" + secretsFile, User: p.getenv("USER"),
		Remote: &wizard.Remote{SSH: host.SSH, Script: mac.RemoteScript}}
	hand.Late(run.Hand)
	hand.Late(gh.Hand)
	install := &wizard.WsInstall{Remote: run.Remote, Version: p.version, URL: p.releases, Out: out}
	setup := wizardSetup(p, stdout, out, later)
	root := &wizard.RootKey{Host: host, Gh: gh}
	files := &wizard.Remote{SSH: host.SSH, Script: mac.RemoteFilesScript}
	cable := &wizard.Cable{Host: host, Run: run, Root: root, Files: files, Setup: setup, Env: p.lookup,
		P: ask, Prog: prog, Tty: p.tty, M: p.machine, Out: out}
	hand.Late(cable.Hand)
	return []step.Step{
		&wizard.Tools{M: p.machine, Out: out, Sleep: time.Sleep},
		&wizard.Homebrew{Lock: lock, M: p.machine, Out: out},
		&wizard.Answers{Conf: conf, Defaults: shellDefaults(),
			Prog: prog, Tty: p.tty, Check: check, Scratch: scratch, M: p.machine, P: ask, Out: out, Env: p.lookup},
		&wizard.MacKey{Conf: later, Home: p.getenv("HOME"),
			Lock: lock, Tty: p.tty,
			M: p.machine, P: ask, Out: out, Sleep: time.Sleep},
		gh,
		setup,
		host,
		root,
		install,
		run,
		&wizard.WsClaude{Run: run, Tty: p.tty},
		&wizard.Signing{Gh: gh, Run: run, Check: check},
		&wizard.Tunnel{Host: host, Run: run, Setup: setup, Hand: hand, Home: p.getenv("HOME"),
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
	if o.update {
		// Its own step.Run, before the lock: the new ws takes the place of this process, which would leave the lock held.
		u := &wizard.MacUpdate{M: p.machine, Out: out, Version: p.version, URL: p.releases, Home: p.getenv("HOME"), Args: p.argv}
		if _, err := step.Run(ctx, []step.Step{u}, out, o.check); err != nil {
			return unwrapStep(err)
		}
	}
	hand := &wizard.Hand{}
	closing := func(ctx context.Context) {
		hand.Close(ctx, c.OutOrStdout())
		term.Flush()
	}
	stages := wizardStages(p, c.OutOrStdout(), signalPrompt{term}, out, prog, o, hand)
	_, err := wizard.Run(ctx, p.machine, out, stages, o.check, tmp, p.pid, closing)
	// The runner names the stage before the stage's own message.
	return unwrapStep(err)
}

// shellDefaults is the shell role's git identity as lines of key: value, which ws mac wizard
// offers as its answers.
func shellDefaults() []byte {
	d := values.Default().Shell
	return []byte("shell_git_name: " + d.GitName + "\nshell_git_email: " + d.GitEmail + "\n")
}
