package wizard

import (
	"errors"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/remote"
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

const loginCall = "ssh -o ControlMaster=auto -o ControlPath=/l/direct -o ControlPersist=30m " +
	"-o ConnectTimeout=10 -o StrictHostKeyChecking=yes -o UpdateHostKeys=no -o CheckHostIP=no " +
	"-o ServerAliveInterval=15 -o PermitLocalCommand=no -o HostName=ws.example.org -o Port=22022 " +
	"-o AddressFamily=inet -o BatchMode=yes ws echo ok"

// newHost returns the stage on a machine whose ws answers at 10.1.1.2:22 with hostLine, and whose
// known_hosts lacks ws.
func newHost(env map[string]string) (*host.Key, *effecttest.Machine) {
	m := effecttest.NewMachine()
	m.Files["/h/mac.conf"] = []byte("WS_TARGET=lan\n" + wayConf)
	m.Dirs["/l"] = 0o700
	s := remote.SSH{M: m, Lock: "/l", Home: "/h", Ways: host.MacWays{M: m, Conf: "/h/mac.conf"}}
	m.Results[wayCall(s)] = effecttest.Result{Out: []byte("user root\nhostname 10.1.1.2\nport 22\n" +
		"userknownhostsfile ~/.ssh/known_hosts\n")}
	m.Results["ssh-keygen -F ws -f "+hostsOf] = effecttest.Result{Err: errors.New("none")}
	m.Results["ssh-keyscan -T 10 -t ed25519 -p 22 10.1.1.2"] = effecttest.Result{
		Out: []byte("10.1.1.2 " + hostLine + "\n")}
	m.Results["ssh-keygen -l -f /l/host.pub"] = effecttest.Result{Out: []byte("256 " + hostFP + " ws (ED25519)\n")}
	return &host.Key{Conf: "/h/mac.conf", Lock: "/l", Tty: true, M: m, Out: &effecttest.Out{},
		SSH: s,
		Env: func(k string) (string, bool) { v, ok := env[k]; return v, ok }}, m
}

// newLogin returns the stage on newHost's machine, where known_hosts holds ws's key.
func newLogin(login effecttest.Result) (*host.Login, *effecttest.Machine) {
	h, m := newHost(nil)
	m.Results["ssh-keygen -F ws -f "+hostsOf] = effecttest.Result{}
	m.Results[loginCall] = login
	ws := macSession(m, h.SSH, h.Out.(*effecttest.Out), session.Status{})
	return &host.Login{SSH: h.SSH, Ws: ws, Gh: gh.CLI{Conf: "/h/mac.conf", Home: "/h", M: m, Out: h.Out}, Tty: h.Tty, Env: h.Env, P: h.P, Out: h.Out}, m
}

// macSession returns the Mac's session over ssh, whose remote script answers status on m with the
// value's text.
func macSession(m *effecttest.Machine, ssh remote.SSH, out *effecttest.Out, status session.Status) *session.Session {
	ws := session.New(sessiontest.Reach{Reach: host.MacReach{SSH: ssh, Conf: "/h/mac.conf", Out: out}, M: m},
		session.Scripts{Remote: []byte("x"), Files: []byte("y")})
	m.Results[sessiontest.Op("status")] = effecttest.Result{Out: sessiontest.Text(status)}
	return ws
}
