package link

import (
	"errors"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/remote"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/stages/gh"
	"github.com/zadykian/ws/roles/client/macos/stages/host"
	"github.com/zadykian/ws/roles/server/ubuntu/session"
	"github.com/zadykian/ws/roles/server/ubuntu/session/sessiontest"
)

const (
	hostLine = "ssh-ed25519 AAAAC3"
	hostFP   = "SHA256:abc"
	hostsOf  = "/h/.ssh/known_hosts"
	// wayConf gives the direct way WS_REMOTE on its port, as every Mac has one.
	wayConf = "WS_REMOTE=ws.example.org\nWS_REMOTE_PORT=22022\n"
)

// loginTo is the login's call up to the host, which a server's name follows.
const loginTo = "ssh -o ControlMaster=auto -o ControlPath=/l/direct -o ControlPersist=30m " +
	"-o ConnectTimeout=10 -o StrictHostKeyChecking=yes -o UpdateHostKeys=no -o CheckHostIP=no " +
	"-o ServerAliveInterval=15 -o PermitLocalCommand=no -o HostName=ws.example.org -o Port=22022 " +
	"-o AddressFamily=inet -o BatchMode=yes "

// newHostNamed returns the stage for a server of that name on a machine whose server answers at
// 10.1.1.2:22 with hostLine, and whose known_hosts lacks the name.
func newHostNamed(name string, env map[string]string) (*host.Key, *effecttest.Machine) {
	m := effecttest.NewMachine()
	m.Files["/h/mac.conf"] = []byte("WS_TARGET=lan\n" + wayConf)
	m.Dirs["/l"] = 0o700
	s := remote.SSH{Host: name, M: m, Lock: "/l", Home: "/h", Ways: host.MacWays{M: m, Conf: "/h/mac.conf"}}
	m.Results[wayCall(s)] = effecttest.Result{Out: []byte("user root\nhostname 10.1.1.2\nport 22\n" +
		"userknownhostsfile ~/.ssh/known_hosts\n")}
	m.Results["ssh-keygen -F "+name+" -f "+hostsOf] = effecttest.Result{Err: errors.New("none")}
	m.Results["ssh-keyscan -T 10 -t ed25519 -p 22 10.1.1.2"] = effecttest.Result{
		Out: []byte("10.1.1.2 " + hostLine + "\n")}
	m.Results["ssh-keygen -l -f /l/host.pub"] = effecttest.Result{Out: []byte("256 " + hostFP + " " + name + " (ED25519)\n")}
	return &host.Key{Conf: "/h/mac.conf", Lock: "/l", Tty: true, M: m, Out: &effecttest.Out{},
		SSH: s,
		Env: func(k string) (string, bool) { v, ok := env[k]; return v, ok }}, m
}

// newLogin returns the stage on newHost's machine, where known_hosts holds ws's key.
func newLogin(login effecttest.Result) (*host.Login, *effecttest.Machine) {
	return newLoginNamed("ws", login)
}

// newLoginNamed is newLogin for a server of that name.
func newLoginNamed(name string, login effecttest.Result) (*host.Login, *effecttest.Machine) {
	h, m := newHostNamed(name, nil)
	m.Results["ssh-keygen -F "+name+" -f "+hostsOf] = effecttest.Result{}
	m.Results[loginTo+name+" echo ok"] = login
	ws := macSession(m, h.SSH, h.Out.(*effecttest.Out), session.Status{}, nil)
	return &host.Login{SSH: h.SSH, Ws: ws, Gh: gh.CLI{Conf: "/h/mac.conf", Home: "/h", M: m, Out: h.Out}, Tty: h.Tty, Env: h.Env, P: h.P, Out: h.Out}, m
}

// macSession returns the Mac's session over ssh, whose remote script answers status on m with the
// value's text. A run's events go to events, which may be nil.
func macSession(m *effecttest.Machine, ssh remote.SSH, out *effecttest.Out, status session.Status, events step.Observer) *session.Session {
	ws := session.New(sessiontest.Reach{Reach: host.MacReach{SSH: ssh, Conf: "/h/mac.conf", Out: out}, M: m},
		session.Scripts{Remote: []byte("x"), Files: []byte("y"), Events: events})
	m.Results[sessiontest.Op("status")] = effecttest.Result{Out: sessiontest.Text(status)}
	return ws
}
