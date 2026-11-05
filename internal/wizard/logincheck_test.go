package wizard

import (
	"context"
	"errors"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

const loginCall = "ssh -o ControlMaster=auto -o ControlPath=/l/direct -o ControlPersist=30m " +
	"-o ConnectTimeout=10 -o StrictHostKeyChecking=yes -o UpdateHostKeys=no -o CheckHostIP=no " +
	"-o ServerAliveInterval=15 -o PermitLocalCommand=no -o HostName=ws.example.org -o Port=22022 " +
	"-o AddressFamily=inet -o BatchMode=yes ws echo ok"

// newLogin returns the stage on newHost's machine, where known_hosts holds ws's key.
func newLogin(login effecttest.Result) (*Login, *effecttest.Machine) {
	h, m := newHost(nil)
	m.Results["ssh-keygen -F ws -f "+hostsOf] = effecttest.Result{}
	m.Results[loginCall] = login
	return &Login{Host: h, Gh: &Gh{Conf: "/h/mac.conf", Home: "/h", M: m, Out: h.Out}}, m
}

// sshFailed is the error of a failed ssh, whose stderr ends each line with \r\n.
func sshFailed(stderr string) error { return errors.New("ssh: exit status 255: " + stderr) }

func TestLoginLogin(t *testing.T) {
	w := Way{User: "root", KnownHosts: hostsOf}
	for name, tc := range map[string]struct {
		login effecttest.Result
		user  string
		ok    bool
		err   string
	}{
		"in":      {effecttest.Result{Out: []byte("ok\n")}, "root", true, ""},
		"refused": {effecttest.Result{Err: sshFailed("denied\r\n\r\nno sign\r\n")}, "root", false, "denied; no sign"},
		"user":    {effecttest.Result{Out: []byte("ok\n")}, "ws", false, ""},
	} {
		r, _ := newLogin(tc.login)
		w.User = tc.user
		if ok, msg := r.login(context.Background(), w); ok != tc.ok || msg != tc.err {
			t.Errorf("%s: login = %v, %q; want %v, %q", name, ok, msg, tc.ok, tc.err)
		}
	}
}
