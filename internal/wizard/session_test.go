package wizard

import (
	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/internal/conf"
)

// wsStatusLine is the status op of wsSession's remote script, as ws's own shell runs it.
const wsStatusLine = "env SUDO_USER=ada sh -s -- status"

// macSession returns the Mac's session over ssh, whose remote script answers status on m with the
// value's text.
func macSession(m *effecttest.Machine, ssh SSH, out *effecttest.Out, status wsStatus) *Session {
	ws := NewSession(MacReach{SSH: ssh, Out: out}, Scripts{Remote: []byte("x"), Files: []byte("y")})
	m.Results[remoteLine(ws.remote, "status")] = effecttest.Result{Out: status.text()}
	return ws
}

// wsSession returns ws's own session, whose host_vars names ada, and whose remote script answers
// status with the value's text.
func wsSession(status wsStatus) (*Session, *effecttest.Machine) {
	m := effecttest.NewMachine()
	want := func() (conf.Conf, error) { return conf.Conf{Target: "lan", User: "ada"}, nil }
	ws := NewSession(WsReach{M: m, Want: want, Default: "alice", Here: m, Out: &effecttest.Out{}},
		Scripts{Remote: []byte("x"), Files: []byte("y")})
	m.Results[wsStatusLine] = effecttest.Result{Out: status.text()}
	return ws, m
}
