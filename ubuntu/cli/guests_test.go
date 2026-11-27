package cli

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"slices"
	"strings"
	"testing"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

const guestsFile = "WS_INCUS_GUESTS:\n  - {name: lab, address: 10.100.0.10, cpu: 4, memory: 8GiB, disk: 200GiB}\n"

// runGuests runs ws ubuntu host guests with stdin, and returns the standard output and the error.
func runGuests(m *effecttest.Machine, stdin string, args ...string) (string, error) {
	c := newGuestsCmd(enginecli.Env{Machine: m})
	var out bytes.Buffer
	c.SetArgs(args)
	c.SetIn(strings.NewReader(stdin))
	c.SetOut(&out)
	c.SetErr(&bytes.Buffer{})
	err := c.Execute()
	return out.String(), err
}

func TestGuestsListShowsTheListThenTheStrays(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files[hostVars] = []byte(guestsFile + "  - {name: build, address: 10.100.0.11, cpu: 2, memory: 4GiB, disk: 100GiB}\n")
	m.Results["incus list --format json"] = effecttest.Result{Out: []byte(
		`[{"name":"lab","status":"Running","profiles":["ws"]},{"name":"old","status":"Stopped","profiles":["ws"]}]`)}
	out, err := runGuests(m, "", "list")
	want := "lab\t10.100.0.10\tRunning\nbuild\t10.100.0.11\tmissing\nold\t-\tStopped\tnot in WS_INCUS_GUESTS\n"
	if err != nil || out != want {
		t.Errorf("list = %q, %v, want %q", out, err, want)
	}
}

func TestGuestsSetupRunsTheGuestsSetupWhateverItsPlanSays(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files[hostVars] = []byte(guestsFile)
	m.Files["/usr/local/bin/ws"] = []byte("binary")
	// lab runs and is in sync: it holds the host's ws, and its own plan finds no change. So only a
	// forced setup runs its setup.
	m.Results["incus list --format json"] = effecttest.Result{Out: []byte(`[{"name":"lab","status":"Running","profiles":["ws"]}]`)}
	m.Results[`incus exec lab -- sh -c test -e "$1" || exit 3; sha256sum -- "$1" sh /usr/local/bin/ws`] = effecttest.Result{
		Out: fmt.Appendf(nil, "%x  /usr/local/bin/ws\n", sha256.Sum256([]byte("binary"))),
	}
	m.Results["incus exec lab -- sh -c systemctl is-system-running --wait >/dev/null 2>&1; cloud-init status --wait >/dev/null sh"] = effecttest.Result{}
	m.Results[`incus exec lab -- sh -c test -x "$1" || exit 3; ws=$1; shift; exec "$ws" ubuntu guest setup --plan "$@" sh /usr/local/bin/ws --secrets`] = effecttest.Result{}
	m.Results["incus exec lab -- /usr/local/bin/ws ubuntu guest setup --secrets"] = effecttest.Result{}
	if _, err := runGuests(m, "", "setup", "lab", "--secrets"); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, c := range m.Calls {
		found = found || c == "run incus exec lab -- /usr/local/bin/ws ubuntu guest setup --secrets"
	}
	if !found {
		t.Errorf("calls %q lack the setup with --secrets", m.Calls)
	}
	if _, err := runGuests(m, "", "setup", "other"); err == nil || !strings.Contains(err.Error(), "names no guest other") {
		t.Errorf("a guest that the list lacks is refused: %v", err)
	}
}

func TestGuestsRemoveAsksThenDeletesAndNamesTheMacsCommand(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files[hostVars] = []byte(guestsFile)
	m.Results["incus list --format json"] = effecttest.Result{Out: []byte(`[{"name":"old","status":"Stopped","profiles":["ws"]},{"name":"x","profiles":["default"]}]`)}
	m.Results["incus delete --force old"] = effecttest.Result{}
	if _, err := runGuests(m, "no\n", "remove", "old"); err == nil || !strings.Contains(err.Error(), "not confirmed") {
		t.Errorf("a no deletes nothing: %v", err)
	}
	if _, err := runGuests(m, "", "remove", "old"); err == nil {
		t.Errorf("no answer deletes nothing")
	}
	for _, c := range m.Calls {
		if strings.Contains(c, "delete") {
			t.Fatalf("calls %q deleted before a yes", m.Calls)
		}
	}
	out, err := runGuests(m, "yes\n", "remove", "old")
	if err != nil || !strings.Contains(out, "ws macos remove old") {
		t.Errorf("remove = %q, %v, want the Mac's command named", out, err)
	}
}

// slotThree is the values of a guest whose tunnel the host ends as wg3.
const slotThree = "WS_TUNNEL_ADDRESS: 10.99.3.1/24\nWS_TUNNEL_PORT: 51823\n"

// removeMachine holds the guest old, which the list lacks, with its values at slot 3 and its
// tunnel on the host, beside the host's own wg0.
func removeMachine() *effecttest.Machine {
	m := effecttest.NewMachine()
	m.Files[hostVars] = []byte(guestsFile)
	m.Results["incus list --format json"] = effecttest.Result{Out: []byte(`[{"name":"old","status":"Stopped","profiles":["ws"]}]`)}
	m.Results["incus delete --force old"] = effecttest.Result{}
	m.Results["ip link delete wg3"] = effecttest.Result{}
	m.Files["/etc/ws/guests/old/values.yml"] = []byte(slotThree)
	for _, f := range []string{"60-wg3.netdev", "60-wg3.network", "60-wg3.key", "60-wg0.netdev", "60-wg0.key"} {
		m.Files["/etc/systemd/network/"+f] = []byte("x\n")
	}
	m.Files["/etc/wireguard/wg3/mac.pub"] = []byte("x\n")
	m.Dirs["/etc/wireguard/wg3"], m.Dirs["/sys/class/net/wg3"] = 0o700, 0o755
	return m
}

