package secrets_test

import (
	"context"
	"errors"
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/macos/roles/secrets"
)

// The known_hosts of the home /h, and ssh-keygen's search of it for the server lab.
const (
	knownHosts = "/h/.ssh/known_hosts"
	findLine   = "ssh-keygen -F lab -f " + knownHosts
	scanLine   = "ssh-keyscan -t ed25519 -p 22022 192.0.2.7"
)

// knowing is a Mac whose known_hosts ssh-keygen -F answers with out.
func knowing(out string) *effecttest.Machine {
	m := effecttest.NewMachine()
	m.Files[knownHosts] = []byte("lab " + hostKey + "\n")
	m.Results[findLine] = effecttest.Result{Out: []byte(out)}
	return m
}

func TestHostKeyTakesTheEd25519Line(t *testing.T) {
	m := knowing("# Host lab found: line 2 \n" +
		"lab ecdsa-sha2-nistp256 AAAAE2VjZHNhLXNoYTItbmlzdHAyNTYAAAAIbmlzdHAyNTY=\n" +
		"# Host lab found: line 3 \n" +
		"|1|lPIqZ/QGD4Ys4dRxDJWPSzVxDEY=|JwPTCV5s9n6jwY9UJX6AyiOdcRA= " + hostKey + "\n" +
		"# Host lab found: line 4 REVOKED\n" +
		"@revoked lab " + otherHost + "\n")
	key, err := secrets.HostKey(context.Background(), m, "/h", "lab")
	if err != nil || key != hostKey {
		t.Errorf("HostKey gives %q, %v, want %s", key, err, hostKey)
	}
}

func TestHostKeyIsEmptyWithoutTheFileOrTheEntry(t *testing.T) {
	m := effecttest.NewMachine()
	if key, err := secrets.HostKey(context.Background(), m, "/h", "lab"); err != nil || key != "" {
		t.Errorf("HostKey without known_hosts gives %q, %v, want nothing", key, err)
	}
	if slices.ContainsFunc(m.Calls, func(c string) bool { return strings.HasPrefix(c, "run ") }) {
		t.Errorf("HostKey without known_hosts ran %q", m.Calls)
	}
	m = knowing("")
	m.Results[findLine] = effecttest.Result{Err: exitOne()}
	if key, err := secrets.HostKey(context.Background(), m, "/h", "lab"); err != nil || key != "" {
		t.Errorf("HostKey without lab's entry gives %q, %v, want nothing", key, err)
	}
}

func TestHostKeyFailsWhereSshKeygenFails(t *testing.T) {
	m := knowing("")
	m.Results[findLine] = effecttest.Result{Err: errors.New("exit status 255: Cannot stat")}
	if key, err := secrets.HostKey(context.Background(), m, "/h", "lab"); err == nil {
		t.Errorf("HostKey gives %q, want an error", key)
	}
}

func TestHostKeyRefusesTwoEd25519Keys(t *testing.T) {
	m := knowing("lab " + hostKey + "\nlab " + newerKey + "\n")
	_, err := secrets.HostKey(context.Background(), m, "/h", "lab")
	if err == nil || !strings.Contains(err.Error(), "two ed25519 keys") {
		t.Errorf("HostKey gives %v, want two keys refused", err)
	}
}

func TestHostKeyRefusesABadName(t *testing.T) {
	m := knowing("")
	if _, err := secrets.HostKey(context.Background(), m, "/h", "-lab"); err == nil || len(m.Calls) > 0 {
		t.Errorf("HostKey of -lab gives %v after %q, want a refusal before any call", err, m.Calls)
	}
}

func TestKnownKeysTakesEveryHostsEd25519Key(t *testing.T) {
	m := effecttest.NewMachine()
	if keys, err := secrets.KnownKeys(m, "/h"); err != nil || len(keys) != 0 {
		t.Errorf("KnownKeys without known_hosts gives %v, %v, want none", keys, err)
	}
	m.Files[knownHosts] = []byte("# a comment\nlab " + hostKey + "\n" +
		"|1|lPIqZ/QGD4Ys4dRxDJWPSzVxDEY=|JwPTCV5s9n6jwY9UJX6AyiOdcRA= " + newerKey + " a comment\n" +
		"@revoked box " + otherHost + "\nbox ecdsa-sha2-nistp256 AAAAE2VjZHNh\nbad ssh-ed25519 not-base64!\n")
	keys, err := secrets.KnownKeys(m, "/h")
	if want := map[string]bool{hostKey: true, newerKey: true}; err != nil || !maps.Equal(keys, want) {
		t.Errorf("KnownKeys gives %v, %v, want %v", keys, err, want)
	}
}

