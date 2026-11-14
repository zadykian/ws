package link

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/client/macos/conf"
	"github.com/zadykian/ws/roles/client/macos/stages/host"
	"github.com/zadykian/ws/roles/server/ubuntu/session"
)

// hostWord returns the host that a recorded ssh or ssh-keygen call names. For ssh, the first
// word after the options and their values. For ssh-keygen, the value of -F or -R. Any other
// call, and one that names none, gives false.
func hostWord(call string) (string, bool) {
	f := strings.Fields(call)
	i := slices.IndexFunc(f, func(w string) bool { return w == "ssh" || w == "ssh-keygen" })
	if i < 0 {
		return "", false
	}
	if f[i] == "ssh-keygen" {
		for j := i + 1; j+1 < len(f); j++ {
			if f[j] == "-F" || f[j] == "-R" {
				return f[j+1], true
			}
		}
		return "", false
	}
	for j := i + 1; j < len(f); j++ {
		switch {
		case f[j] == "-o":
			j++
		case !strings.HasPrefix(f[j], "-"):
			return f[j], true
		}
	}
	return "", false
}

// namesTheServer fails for each ssh and ssh-keygen call of m that names a host but lab, and for a
// machine whose stage made no such call.
func namesTheServer(t *testing.T, stage string, m *effecttest.Machine) {
	seen := 0
	for _, c := range m.Calls {
		if h, ok := hostWord(c); ok {
			seen++
			if h != "lab" {
				t.Errorf("%s: %q names the host %q; want lab", stage, c, h)
			}
		}
	}
	if seen == 0 {
		t.Errorf("%s: no call names a host in %q", stage, m.Calls)
	}
}

func TestHostWordReadsTheHostOfACall(t *testing.T) {
	for call, want := range map[string]string{
		"run ssh -o BatchMode=yes -o Port=22 lab echo ok": "lab",
		"foreground ssh -t lab exec bash":                 "lab",
		"run ssh -G -o X=1 lab":                           "lab",
		"run ssh-keygen -F lab -f /h/.ssh/known_hosts":    "lab",
		"run ssh-keygen -l -f /l/host.pub":                "",
	} {
		if got, _ := hostWord(call); got != want {
			t.Errorf("hostWord(%q) = %q; want %q", call, got, want)
		}
	}
}

func TestStagesNameTheServerInEachSSHCall(t *testing.T) {
	ctx := context.Background()
	t.Run("host key", func(t *testing.T) {
		h, m := newHostNamed("lab", map[string]string{"WS_HOST_KEY": "yes"})
		p, err := h.Plan(ctx)
		if err == nil {
			err = h.Apply(ctx, p)
		}
		if got := string(m.Files[hostsOf]); err != nil || !strings.HasPrefix(got, "lab ") {
			t.Errorf("known_hosts = %q, %v; want a line under lab", got, err)
		}
		namesTheServer(t, "host key", m)
	})
	t.Run("login", func(t *testing.T) {
		l, m := newLoginNamed("lab", effecttest.Result{Err: errors.New("Host key verification failed")})
		l.Check = true
		_, err := l.Plan(ctx)
		if err == nil || !strings.Contains(err.Error(), "under lab") || !strings.Contains(err.Error(), "-R lab,") {
			t.Errorf("Plan = %v; want the stop that names lab", err)
		}
		namesTheServer(t, "login", m)
	})
	t.Run("sudo", func(t *testing.T) {
		h, m := newHostNamed("lab", nil)
		m.Results[wayCall(h.SSH)] = effecttest.Result{Out: []byte("user ada\nhostname 10.1.1.2\nport 22\n" +
			"userknownhostsfile ~/.ssh/known_hosts\n")}
		s := &host.Sudo{SSH: h.SSH, Tty: true, P: &effecttest.Prompt{T: t, Hidden: []string{"pw"}}, Out: h.Out}
		p, err := s.Plan(ctx)
		if err == nil {
			_ = s.Apply(ctx, p)
		}
		if !slices.ContainsFunc(m.Calls, func(c string) bool { return strings.Contains(c, "sudo -k -S") }) {
			t.Errorf("no password check in %q", m.Calls)
		}
		namesTheServer(t, "sudo", m)
	})
	t.Run("claude login", func(t *testing.T) {
		h, m := newHostNamed("lab", nil)
		_ = host.MacReach{SSH: h.SSH, Conf: "/h/mac.conf", Out: h.Out}.Claude(ctx)
		namesTheServer(t, "claude login", m)
	})
	t.Run("a command as root", func(t *testing.T) {
		h, m := newHostNamed("lab", nil)
		_, _ = host.MacReach{SSH: h.SSH, Conf: "/h/mac.conf", Out: h.Out}.Shell().Run(ctx, nil, "true")
		namesTheServer(t, "a command as root", m)
	})
	t.Run("tunnel", func(t *testing.T) {
		tun, m := newTunnel(keyB, session.Status{WgKey: keyB, Peered: true, MacKey: keyA}, false)
		tun.SSH.Host, tun.Server = "lab", "lab"
		m.Results["ssh -G lab"] = effecttest.Result{Out: []byte("user root\nhostname 10.99.0.1\n")}
		m.Results[libexec+" client macos tunnel status --server lab"] = effecttest.Result{Out: []byte("latest handshake: 5 s ago\n")}
		m.Results[tunnelCall(tun.SSH, "-o", "BatchMode=yes", "lab", "echo", "ok")] = effecttest.Result{Out: []byte("ok\n")}
		tun.Sleep = func(time.Duration) {}
		if err := tun.handshake(ctx, conf.Conf{}, false); err != nil {
			t.Errorf("handshake = %v", err)
		}
		namesTheServer(t, "tunnel", m)
	})
}
