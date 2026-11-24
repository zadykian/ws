package cli

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/kind"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/ubuntu/values"
)

// The lockout guards and sshd are tried on the fake machine, whose login no change can cut.

const (
	sshdDropIn = "/etc/ssh/sshd_config.d/10-keys-only.conf"
	keysOfRoot = "/root/.ssh/authorized_keys"
)

// securityMachine is a machine, not a container, whose root holds a key, and
// whose checkers pass.
func securityMachine() *effecttest.Machine {
	m := effecttest.NewMachine()
	m.Results["getent passwd root"] = effecttest.Result{Out: []byte("root:x:0:0:root:/root:/bin/bash\n")}
	m.Dirs["/root"], m.Dirs["/root/.ssh"] = 0o700, 0o700
	m.Files[keysOfRoot], m.Modes[keysOfRoot] = []byte("ssh-ed25519 AAAA me\n"), 0o600
	m.Results["/usr/sbin/sshd -G -f "+sshdDropIn+".ws-check"] = effecttest.Result{}
	m.Packages = effecttest.Packages{}
	return m
}

// securityUnits loads the specs for the default values on m and u, and returns the security role's
// units in the order they run.
func securityUnits(t *testing.T, m *effecttest.Machine, u *effecttest.Units) []kind.Unit {
	v, err := values.Parse(nil)
	if err != nil {
		t.Fatal(err)
	}
	all := loadSpecs(t, kind.Env{M: m, U: u, Out: &effecttest.Out{}, Data: v, Side: "guest", Self: selfPath})
	return slices.DeleteFunc(all, func(u kind.Unit) bool { return !strings.Contains(u.Name, "/security-") })
}

// runUnits runs the units as a write does, and returns its error.
func runUnits(t *testing.T, units []kind.Unit) error {
	_, err := step.Run(t.Context(), kind.Steps(units), &effecttest.Out{}, false)
	return err
}

func TestSecurityUnitsOrder(t *testing.T) {
	var names []string
	for _, u := range securityUnits(t, securityMachine(), &effecttest.Units{}) {
		names = append(names, u.Name)
	}
	want := []string{"guard/security-keys", "service/security-sshd"}
	if !slices.Equal(names[:len(want)], want) {
		t.Errorf("order = %q, want the keys' guard, then sshd", names)
	}
}

func TestSecurityMissingKeyStopsBeforeSshd(t *testing.T) {
	m := securityMachine()
	delete(m.Files, keysOfRoot)
	if err := runUnits(t, securityUnits(t, m, &effecttest.Units{})); err == nil || !strings.Contains(err.Error(), "holds no key") {
		t.Fatalf("run = %v, want the guard's refusal", err)
	}
	if _, ok := m.Files[sshdDropIn]; ok {
		t.Errorf("%s was written past the guard", sshdDropIn)
	}
}

func TestSecuritySshdFileLetsRootInByKeyAlone(t *testing.T) {
	const want = "PermitRootLogin prohibit-password\n"
	m := securityMachine()
	u := &effecttest.Units{States: map[string]effect.UnitState{"ssh.service": {Active: true}}}
	units := securityUnits(t, m, u)
	applyUnit(t, units, "service/security-sshd")
	got := string(m.Files[sshdDropIn])
	if !strings.Contains(got, "PasswordAuthentication no\nKbdInteractiveAuthentication no\n") ||
		!strings.HasSuffix(got, want) || m.Modes[sshdDropIn] != 0o644 {
		t.Errorf("the drop-in at %o:\n%s\nwant it to end %q", m.Modes[sshdDropIn], got, want)
	}
}

func TestSecuritySshdRefusedLeavesTheLiveFile(t *testing.T) {
	m := securityMachine()
	m.Files[sshdDropIn] = []byte("old\n")
	m.Results["/usr/sbin/sshd -G -f "+sshdDropIn+".ws-check"] = effecttest.Result{Err: errors.New("exit status 255")}
	u := &effecttest.Units{States: map[string]effect.UnitState{"ssh.service": {Active: true}}}
	units := securityUnits(t, m, u)
	i := slices.IndexFunc(units, func(u kind.Unit) bool { return u.Name == "service/security-sshd" })
	p, err := units[i].Step.Plan(t.Context())
	if err == nil {
		err = units[i].Step.Apply(t.Context(), p)
	}
	if err == nil || string(m.Files[sshdDropIn]) != "old\n" || slices.Contains(u.Calls, "reload ssh.service") {
		t.Errorf("apply = %v, drop-in %q, calls %q; want the refusal, the old file and no reload", err, m.Files[sshdDropIn], u.Calls)
	}
}

func TestSecuritySshdReloadsOnceOnAChange(t *testing.T) {
	m := securityMachine()
	m.Files[sshdDropIn] = []byte("old\n")
	u := &effecttest.Units{States: map[string]effect.UnitState{"ssh.service": {Active: true}}}
	units := securityUnits(t, m, u)
	applyUnit(t, units, "service/security-sshd")
	if got := applyUnit(t, units, "service/security-sshd"); got != nil || !slices.Equal(u.Calls, []string{"reload ssh.service"}) {
		t.Errorf("second plan %q, calls %q; want one reload", got, u.Calls)
	}
}
