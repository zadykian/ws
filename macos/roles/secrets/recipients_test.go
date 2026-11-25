package secrets_test

import (
	"maps"
	"strings"
	"testing"

	"github.com/zadykian/ws/macos/roles/secrets"
)

// otherHost is a second server's host key.
const otherHost = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIHostB"

func TestRecipientsReadsTheLabels(t *testing.T) {
	r, err := secrets.Recipients([]byte(fnoxWrote), secrets.SharedProvider)
	want := map[string]string{secrets.NanoLabel: recipientB, secrets.BackupLabel: backupKey, "ws-a": hostKey}
	if err != nil || !maps.Equal(r, want) {
		t.Errorf("Recipients gives %v, %v, want %v", r, err, want)
	}
	if _, err := secrets.Recipients([]byte(fnoxWrote), secrets.ProfileProvider); err == nil {
		t.Error("Recipients of a provider the file lacks gives no error")
	}
}

func TestWithRecipientsRewritesTheTableAlone(t *testing.T) {
	r := map[string]string{"ws-b": otherHost, secrets.BackupLabel: backupKey, "ws-a": hostKey, secrets.NanoLabel: recipientB}
	text, err := secrets.WithRecipients([]byte(fnoxWrote), secrets.SharedProvider, r)
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Replace(fnoxWrote, `ws-a = "`+hostKey+`"`+"\n", `ws-a = "`+hostKey+`"`+"\n"+`ws-b = "`+otherHost+`"`+"\n", 1)
	if string(text) != want {
		t.Errorf("WithRecipients gives\n%s\nwant\n%s", text, want)
	}
}

func TestWithRecipientsOrdersTheLabels(t *testing.T) {
	text := "[providers.age.recipients]\nws-b = \"" + otherHost + "\"\nbackup = \"" + backupKey + "\"\n"
	r := map[string]string{"ws-b": otherHost, "ws-a": hostKey, secrets.BackupLabel: backupKey, secrets.NanoLabel: recipientB}
	out, err := secrets.WithRecipients([]byte("[providers.age]\ntype = \"age\"\n\n"+text), secrets.SharedProvider, r)
	want := "[providers.age]\ntype = \"age\"\n\n[providers.age.recipients]\nnano = \"" + recipientB + "\"\nbackup = \"" +
		backupKey + "\"\nws-a = \"" + hostKey + "\"\nws-b = \"" + otherHost + "\"\n"
	if err != nil || string(out) != want {
		t.Errorf("WithRecipients gives %v and\n%s\nwant\n%s", err, out, want)
	}
}

func TestWithRecipientsKeepsAnEmptyTablesBlankLines(t *testing.T) {
	text := "[providers.host]\ntype = \"age\"\n\n[providers.host.recipients]\n\n[secrets]\n"
	out, err := secrets.WithRecipients([]byte(text), secrets.ProfileProvider, map[string]string{"ws-a": hostKey})
	want := "[providers.host]\ntype = \"age\"\n\n[providers.host.recipients]\nws-a = \"" + hostKey + "\"\n\n[secrets]\n"
	if err != nil || string(out) != want {
		t.Errorf("WithRecipients gives %v and\n%s\nwant\n%s", err, out, want)
	}
}

func TestWithRecipientsRefuses(t *testing.T) {
	table := "[providers.age]\ntype = \"age\"\n\n[providers.age.recipients]\nnano = \"" + recipientB + "\"\n"
	for name, c := range map[string]struct {
		text  string
		r     map[string]string
		words string
	}{
		"no table":       {"[providers.age]\ntype = \"age\"\n", map[string]string{"ws-a": hostKey}, "no table"},
		"a comment":      {table + "# the Nano\n", map[string]string{"ws-a": hostKey}, "# the Nano"},
		"an inline note": {table + "ws-a = \"" + hostKey + "\" # ws\n", map[string]string{"ws-a": hostKey}, "would lose"},
		"a bad label":    {table, map[string]string{"ws a": hostKey}, "bare key"},
		"a bad key":      {table, map[string]string{"ws-a": "ssh-rsa AAAA"}, "ssh-ed25519"},
		"a quote":        {table, map[string]string{"ws-a": `age1q" = 1`}, "age1"},
		"a second table": {table + "\n[providers]\nage.recipients.ws-z = \"" + otherHost + "\"\n",
			map[string]string{"ws-a": hostKey}, "reads back"},
	} {
		if _, err := secrets.WithRecipients([]byte(c.text), secrets.SharedProvider, c.r); err == nil ||
			!strings.Contains(err.Error(), c.words) {
			t.Errorf("%s: WithRecipients gives %v, want an error naming %q", name, err, c.words)
		}
	}
}
