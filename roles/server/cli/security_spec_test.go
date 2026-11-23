package cli

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/kind"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/server/values"
)

// The lockout guards and sshd are tried on the fake machine, whose login no change can cut.

const (
	sshdDropIn  = "/etc/ssh/sshd_config.d/10-keys-only.conf"
	keysOfRoot  = "/root/.ssh/authorized_keys"
	tableDir    = "/usr/local/lib/ws/security"
	unitDir     = "/etc/systemd/system/"
	destroyLine = "nft destroy table inet ws"
	// asRoot is how a run state's command runs as root, with no session's env.
	asRoot = "env HOME=/root env -u XDG_RUNTIME_DIR -u DBUS_SESSION_BUS_ADDRESS " +
		"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin "
	// tableGone is the run state's check that no table is loaded.
	tableGone = asRoot + "sh -c ! nft list table inet ws >/dev/null 2>&1"
)

// outboundUnits are the outbound table's units, in the order they stop.
var outboundUnits = []string{"security-resolvers.path", "security-resolvers.timer", "security-resolvers.service",
	"security-outbound.service"}

// securityMachine is a machine, not a container, whose root holds a key, and
// whose checkers pass.
func securityMachine() *effecttest.Machine {
	m := effecttest.NewMachine()
	m.Results["getent passwd root"] = effecttest.Result{Out: []byte("root:x:0:0:root:/root:/bin/bash\n")}
	m.Dirs["/root"], m.Dirs["/root/.ssh"] = 0o700, 0o700
	m.Files[keysOfRoot], m.Modes[keysOfRoot] = []byte("ssh-ed25519 AAAA me\n"), 0o600
	m.Results["/usr/sbin/sshd -G -f "+sshdDropIn+".ws-check"] = effecttest.Result{}
	m.Results["/usr/sbin/nft -c -f "+tableDir+"/outbound.nft.ws-check"] = effecttest.Result{}
	m.Results[tableGone] = effecttest.Result{}
	m.Packages = effecttest.Packages{}
	return m
}

// securityUnits loads the specs for the values in yml on m and u, and returns the security role's
// units in the order they run.
func securityUnits(t *testing.T, m *effecttest.Machine, u *effecttest.Units, yml string) []kind.Unit {
	v, err := values.Parse([]byte(yml))
	if err != nil {
		t.Fatal(err)
	}
	all := loadSpecs(t, kind.Env{M: m, U: u, Out: &effecttest.Out{}, Data: v, Self: selfPath})
	return slices.DeleteFunc(all, func(u kind.Unit) bool { return !strings.Contains(u.Name, "/security-") })
}

// runUnits runs the units as a write does, and returns its error.
func runUnits(t *testing.T, units []kind.Unit) error {
	_, err := step.Run(t.Context(), kind.Steps(units), &effecttest.Out{}, false)
	return err
}