func TestRecordHostKeyAppendsALine(t *testing.T) {
	m := effecttest.NewMachine()
	m.Dirs["/h"] = 0o755
	if err := secrets.RecordHostKey(m, "/h", "lab", hostKey); err != nil {
		t.Fatal(err)
	}
	if got := string(m.Files[knownHosts]); got != "lab "+hostKey+"\n" || m.Modes[knownHosts] != 0o600 ||
		m.Dirs["/h/.ssh"] != 0o700 {
		t.Errorf("known_hosts is %q, %o in .ssh %o, want lab's line, 600 in 700", got, m.Modes[knownHosts], m.Dirs["/h/.ssh"])
	}
	m.Files[knownHosts] = []byte("box " + otherHost)
	m.Modes[knownHosts] = 0o644
	if err := secrets.RecordHostKey(m, "/h", "lab", hostKey); err != nil {
		t.Fatal(err)
	}
	want := "box " + otherHost + "\nlab " + hostKey + "\n"
	if got := string(m.Files[knownHosts]); got != want || m.Modes[knownHosts] != 0o600 {
		t.Errorf("known_hosts is %q, %o, want %q, 600", got, m.Modes[knownHosts], want)
	}
}

func TestRecordHostKeyRefusesALink(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/h/dotfiles/known_hosts"] = []byte("box " + otherHost + "\n")
	m.Links[knownHosts] = "/h/dotfiles/known_hosts"
	err := secrets.RecordHostKey(m, "/h", "lab", hostKey)
	if err == nil || !strings.Contains(err.Error(), "by hand") {
		t.Errorf("RecordHostKey gives %v, want the link left by hand", err)
	}
	if slices.ContainsFunc(m.Calls, func(c string) bool { return strings.HasPrefix(c, "write") }) {
		t.Errorf("RecordHostKey wrote through a link: %q", m.Calls)
	}
}

func TestRecordHostKeyRefusesAKeyOfAnotherForm(t *testing.T) {
	m := effecttest.NewMachine()
	for _, key := range []string{"ssh-rsa AAAAB3", hostKey + "\nlab " + otherHost, ""} {
		if err := secrets.RecordHostKey(m, "/h", "lab", key); err == nil {
			t.Errorf("RecordHostKey of %q passes, want a refusal", key)
		}
	}
	if len(m.Calls) > 0 {
		t.Errorf("RecordHostKey of a bad key made calls: %q", m.Calls)
	}
}

func TestScanHostKeyTakesTheEd25519Line(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[scanLine] = effecttest.Result{Out: []byte("[192.0.2.7]:22022 " + hostKey + "\n")}
	key, err := secrets.ScanHostKey(context.Background(), m, "192.0.2.7", 22022)
	if err != nil || key != hostKey {
		t.Errorf("ScanHostKey gives %q, %v, want %s", key, err, hostKey)
	}
}

func TestScanHostKeyWithNoKeyFails(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[scanLine] = effecttest.Result{}
	if key, err := secrets.ScanHostKey(context.Background(), m, "192.0.2.7", 22022); err == nil {
		t.Errorf("ScanHostKey of no key gives %q, want an error", key)
	}
	m.Results[scanLine] = effecttest.Result{Err: exitOne()}
	if key, err := secrets.ScanHostKey(context.Background(), m, "192.0.2.7", 22022); err == nil {
		t.Errorf("ScanHostKey of a failed scan gives %q, want an error", key)
	}
}

func TestScanHostKeyRefusesABadTarget(t *testing.T) {
	m := effecttest.NewMachine()
	for _, tc := range []struct {
		host string
		port int
	}{{"-oProxyCommand=x", 22}, {"", 22}, {"192.0.2.7", 0}, {"192.0.2.7", 65536}} {
		if _, err := secrets.ScanHostKey(context.Background(), m, tc.host, tc.port); err == nil {
			t.Errorf("ScanHostKey of %q, %d passes, want a refusal", tc.host, tc.port)
		}
	}
	if len(m.Calls) > 0 {
		t.Errorf("ScanHostKey of a bad target ran %q", m.Calls)
	}
}

func TestFingerprintGivesTheKeyOnStdin(t *testing.T) {
	m := effecttest.NewMachine()
	const line = "256 SHA256:VGc8S0NoA9ZOJ4miob/XKIfRhlAe01DG9Zhy61iO4Qc no comment (ED25519)"
	m.Results["ssh-keygen -l -f -"] = effecttest.Result{Out: []byte(line + "\n")}
	got, err := secrets.Fingerprint(context.Background(), m, hostKey)
	if err != nil || got != line {
		t.Errorf("Fingerprint gives %q, %v, want %s", got, err, line)
	}
	if !slices.Equal(m.Inputs, []string{hostKey + "\n"}) || slices.ContainsFunc(m.Calls,
		func(c string) bool { return strings.Contains(c, "AAAA") }) {
		t.Errorf("Fingerprint's input is %q and its calls %q, want the key on stdin alone", m.Inputs, m.Calls)
	}
}

func TestPubKeyTakesTheEd25519KeyOfAPubFile(t *testing.T) {
	for text, want := range map[string]string{
		"ssh-ed25519 AAAAC3Nza root@lab\n": "ssh-ed25519 AAAAC3Nza",
		"ssh-ed25519 AAAAC3Nza\n":          "ssh-ed25519 AAAAC3Nza",
		"ssh-rsa AAAAB3Nza root@lab\n":     "",
		"":                                 "",
	} {
		if got, err := secrets.PubKey([]byte(text)); err != nil || got != want {
			t.Errorf("secrets.PubKey(%q) = %q, %v; want %q", text, got, err, want)
		}
	}
}
