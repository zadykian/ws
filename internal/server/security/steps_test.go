package security

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/internal/server/ufw"
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
	// What ufw's own commands leave in its files.
	m.Files[ufw.UserRules] = []byte("### tuple ### allow tcp 22 0.0.0.0/0 any 0.0.0.0/0 in comment=77733a20535348\n" +
		"### tuple ### allow tcp 22 ::/0 any ::/0 in comment=77733a20535348\n")
	m.Files[ufw.DefaultFile] = []byte("DEFAULT_INPUT_POLICY=\"DROP\"\nDEFAULT_OUTPUT_POLICY=\"ACCEPT\"\n")
	m.Files[ufw.ConfFile] = []byte("ENABLED=yes\n")
	m.Results["ufw status"] = effecttest.Result{Out: []byte("Status: active\n")}
	m.Results[unshare] = effecttest.Result{Out: []byte("table\n")}
	m.Results["nft -s list table inet ws"] = effecttest.Result{Out: []byte("table\n")}
	m.Results["/usr/sbin/nft -c -f "+tableFile+".ws-check"] = effecttest.Result{}
	m.Results["rm -f "+tableFile+".ws-check"] = effecttest.Result{}
	u := &effecttest.Units{}
	u.States = map[string]effect.UnitState{"ufw.service": {Enabled: true}, "fail2ban.service": {Enabled: true, Active: true}}
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
	m, u, ss := fakeSteps(v)
	u.States = map[string]effect.UnitState{"ssh.service": {Active: true}}
	run(t, ss)
	calls, ran := len(u.Calls), len(m.Calls)
	run(t, ss)
	for _, c := range m.Calls[ran:] {
		if (strings.HasPrefix(c, "root ufw") && c != "root ufw status") || strings.HasPrefix(c, "write") {
			t.Errorf("second run made %q", c)
		}
	}
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

func TestUFWPlansSSHTheDefaultsAndTheEnable(t *testing.T) {
	v := values.Default()
	v.Home = "/root"
	m, _, ss := fakeSteps(v)
	delete(m.Files, ufw.UserRules)
	delete(m.Files, ufw.DefaultFile)
	delete(m.Files, ufw.ConfFile)
	var ufwStep step.Step
	for _, s := range ss {
		if s.Name() == "ufw" {
			ufwStep = s
		}
	}
	p, err := ufwStep.Plan(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, c := range p.Changes {
		got = append(got, c.String())
	}
	want := []string{"ufw allow in from any to any port 22 proto tcp", "ufw default deny incoming",
		"ufw default allow outgoing", "ufw enable"}
	if !slices.Equal(got, want) {
		t.Errorf("changes = %q, want %q", got, want)
	}
}