// removeOld runs ws ubuntu host guests remove old with the answer, and returns its standard
// output, its standard error and its error.
func removeOld(m *effecttest.Machine, answer string) (string, string, error) {
	c := newGuestsCmd(enginecli.Env{Machine: m})
	var out, errs bytes.Buffer
	c.SetArgs([]string{"remove", "old"})
	c.SetIn(strings.NewReader(answer))
	c.SetOut(&out)
	c.SetErr(&errs)
	err := c.Execute()
	return out.String(), errs.String(), err
}

func TestGuestsRemoveDeletesTheGuestsTunnelAfterTheGuest(t *testing.T) {
	m := removeMachine()
	if _, prompt, _ := removeOld(m, "no\n"); !strings.Contains(prompt, "its tunnel wg3 on the host?") {
		t.Errorf("prompt %q names no tunnel", prompt)
	}
	if _, ok := m.Files["/etc/systemd/network/60-wg3.key"]; !ok {
		t.Fatalf("a no deletes the tunnel's key")
	}
	out, _, err := removeOld(m, "yes\n")
	if err != nil || !strings.Contains(out, "deleted its tunnel wg3") || !strings.Contains(out, "ws macos remove old") {
		t.Fatalf("remove = %q, %v", out, err)
	}
	for _, f := range []string{"60-wg3.netdev", "60-wg3.network", "60-wg3.key"} {
		if _, ok := m.Files["/etc/systemd/network/"+f]; ok {
			t.Errorf("%s stays, for the next guest of slot 3", f)
		}
	}
	if _, ok := m.Files["/etc/wireguard/wg3/mac.pub"]; ok {
		t.Errorf("the old guest's peer stays, for the next guest of slot 3")
	}
	if _, ok := m.Files["/etc/systemd/network/60-wg0.key"]; !ok {
		t.Errorf("wg0's key is gone")
	}
	guest, link := slices.Index(m.Calls, "run incus delete --force old"), slices.Index(m.Calls, "root ip link delete wg3")
	if guest < 0 || link < guest {
		t.Errorf("calls %q do not delete wg3 after the guest", m.Calls)
	}
}

func TestGuestsRemoveNamesTheListedGuestThatHoldsTheSlotNow(t *testing.T) {
	m := removeMachine()
	m.Files["/etc/ws/guests/lab/values.yml"] = []byte(slotThree)
	out, prompt, err := removeOld(m, "yes\n")
	if err != nil || !strings.Contains(prompt, "wg3 on the host, which lab holds now?") {
		t.Fatalf("prompt %q, err %v: lab is not named", prompt, err)
	}
	if !strings.Contains(out, "gives wg3 a new key for lab") || !strings.Contains(out, "/etc/wireguard/wg3 again") {
		t.Errorf("out %q says not how lab gets its tunnel again", out)
	}
}

// The host's values load with a clash of the listed guests' tunnels, and with one bad file: only the
// strict list refuses them, so remove still runs.
func TestGuestsRemoveRunsWhereTheListedGuestsClashOrHoldABadFile(t *testing.T) {
	m := removeMachine()
	m.Files[hostVars] = []byte(guestsFile + "  - {name: build, address: 10.100.0.11, cpu: 2, memory: 4GiB, disk: 100GiB}\n")
	m.Files["/etc/ws/guests/lab/values.yml"] = []byte(slotThree)
	m.Files["/etc/ws/guests/build/values.yml"] = []byte(slotThree)
	if out, _, err := removeOld(m, "yes\n"); err != nil || !strings.Contains(out, "deleted its tunnel wg3") {
		t.Errorf("a clash: remove = %q, %v", out, err)
	}
	m = removeMachine()
	m.Files["/etc/ws/guests/lab/values.yml"] = []byte("WS_NOPE: 1\n")
	if out, _, err := removeOld(m, "yes\n"); err != nil || !strings.Contains(out, "deleted the guest old") {
		t.Errorf("a bad file: remove = %q, %v", out, err)
	}
}

func TestGuestsRemoveWithNoSlotWarnsAndDeletesNoTunnel(t *testing.T) {
	m := removeMachine()
	delete(m.Files, "/etc/ws/guests/old/values.yml")
	out, warning, err := removeOld(m, "yes\n")
	if err != nil || !strings.Contains(out, "deleted the guest old") || !strings.Contains(warning, "values.yml is missing, so no tunnel is deleted") {
		t.Fatalf("remove = %q, %q, %v", out, warning, err)
	}
	if slices.ContainsFunc(m.Calls, func(c string) bool { return strings.HasPrefix(c, "root ") }) {
		t.Errorf("calls %q change the host with no slot named", m.Calls)
	}
}

func TestGuestsRemoveRefusesAListedGuestAndAnInstanceOfAnotherProfile(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files[hostVars] = []byte(guestsFile)
	m.Results["incus list --format json"] = effecttest.Result{Out: []byte(`[{"name":"lab","profiles":["ws"]},{"name":"x","profiles":["default"]}]`)}
	if _, err := runGuests(m, "yes\n", "remove", "lab"); err == nil || !strings.Contains(err.Error(), "still names lab") {
		t.Errorf("a listed guest: %v", err)
	}
	if _, err := runGuests(m, "yes\n", "remove", "x"); err == nil || !strings.Contains(err.Error(), "no guest of ws's") {
		t.Errorf("another profile's instance: %v", err)
	}
}
