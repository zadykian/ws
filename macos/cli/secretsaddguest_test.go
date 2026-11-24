package cli

import (
	"os/exec"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/macos/roles/secrets/secretstest"
	"github.com/zadykian/ws/macos/values"
)

// The guest lab of the host ws, and the lines that ws macos secrets add lab runs through ws.
const (
	guestKeyLine  = "ssh ws incus exec lab -- cat /etc/ssh/ssh_host_ed25519_key.pub"
	guestHostname = "ssh ws incus exec lab -- hostname"
	guestRun      = "ssh -t ws /usr/local/bin/ws ubuntu host guests setup lab --secrets"
	findWs        = "ssh-keygen -F ws -f " + knownHosts
	wsKey         = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIWs"
)

// guestFixture is lab as a guest of ws, with ws's key in known_hosts and none of lab's.
func guestFixture() (*secretstest.Mac, func(stdin string, args ...string) (int, string, string)) {
	p, mac := addFixture(nil)
	mac.Machine.Files["/etc/ws/servers/lab/values.yml"] = []byte(values.Text("WS_HOST", "ws", "WS_SLOT", "1"))
	mac.Machine.Files[knownHosts] = []byte("ws " + wsKey + "\n")
	mac.Results[findWs] = effecttest.Result{Out: []byte("# Host ws found: line 1\nws " + wsKey + "\n")}
	mac.Results[findLab] = effecttest.Result{Err: exec.Command("sh", "-c", "exit 1").Run()}
	delete(mac.Results, hostnameLine)
	delete(mac.Results, runLine)
	mac.Results[guestKeyLine] = effecttest.Result{Out: []byte(labKey + " root@lab\n")}
	mac.Results[guestHostname] = effecttest.Result{Out: []byte("ws-a\n")}
	mac.Results[guestRun] = effecttest.Result{}
	return mac, func(stdin string, args ...string) (int, string, string) {
		return addWs(p, stdin, append([]string{"lab"}, args...)...)
	}
}

func TestSecretsAddReachesAGuestThroughItsHost(t *testing.T) {
	mac, add := guestFixture()
	status, _, stderr := add("")
	if status != 0 {
		t.Fatalf("add of a guest: status = %d, stderr = %q", status, stderr)
	}
	for _, line := range []string{guestKeyLine, guestHostname, guestRun} {
		if !slices.Contains(mac.Calls, "run "+line) {
			t.Errorf("calls lack %q:\n%q", line, mac.Calls)
		}
	}
	if got := string(mac.Machine.Files[knownHosts]); got != "ws "+wsKey+"\nlab "+labKey+"\n" {
		t.Errorf("known_hosts holds %q, want lab's key from its host, recorded with no question", got)
	}
	for _, call := range mac.Calls {
		if strings.HasPrefix(call, "run ssh lab ") || strings.HasPrefix(call, "run ssh -t lab ") || strings.Contains(call, "keyscan") {
			t.Errorf("add of a guest reached the guest itself: %q", call)
		}
	}
}

func TestSecretsAddRefusesAGuestWhoseHostKnownHostsLacks(t *testing.T) {
	mac, add := guestFixture()
	delete(mac.Machine.Files, knownHosts)
	status, _, stderr := add("")
	if status != 1 || !strings.Contains(stderr, "no host key of ws") || cloned(mac.Calls) {
		t.Errorf("status = %d, stderr = %q, cloned = %v; want 1, no host key and no clone", status, stderr,
			cloned(mac.Calls))
	}
	for _, call := range mac.Calls {
		if strings.HasPrefix(call, "run ssh ") || strings.HasPrefix(call, "run ssh -t ") {
			t.Errorf("add reached a host that known_hosts does not hold: %q", call)
		}
	}
}

func TestSecretsAddTakesTheKnownKeyOfAGuest(t *testing.T) {
	mac, add := guestFixture()
	mac.Machine.Files[knownHosts] = []byte("ws " + wsKey + "\nlab " + labKey + "\n")
	mac.Results[findLab] = effecttest.Result{Out: []byte("# Host lab found: line 2\nlab " + labKey + "\n")}
	if status, _, stderr := add(""); status != 0 {
		t.Fatalf("add of a guest: status = %d, stderr = %q", status, stderr)
	}
	if slices.Contains(mac.Calls, "run "+guestKeyLine) {
		t.Errorf("add read the key of a guest that known_hosts holds: %q", mac.Calls)
	}
}

func TestSecretsAddReachesTheHostByAddress(t *testing.T) {
	mac, add := guestFixture()
	target := "-o HostName=192.0.2.7 -o Port=22022 ws"
	mac.Results["ssh "+target+" incus exec lab -- cat /etc/ssh/ssh_host_ed25519_key.pub"] = mac.Results[guestKeyLine]
	mac.Results["ssh "+target+" incus exec lab -- hostname"] = mac.Results[guestHostname]
	run := "ssh -t " + target + " /usr/local/bin/ws ubuntu host guests setup lab --secrets"
	mac.Results[run] = effecttest.Result{}
	if status, _, stderr := add("", "--address", "192.0.2.7", "--port", "22022"); status != 0 {
		t.Fatalf("add with an address: status = %d, stderr = %q", status, stderr)
	}
	if !slices.Contains(mac.Calls, "run "+run) {
		t.Errorf("calls lack %q:\n%q", run, mac.Calls)
	}
}

func TestSecretsAddRefusesAGuestKeyThatIsNoEd25519Key(t *testing.T) {
	mac, add := guestFixture()
	mac.Results[guestKeyLine] = effecttest.Result{Out: []byte("ssh-rsa AAAA root@lab\n")}
	status, _, stderr := add("")
	if status != 1 || !strings.Contains(stderr, "no ed25519 host key") || cloned(mac.Calls) {
		t.Errorf("status = %d, stderr = %q, cloned = %v; want 1, no key and no clone", status, stderr, cloned(mac.Calls))
	}
}
