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
	// add refuses the host, so the hint is the runbook's check at the host's console.
	if strings.Contains(stderr, "secrets add ws") || !strings.Contains(stderr, "ws's host key") {
		t.Errorf("stderr = %q, want the runbook's console check and no add of the host", stderr)
	}
	for _, call := range mac.Calls {
		if strings.HasPrefix(call, "run ssh ") || strings.HasPrefix(call, "run ssh -t ") {
			t.Errorf("add reached a host that known_hosts does not hold: %q", call)
		}
	}
}

// Add refuses for a guest's own file alone. Its host's bad file stops it at no step but the host's
// key, which known_hosts lacks here.
func TestSecretsAddTakesAGuestWhoseHostFileTheFleetRefuses(t *testing.T) {
	for name, tc := range map[string]struct {
		host    string
		files   map[string]string
		wantKey string
	}{
		"host that is no server": {"nope", nil, "no host key of nope"},
		"host whose file is bad": {"ws", map[string]string{"ws": values.Text("WS_SLOT", "999")}, "no host key of ws"},
	} {
		t.Run(name, func(t *testing.T) {
			mac, add := guestFixture()
			mac.Machine.Files["/etc/ws/servers/lab/values.yml"] = []byte(values.Text("WS_HOST", tc.host, "WS_SLOT", "1"))
			for server, text := range tc.files {
				mac.Machine.Files["/etc/ws/servers/"+server+"/values.yml"] = []byte(text)
			}
			delete(mac.Machine.Files, knownHosts)
			status, _, stderr := add("")
			if status != 1 || !strings.Contains(stderr, tc.wantKey) || strings.Contains(stderr, "server ws: ") {
				t.Errorf("status = %d, stderr = %q; want 1 and %q, the host's file not named", status, stderr, tc.wantKey)
			}
		})
	}
}

// A server that names itself as its host is no host of itself: add refuses its file.
func TestSecretsAddRefusesAServerThatNamesItselfAsItsHostForItsFile(t *testing.T) {
	p, mac := addFixture(nil)
	mac.Machine.Files["/etc/ws/servers/lab/values.yml"] = []byte(values.Text("WS_HOST", "lab", "WS_SLOT", "1"))
	status, _, stderr := addWs(p, "", "lab")
	if want := "WS_HOST=lab names the guest's own server"; status != 1 || !strings.Contains(stderr, want) ||
		strings.Contains(stderr, "is the host of") {
		t.Errorf("status = %d, stderr = %q; want 1 and %q", status, stderr, want)
	}
	if i := callAt(mac.Calls, "run "); i >= 0 {
		t.Errorf("add ran %q, want nothing run", mac.Calls[i])
	}
}

func TestSecretsAddRefusesAHost(t *testing.T) {
	p, mac := addFixture(nil)
	mac.Machine.Files["/etc/ws/servers/lab/values.yml"] = []byte(values.Text("WS_HOST", "ws", "WS_SLOT", "1"))
	status, _, stderr := addWs(p, "", "ws")
	if status != 1 || !strings.Contains(stderr, "ws is the host of lab") {
		t.Errorf("add of a host: status = %d, stderr = %q; want 1 and the host refused", status, stderr)
	}
	if i := callAt(mac.Calls, "run "); i >= 0 {
		t.Errorf("add of a host ran %q, want nothing run", mac.Calls[i])
	}
}

func TestSecretsAddKeepsTheKnownKeyOfAGuest(t *testing.T) {
	mac, add := guestFixture()
	mac.Machine.Files[knownHosts] = []byte("ws " + wsKey + "\nlab " + labKey + "\n")
	mac.Results[findLab] = effecttest.Result{Out: []byte("# Host lab found: line 2\nlab " + labKey + "\n")}
	if status, _, stderr := add(""); status != 0 {
		t.Fatalf("add of a guest: status = %d, stderr = %q", status, stderr)
	}
	if !slices.Contains(mac.Calls, "run "+guestKeyLine) || callAt(mac.Calls, "write "+knownHosts) >= 0 {
		t.Errorf("calls %q: want the guest's key read through its host, and known_hosts left as it is", mac.Calls)
	}
}

// A guest made again under its name offers a new key, which its host reads in place of the older
// one that known_hosts holds. The older one would land nothing, and the guest could not decrypt.
func TestSecretsAddReplacesTheChangedKeyOfAGuest(t *testing.T) {
	mac, add := guestFixture()
	mac.Machine.Files[knownHosts] = []byte("ws " + wsKey + "\nlab " + olderKey + "\n")
	mac.Results[findLab] = effecttest.Result{Out: []byte("# Host lab found: line 2\nlab " + olderKey + "\n")}
	drop := "ssh-keygen -R lab -f " + knownHosts
	mac.Results[drop] = effecttest.Result{}
	status, _, stderr := add("")
	if status != 0 || !strings.Contains(stderr, "replaced the older key by the host key of lab") {
		t.Fatalf("add of a rebuilt guest: status = %d, stderr = %q", status, stderr)
	}
	at := callAt(mac.Calls, drop)
	if at < 0 || callAt(mac.Calls, "write "+knownHosts) < at || !slices.Contains(mac.Calls, "run "+postLine) {
		t.Errorf("calls %q: want lab's older key dropped, its new one recorded and kept as the deploy key", mac.Calls)
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
