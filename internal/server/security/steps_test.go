package security

import (
	"errors"
	"slices"
	"testing"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/internal/server/values"
	"github.com/zadykian/ws/step"
)

// fakeSteps is a booted machine with the packages in and a key for the user.
func fakeSteps(v values.Values) (*effecttest.Machine, *effecttest.Units, []step.Step) {
	m := effecttest.NewMachine()
	m.Dirs["/run/systemd/system"] = 0o755
	m.Files["/root/.ssh/authorized_keys"] = []byte("ssh-ed25519 AAAA me\n")
	m.Results["dpkg-query -W -f ${Package} ${db:Status-Abbrev}\n openssh-server ufw fail2ban nftables"] =
		effecttest.Result{Out: []byte("openssh-server ii\nufw ii\nfail2ban ii\nnftables ii\n")}
	m.Results["/usr/sbin/sshd -G -f /etc/ssh/sshd_config.d/10-keys-only.conf.ws-check"] = effecttest.Result{}
	m.Results["rm -f /etc/ssh/sshd_config.d/10-keys-only.conf.ws-check"] = effecttest.Result{}
	u := &effecttest.Units{}
	return m, u, steps(v, m, u, &effecttest.Out{})
}

func run(t *testing.T, ss []step.Step) {
	if _, err := step.Run(t.Context(), ss, &effecttest.Out{}, false); err != nil {
		t.Fatal(err)
	}
}

func TestSSHDKeepsKeysOnlyAndReloads(t *testing.T) {
	v := values.Default()
	v.Home = "/root"
	m, u, ss := fakeSteps(v)
	u.States = map[string]effect.UnitState{"ssh.service": {Active: true}}
	run(t, ss)
	if got := string(m.Files["/etc/ssh/sshd_config.d/10-keys-only.conf"]); got == "" {
		t.Error("the drop-in was not written")
	}
	if !slices.Contains(u.Calls, "reload ssh.service") {
		t.Errorf("calls = %q, want a reload of ssh.service", u.Calls)
	}
}

func TestSecondRunPlansNothing(t *testing.T) {
	v := values.Default()
	v.Home = "/root"
	_, u, ss := fakeSteps(v)
	u.States = map[string]effect.UnitState{"ssh.service": {Active: true}}
	run(t, ss)
	calls := len(u.Calls)
	run(t, ss)
	if len(u.Calls) != calls {
		t.Errorf("second run called %q", u.Calls[calls:])
	}
}

func TestSSHDKeepsTheOldFileWhereSSHDRefusesTheNew(t *testing.T) {
	v := values.Default()
	v.Home = "/root"
	m, _, ss := fakeSteps(v)
	m.Results["/usr/sbin/sshd -G -f /etc/ssh/sshd_config.d/10-keys-only.conf.ws-check"] =
		effecttest.Result{Err: errors.New("bad option")}
	if _, err := step.Run(t.Context(), ss, &effecttest.Out{}, false); err == nil {
		t.Error("a file that sshd refuses was written")
	}
	if _, ok := m.Files["/etc/ssh/sshd_config.d/10-keys-only.conf"]; ok {
		t.Error("the refused file replaced the target")
	}
}
