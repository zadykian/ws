package main

import (
	"context"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
	"github.com/zadykian/ws/internal/conf"
	"github.com/zadykian/ws/internal/server/values"
	"github.com/zadykian/ws/internal/tui"
	"github.com/zadykian/ws/internal/wizard"
	"github.com/zadykian/ws/mac"
	"github.com/zadykian/ws/step"
)

// serverPick is what a run of ws server wizard's stages takes beside its effects. A list that chose
// ws's run in part gives only, its roles by --only's entries, and one that ticked it whole update,
// as on the Mac.
type serverPick struct {
	check, update bool
	only          []string
}

// serverWizardStages returns the stages of ws server wizard, in order. They act on ws itself, with
// the remote script on sh's stdin, as the Mac's wizard sends it over ssh. events hears ws's run,
// talks runs claude's login, and hand takes what stays by hand.
func serverWizardStages(p process, ask effect.Prompt, talks effect.Foreground, out effect.Out, events step.Observer,
	hand *wizard.Hand, r serverPick) []step.Step {
	m := p.machine
	vals := &wizard.WsValues{Vars: serverVars, User: p.getenv("SUDO_USER"), Tty: p.tty, M: m, P: ask,
		Set: func(key, value string) error { return setServerValue(m, out, key, value) }}
	def := values.Default().User
	ws := serverSession(m, talks, out, events)
	run := &wizard.WsRun{Ws: ws, Await: vals.Name(), Unbooted: !host.Booted(m), Check: r.check, Out: out,
		Version: p.version, Default: def, Update: r.update, Only: r.only}
	hand.Late(ws.Hand)
	return []step.Step{vals, run, &wizard.WsClaude{Ws: ws, Tty: p.tty},
		&wizard.Signing{Ws: ws, GitHub: &wizard.WsGitHub{M: m, Out: out, Hand: hand}}}
}

// serverSession is ws's session to itself on m: the scripts run here as root, as the Mac's ssh logs
// in as host_vars' ws_user. talks runs claude's login, and events hears ws's run.
func serverSession(m effect.Machine, talks effect.Foreground, out effect.Out, events step.Observer) *wizard.Session {
	want := func() (conf.Conf, error) { return wsWant(m) }
	return wizard.NewSession(wizard.WsReach{M: m, Want: want, Default: values.Default().User, Here: talks, Out: out},
		wizard.Scripts{Remote: mac.RemoteScript, Files: mac.RemoteFilesScript, Events: events})
}

// serverOpen is ws's session through a screen, for the host_vars sheet and the state screen, which
// run no login and print nothing.
func serverOpen(p process) sessionOpen {
	return func(ctx context.Context, b *tui.Bridge) (*wizard.Session, error) {
		return serverSession(screenMachine(ctx, b, p.machine), nil, nil, nil), nil
	}
}

// wsWant is what ws's run is to leave, as host_vars names it: a cloud VM where it turns the outbound
// table off, and its ws_user.
func wsWant(m effect.Machine) (conf.Conf, error) {
	v, err := values.Load(m, serverVars)
	target := "lan"
	if !v.Security.Outbound {
		target = "cloud"
	}
	return conf.Conf{Target: target, User: v.User}, err
}
