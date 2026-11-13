package setup_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/client/macos/conf"
	"github.com/zadykian/ws/roles/client/macos/internal/mactest"
	"github.com/zadykian/ws/roles/client/macos/setup"
	"github.com/zadykian/ws/roles/internal/sshcfg"
)

// keyUses are the two uses, each with the variable, the words and the page that its stops name.
var keyUses = []struct {
	use               setup.KeyUse
	name, words, page string
	conf              func(pub string) conf.Conf
}{
	{setup.LoginKey, "WS_IDENTITY_PUB_KEY", "ws's login key", "docs/mac.md",
		func(pub string) conf.Conf { return conf.Conf{IdentityPubKey: pub} }},
	{setup.SigningKey, "WS_GIT_SIGNING_PUB_KEY", "git's signing key", "docs/mac-git.md",
		func(pub string) conf.Conf { return conf.Conf{GitSigningPubKey: pub} }},
}

const (
	inBox    = mactest.AgentData + "/PublicKeys/a.pub"
	filePub  = "~/k.pub"
	fileKey  = "ssh-ed25519 AAAAC3Nz me@mac\n"
	fileLine = mactest.AgentHome + "/k.pub"
	mldsa    = "ssh-mldsa-65 AAAADHNzaC1tbGRzYS02NQ== ws\n"
)

func TestReadKeyStops(t *testing.T) {
	for _, tc := range []struct {
		name string
		set  func(m *effecttest.Machine) string // returns the .pub that mac.conf names
		want string                             // VAR, WORDS and PAGE stand for the use's
	}{
		{"none named, no agent", func(*effecttest.Machine) string { return "" },
			"mac.conf names no VAR, WORDS: run ws client macos wizard to pick or make one in Secretive"},
		{"none named, two agent keys", func(m *effecttest.Machine) string {
			mactest.AgentMachine(m, mactest.AgentKey, mldsa)
			return ""
		}, "mac.conf names no VAR, WORDS: run ws client macos wizard, or add one of these lines to mac.conf, VAR=" +
			sshcfg.AgentKeys(mactest.AgentKey)[0].Pub() + " for " + sshcfg.AgentKeys(mactest.AgentKey)[0].Label()},
		{"missing", func(*effecttest.Machine) string { return filePub },
			fileLine + ", mac.conf's VAR, is missing: make the key first, as PAGE says"},
		{"unreadable", func(m *effecttest.Machine) string {
			m.Root[fileLine] = m.Root[mactest.AgentSock]
			return filePub
		}, "cannot read " + fileLine + ": make it readable, or name another in mac.conf's VAR"},
		{"denied in Secretive's container, no key listed", func(m *effecttest.Machine) string {
			mactest.AgentMachine(m)
			m.Root[inBox] = m.Root[mactest.AgentSock]
			return inBox
		}, "cannot read " + inBox + ", and Secretive's agent lists no key of it: open Secretive, or name another in mac.conf's VAR"},
		{"certificate", fileText("ssh-ed25519-cert-v01@openssh.com AAAA\n"),
			fileLine + " is a certificate: name its key's own .pub in mac.conf's VAR"},
		{"type ws lacks", fileText("ssh-mldsa-65 AAAA\n"),
			fileLine + " holds a key of type ssh-mldsa-65, which ws's OpenSSH lacks: name the .pub of a P-256 or ed25519 key in mac.conf's VAR"},
		{"type ws lacks, in the agent", func(m *effecttest.Machine) string {
			mactest.AgentMachine(m, mldsa)
			return sshcfg.AgentKeys(mldsa)[0].Pub()
		}, "which ws's OpenSSH lacks: name the .pub of a P-256 or ed25519 key in mac.conf's VAR"},
		{"private key", fileText("-----BEGIN OPENSSH PRIVATE KEY-----\nAAAA\n-----END OPENSSH PRIVATE KEY-----\n"),
			fileLine + " holds no public key alone"},
		{"ssh-keygen fails", func(m *effecttest.Machine) string {
			fileText(fileKey)(m)
			m.Results["ssh-keygen -l -f "+fileLine] = effecttest.Result{Err: errors.New("bad")}
			return filePub
		}, "ssh-keygen cannot read " + fileLine + ": name the .pub of a P-256 or ed25519 key in mac.conf's VAR"},
	} {
		for _, u := range keyUses {
			m := effecttest.NewMachine()
			pub := tc.set(m)
			_, err := setup.ReadKey(context.Background(), m, u.conf(pub), mactest.AgentHome, u.use)
			want := strings.NewReplacer("VAR", u.name, "WORDS", u.words, "PAGE", u.page).Replace(tc.want)
			if err == nil || !strings.Contains(err.Error(), want) {
				t.Errorf("%s, %s: err = %v; want %q", tc.name, u.name, err, want)
			}
		}
	}
}

func fileText(text string) func(m *effecttest.Machine) string {
	return func(m *effecttest.Machine) string {
		m.Files[fileLine] = []byte(text)
		m.Results["ssh-keygen -l -f "+fileLine] = effecttest.Result{}
		return filePub
	}
}

func TestReadKeyReads(t *testing.T) {
	line := sshcfg.AgentKeys(mactest.AgentKey)[0]
	for _, tc := range []struct {
		name            string
		set             func(m *effecttest.Machine) string
		pub, file, text string
		askedAgent      bool
	}{
		{"a file key", fileText(fileKey), filePub, fileLine, fileKey, false},
		{"a .pub in the container, held", func(m *effecttest.Machine) string {
			mactest.AgentMachine(m, mactest.AgentKey)
			return "~/" + sshcfg.SecretiveData + "/PublicKeys/" + line.Name
		}, line.Pub(), mactest.CopyOf(mactest.AgentKey), mactest.AgentKey, true},
		{"the copy before it exists", func(m *effecttest.Machine) string {
			mactest.AgentMachine(m, mactest.AgentKey, mactest.AgentGit)
			return sshcfg.AgentKeys(mactest.AgentGit)[0].Pub()
		}, sshcfg.AgentKeys(mactest.AgentGit)[0].Pub(), mactest.CopyOf(mactest.AgentGit), mactest.AgentGit, true},
		{"the agent down, the copy there", func(m *effecttest.Machine) string {
			mactest.AgentMachine(m)
			m.Results[mactest.AgentList] = effecttest.Result{Err: errors.New("exit status 2: Connection refused")}
			m.Files[mactest.CopyOf(mactest.AgentKey)] = []byte(mactest.AgentKey)
			m.Results["ssh-keygen -l -f "+mactest.CopyOf(mactest.AgentKey)] = effecttest.Result{}
			return line.Pub()
		}, line.Pub(), mactest.CopyOf(mactest.AgentKey), mactest.AgentKey, true},
	} {
		for _, u := range keyUses {
			m := effecttest.NewMachine()
			pub := tc.set(m)
			k, err := setup.ReadKey(context.Background(), m, u.conf(pub), mactest.AgentHome, u.use)
			if err != nil || k.Pub != tc.pub || k.File != tc.file || k.Text != tc.text || k.Key.Base64 == "" {
				t.Errorf("%s, %s: ReadKey = %+v, %v; want %s, %s, text %q", tc.name, u.name, k, err, tc.pub, tc.file, tc.text)
			}
			if asked := strings.Contains(strings.Join(m.Calls, "\n"), "ssh-add"); asked != tc.askedAgent {
				t.Errorf("%s, %s: asked the agent = %v; want %v", tc.name, u.name, asked, tc.askedAgent)
			}
		}
	}
}