func TestSecurityUnitsOrder(t *testing.T) {
	var names []string
	for _, u := range securityUnits(t, securityMachine(), &effecttest.Units{}, "") {
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
	if err := runUnits(t, securityUnits(t, m, &effecttest.Units{}, "")); err == nil || !strings.Contains(err.Error(), "holds no key") {
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
	units := securityUnits(t, m, u, "")
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
	units := securityUnits(t, m, u, "")
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
	units := securityUnits(t, m, u, "")
	applyUnit(t, units, "service/security-sshd")
	if got := applyUnit(t, units, "service/security-sshd"); got != nil || !slices.Equal(u.Calls, []string{"reload ssh.service"}) {
		t.Errorf("second plan %q, calls %q; want one reload", got, u.Calls)
	}
}

// onUnits writes the outbound table's units on m, as a run with the table on leaves them, and
// returns the Units that run them.
func onUnits(t *testing.T, m *effecttest.Machine) *effecttest.Units {
	m.Results["nft -s list table inet ws"] = effecttest.Result{Out: []byte("table\n")}
	m.Results[`unshare --net sh -c "$1" "$2" && nft -s list table inet ws sh `+tableDir+"/resolvers.sh "+tableDir+"/outbound.nft"] =
		effecttest.Result{Out: []byte("table\n")}
	u := &effecttest.Units{}
	units := securityUnits(t, m, u, "")
	for _, n := range []string{"service/security-resolvers-fill", "service/security-resolvers-path",
		"service/security-resolvers-timer", "service/security-outbound"} {
		applyUnit(t, units, n)
	}
	return u
}

func TestSecurityOutboundFilesAreTodays(t *testing.T) {
	m := securityMachine()
	u := onUnits(t, m)
	for path, source := range map[string]string{
		tableDir + "/resolvers.sh": "resolvers.sh", tableDir + "/outbound.nft": "outbound.nft",
		unitDir + "security-outbound.service": "security-outbound.service", unitDir + "security-resolvers.path": "security-resolvers.path",
		unitDir + "security-resolvers.timer": "security-resolvers.timer", unitDir + "security-resolvers.service": "security-resolvers.service",
	} {
		want, err := fs.ReadFile(os.DirFS(rolesDir), "security/files/"+source)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(m.Files[path], want) {
			t.Errorf("%s is not files/%s", path, source)
		}
	}
	if m.Modes[tableDir+"/resolvers.sh"] != 0o755 {
		t.Errorf("resolvers.sh at %o, want 755", m.Modes[tableDir+"/resolvers.sh"])
	}
	if got := string(m.Files[unitDir+"security-outbound.service"]); !strings.Contains(got,
		"ExecStart="+tableDir+"/resolvers.sh "+tableDir+"/outbound.nft\n") {
		t.Errorf("the unit does not load the table through the script:\n%s", got)
	}
	for _, want := range []string{"enable security-resolvers.path", "start security-resolvers.path", "enable security-resolvers.timer",
		"enable security-outbound.service", "start security-outbound.service"} {
		if !slices.Contains(u.Calls, want) {
			t.Errorf("calls = %q, want %q", u.Calls, want)
		}
	}
}

// offValues turns the table off.
const offValues = "---\nWS_SECURITY_OUTBOUND: false\n"

func TestSecurityOutboundOffStopsThenRemoves(t *testing.T) {
	m := securityMachine()
	u := onUnits(t, m)
	u.Calls = nil
	m.Files["/run/security-resolvers.conf"] = []byte("x")
	units := securityUnits(t, m, u, offValues)
	if err := runUnits(t, units); err != nil {
		t.Fatal(err)
	}
	var stops []string
	for _, c := range u.Calls {
		if strings.HasPrefix(c, "stop ") {
			stops = append(stops, strings.TrimPrefix(c, "stop "))
		}
	}
	if !slices.Equal(stops, outboundUnits) {
		t.Errorf("stops = %q, want %q", stops, outboundUnits)
	}
	for _, n := range outboundUnits {
		if _, ok := m.Files[unitDir+n]; ok || u.States[n].Enabled {
			t.Errorf("%s: file there %v, state %+v; want it gone and disabled", n, ok, u.States[n])
		}
	}
	for _, p := range []string{unitDir + "security-resolvers.path", tableDir} {
		if !slices.Contains(m.Calls, "root rm -rf -- "+p) {
			t.Errorf("calls = %q, want %s removed", m.Calls, p)
		}
	}
	for _, p := range []string{tableDir + "/outbound.nft", "/run/security-resolvers.conf"} {
		if _, ok := m.Files[p]; ok {
			t.Errorf("%s is left", p)
		}
	}
	for _, unit := range units {
		if got := applyUnit(t, units, unit.Name); got != nil {
			t.Errorf("%s: a second plan %q, want none", unit.Name, got)
		}
	}
}

func TestSecurityOutboundOnAgainRestores(t *testing.T) {
	m := securityMachine()
	u := onUnits(t, m)
	if err := runUnits(t, securityUnits(t, m, u, offValues)); err != nil {
		t.Fatal(err)
	}
	u.Calls = nil
	onUnits(t, m)
	units := securityUnits(t, m, u, "")
	for _, n := range []string{"service/security-resolvers-fill", "service/security-resolvers-path",
		"service/security-resolvers-timer", "service/security-outbound"} {
		applyUnit(t, units, n)
	}
	for _, n := range outboundUnits {
		if _, ok := m.Files[unitDir+n]; !ok {
			t.Errorf("%s was not written again", n)
		}
	}
	for _, n := range []string{"security-resolvers.path", "security-resolvers.timer", "security-outbound.service"} {
		if s := u.States[n]; !s.Enabled || !s.Active {
			t.Errorf("%s: %+v, want it enabled and running again", n, s)
		}
	}
}

func TestSecurityOutboundOffDestroysAHandLoadedTable(t *testing.T) {
	m := securityMachine()
	m.Results[tableGone] = effecttest.Result{Err: errors.New("exit status 1")}
	m.Results[asRoot+destroyLine] = effecttest.Result{}
	units := securityUnits(t, m, &effecttest.Units{}, offValues)
	if got := applyUnit(t, units, "state/security-outbound-off"); len(got) != 1 || !strings.Contains(got[0], destroyLine) {
		t.Errorf("plan = %q, want %q", got, destroyLine)
	}
	if !slices.Contains(m.Calls, "run "+asRoot+destroyLine) {
		t.Errorf("calls = %q, want %q", m.Calls, destroyLine)
	}
}
