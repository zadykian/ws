package main

import (
	"context"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/host"
	"github.com/zadykian/ws/internal/conf"
	"github.com/zadykian/ws/internal/server/values"
	"github.com/zadykian/ws/internal/tui"
	"github.com/zadykian/ws/internal/wizard"
	"github.com/zadykian/ws/mac"
)

// serverWizardStages returns the stages of ws server wizard, in order. They act on ws itself, with
// the remote script on sh's stdin, as the Mac's wizard sends it over ssh. e.events hears ws's run, e.talks
// runs claude's login, and e.hand takes what stays by hand. A list that chose ws's run in part gives
// o.only, and one that ticked it whole o.update, as on the Mac.
func serverWizardStages(e wizardEffects, o wizardOptions) []step.Step {
	p, out, hand := e.p, e.out, e.hand
	m := p.machine
	vals := &wizard.WsValues{Vars: serverVars, User: p.getenv("SUDO_USER"), Tty: p.tty, M: m, P: e.ask,
		Set: func(key, value string) error { return setServerValue(m, out, key, value) }}
	ws := serverSession(m, e.talks, out, e.events)
	run := &wizard.WsRun{Ws: ws, Await: vals.Name(), Unbooted: !host.Booted(m), Check: o.check, Out: out,
		Version: p.version, Update: o.update, Only: o.only}
	hand.Late(ws.Hand)
	return []step.Step{vals, run, &wizard.WsClaude{Ws: ws, Tty: p.tty},
		&wizard.Signing{Ws: ws, GitHub: &wizard.WsGitHub{M: m, Out: out, Hand: hand}}}
}

// serverSession is ws's session to itself on m: the scripts run here as root, as the Mac's ssh logs
// in as host_vars' ws_user. talks runs claude's login, and events hears ws's run.
func serverSession(m effect.Machine, talks effect.Foreground, out effect.Out, events step.Observer) *wizard.Session {
	want := func() (conf.Conf, error) { return wsWant(m) }
	return wizard.NewSession(wizard.WsReach{M: m, Want: want, Here: talks, Out: out},
		wizard.Scripts{Remote: mac.RemoteScript, Files: mac.RemoteFilesScript, Events: events})
}

// serverOpen is ws's session through a screen, for the host_vars sheet and the state screen, which
// run no login and print nothing. It runs on the screen's machine.
func serverOpen(_ context.Context, b *tui.Bridge) (*wizard.Session, error) {
	return serverSession(b.Effects().M, nil, nil, nil), nil
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
