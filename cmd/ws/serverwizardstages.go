package main

import (
	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
	"github.com/zadykian/ws/internal/conf"
	"github.com/zadykian/ws/internal/server/values"
	"github.com/zadykian/ws/internal/wizard"
	"github.com/zadykian/ws/mac"
	"github.com/zadykian/ws/step"
)

// serverWizardStages returns the stages of ws server wizard, in order. They act on ws itself, with
// the remote script on sh's stdin, as the Mac's wizard sends it over ssh. events hears ws's run.
func serverWizardStages(p process, ask effect.Prompt, out effect.Out, events step.Observer, check bool) []step.Step {
	m := p.machine
	vals := &wizard.WsValues{Vars: serverVars, User: p.getenv("SUDO_USER"), Tty: p.tty, M: m, P: ask,
		Set: func(key, value string) error { return setServerValue(m, out, key, value) }}
	shell := wizard.LocalShell{M: m}
	run := &wizard.WsRun{Want: func() (conf.Conf, error) { return wsWant(m) }, Await: vals.Name(),
		Unbooted: !host.Booted(m), Check: check, Out: out, Version: p.version, Default: values.Default().User,
		Remote: &wizard.Remote{Shell: shell, Script: mac.RemoteScript, Events: events},
		Files:  &wizard.Remote{Shell: shell, Script: mac.RemoteFilesScript}}
	return []step.Step{vals, run}
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
