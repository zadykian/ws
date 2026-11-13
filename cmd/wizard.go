package cmd

import (
	"errors"
	"github.com/zadykian/ws/engine/core/wizard"
	"github.com/zadykian/ws/roles/client/macos/stages/host"
	"github.com/zadykian/ws/roles/server/ubuntu/session"
	"github.com/zadykian/ws/roles/server/ubuntu/stages"
	"io"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/remote"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/stages/answers"
	"github.com/zadykian/ws/roles/client/macos/stages/gh"
	"github.com/zadykian/ws/roles/client/macos/stages/keys"
	"github.com/zadykian/ws/roles/client/macos/stages/link"
	"github.com/zadykian/ws/roles/client/macos/stages/tools"
	"github.com/zadykian/ws/roles/server/ubuntu/values"
)

// wizardOptions is what a run of a wizard's stages takes beside its effects: the flags, and what the
// lists chose. Nil left and only run every stage whole.
type wizardOptions struct {
	check, update bool
	// only is ws's roles where the lists chose ws's run in part, as --only's entries.
	only []string
	// left is the run's LeftOut, which reaches the steps of ws client macos apply too.
	left func(parent, name string) bool
}

// newWizardCmd returns ws client macos wizard.
func newWizardCmd(p cli.Env) *cobra.Command {
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

// wizardRefusal returns why ws client macos wizard does not run here, or an empty string. It names
// the system as uname -s does.
func wizardRefusal(p cli.Env) string {
	switch {
	case p.GOOS != "darwin":
		return "it sets up a Mac, not " + strings.ToUpper(p.GOOS[:1]) + p.GOOS[1:]
	case p.Euid == 0:
		return "run it as yourself, not root: it asks sudo where it needs root"
	}
	return ""
}

// wizardEffects is what the stages act through: a plain run's terminal, or a screen's Bridge.
type wizardEffects struct {
	// p holds the machine that the stages act on.
	p      cli.Env
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
	sudo *host.Sudo
	// stop is the run's Stop, which reaches the steps of ws client macos apply too. Nil runs all.
	stop func() error
	// events hears ws's run. A plain run has none: it follows ws's run by its lines.
	events step.Observer
}

// wizardStages returns the stages of ws client macos wizard, in order.
func wizardStages(e wizardEffects, o wizardOptions) []step.Step {
	p, ask, out, prog, hand := e.p, e.ask, e.out, e.prog, e.hand
	check := o.check
	conf := confPath(p.Getenv, p.Pwd)
	lock := wizard.LockPath(p.LockDir())
	scratch := lock + "/mac.conf"
	// Under --check the stages after Answers read the mac.conf that it would write, a file in the lock,
	// as the old script pointed WS_CONF there.
	later := conf
	if check {
		later = scratch
	}
	ghStage := &gh.Gh{CLI: gh.CLI{Conf: later, Home: p.Getenv("HOME"), M: p.Machine, Out: out}, Tty: p.TTY, F: e.talks}
	hostKey := &host.Key{Conf: later, Lock: lock, Tty: p.TTY, Check: check, Env: p.Lookup,
		SSH: remote.SSH{M: p.Machine, F: e.talks, Lock: lock, Home: p.Getenv("HOME"),
			Ways: host.MacWays{M: p.Machine, Conf: later}}, M: p.Machine, P: ask, Out: out}
	sudo := e.sudo
	ws := session.New(host.MacReach{SSH: hostKey.SSH, Sudo: sudo, Conf: later, Out: out},
		session.Scripts{Remote: session.RemoteScript, Files: session.RemoteFilesScript, Events: e.events})
	sudo.SSH, sudo.Tty, sudo.Check, sudo.P, sudo.Out = hostKey.SSH, p.TTY, check, e.hidden, out
	install := &tools.WsInstall{Ws: ws, Version: p.Version, URL: p.Releases, Out: out}
	run := &stages.WsRun{Ws: ws, Await: install.Name(), Check: check, Out: out, Version: p.Version, Update: o.update, Only: o.only}
	hand.Late(ws.Hand)
	hand.Late(ghStage.Hand)
	setup := wizardSetup(e, o.left, later)
	login := &host.Login{SSH: hostKey.SSH, Ws: ws, Gh: ghStage.CLI, Tty: p.TTY, Check: check, Env: p.Lookup, P: ask, Out: out}
	cable := &link.Cable{Ws: ws, SSH: hostKey.SSH, Conf: later, Check: check, RunDone: run.RunDone,
		MacApply: setup.Run, Env: p.Lookup, P: ask, Prog: prog, Tty: p.TTY, M: p.Machine, Out: out}
	hand.Late(cable.Hand)
	return []step.Step{
		&tools.Tools{M: p.Machine, Out: out, Sleep: time.Sleep},
		&tools.Homebrew{Lock: lock, M: p.Machine, Out: out},
		&answers.Answers{Conf: conf, Defaults: shellDefaults(),
			Prog: prog, Tty: p.TTY, Check: check, Scratch: scratch, M: p.Machine, P: ask, Out: out, Env: p.Lookup},
		&keys.MacKey{Conf: later, Home: p.Getenv("HOME"),
			Lock: lock, Tty: p.TTY,
			M: p.Machine, P: ask, Out: out, Sleep: time.Sleep},
		ghStage,
		setup,
		hostKey,
		login,
		sudo,
		install,
		run,
		&stages.WsClaude{Ws: ws, Tty: p.TTY},
		&stages.Signing{Ws: ws, GitHub: &gh.MacGitHub{Gh: ghStage.CLI, Lock: lock, Check: check}},
		&link.Tunnel{SSH: hostKey.SSH, Ws: ws, Conf: later, Check: check, MacApply: setup.Run, Hand: hand, Home: p.Getenv("HOME"),
			Sleep: time.Sleep, M: p.Machine, Out: out},
		cable,
	}
}

// runWizard runs ws client macos wizard: its stages under a lock, and what stays by hand after a
// run in which a stage acted. Under --check it prints what each stage would do and changes nothing.
func runWizard(c *cobra.Command, p cli.Env, o wizardOptions) error {
	if why := wizardRefusal(p); why != "" {
		return errors.New(why)
	}
	// Homebrew's bin comes first in PATH, as a shell started before its install lacks it, and WS_CONF
	// is set for the scripts that stages run.
	_ = p.Setenv("PATH", "/opt/homebrew/bin:"+p.Getenv("PATH"))
	_ = p.Setenv("WS_CONF", confPath(p.Getenv, p.Pwd))
	var before []func(effect.Out) step.Step
	if o.update {
		// Its own run, before the lock: the new ws takes the place of this process, which would leave the
		// lock held. It runs before the screen too, on lines, or the new ws would find a terminal left raw.
		before = append(before, func(out effect.Out) step.Step {
			return &tools.MacUpdate{M: p.Machine, Out: out, Version: p.Version, URL: p.Releases,
				Home: p.Getenv("HOME"), Args: p.Argv}
		})
	}
	return wizardRun{c, p, macSide(c, p), o}.start(before...)
}

// shellDefaults is the shell role's git identity as lines of key: value, which ws client macos
// wizard offers as its answers.
func shellDefaults() []byte {
	d := values.Default().Shell
	return []byte("shell_git_name: " + d.GitName + "\nshell_git_email: " + d.GitEmail + "\n")
}
