package secrets_test

import (
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/client/macos/secrets"
)

// The clone's files, and recipients of the forms that fnox reads. A value here is a stand-in for
// an age ciphertext.
const (
	sharedFile  = cloneDir + "/" + secrets.SharedFile
	profileFile = cloneDir + "/fnox.ws-a.toml"
	backupKey   = "age1qbackup"
	hostKey     = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIHostA"
)

// fnoxWrote is fnox.toml as fnox leaves it once fnox set has added two values and #1330 a host.
const fnoxWrote = `root = true
if_missing = "error"

[providers.age]
type = "age"

[providers.age.recipients]
nano = "` + recipientB + `"
backup = "` + backupKey + `"
ws-a = "` + hostKey + `"

[secrets]
WS_GITHUB_TOKEN= { provider = "age", value = "YWdl" }
OTEL_CLAUDE_EXPORTER_OTLP_ENDPOINT= { provider = "age", value = "YWdl" }
`

func TestStartReadsBackAsBothRecipients(t *testing.T) {
	text, err := secrets.Start(recipientB, backupKey)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range []string{"root = true", `if_missing = "error"`} {
		if !strings.Contains(string(text), line+"\n") {
			t.Errorf("Start's text lacks %s:\n%s", line, text)
		}
	}
	m := effecttest.NewMachine()
	m.Files[sharedFile] = text
	conf, ok, err := secrets.ReadConfig(m, sharedFile, secrets.SharedProvider)
	if err != nil || !ok {
		t.Fatalf("ReadConfig of Start's text gives %v, %v", ok, err)
	}
	want := map[string]string{secrets.NanoLabel: recipientB, secrets.BackupLabel: backupKey}
	if !maps.Equal(conf.Recipients, want) || len(conf.Secrets) != 0 {
		t.Errorf("ReadConfig of Start's text gives %+v, want both recipients and no secret", conf)
	}

	for _, bad := range []string{"", "age1q\"", "AGE1Q", "ssh-ed25519 AAAA"} {
		if _, err := secrets.Start(recipientB, bad); err == nil {
			t.Errorf("Start takes the backup %q, want an error", bad)
		}
	}
}

func TestReadConfigTakesFnoxsInlineSecrets(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files[sharedFile] = []byte(fnoxWrote)
	conf, ok, err := secrets.ReadConfig(m, sharedFile, secrets.SharedProvider)
	if err != nil || !ok {
		t.Fatalf("ReadConfig gives %v, %v", ok, err)
	}
	if conf.Recipients["ws-a"] != hostKey || len(conf.Recipients) != 3 {
		t.Errorf("ReadConfig gives the recipients %v, want nano, backup and ws-a", conf.Recipients)
	}
	if want := []string{"OTEL_CLAUDE_EXPORTER_OTLP_ENDPOINT", "WS_GITHUB_TOKEN"}; !slices.Equal(conf.Secrets, want) {
		t.Errorf("ReadConfig gives the secrets %v, want %v", conf.Secrets, want)
	}

	m.Files[profileFile] = []byte("[providers.host]\ntype = \"age\"\n\n[providers.host.recipients]\n" +
		"ws-a = \"" + hostKey + "\"\n\n[secrets]\nWS_IKEWARD_PASSWORD= { provider = \"host\", value = \"YWdl\" }\n")
	conf, _, err = secrets.ReadConfig(m, profileFile, "host")
	if err != nil || conf.Recipients["ws-a"] != hostKey || !slices.Equal(conf.Secrets, []string{"WS_IKEWARD_PASSWORD"}) {
		t.Errorf("ReadConfig of a profile's file gives %+v, %v", conf, err)
	}
}

func TestReadConfigOfAMissingFile(t *testing.T) {
	conf, ok, err := secrets.ReadConfig(effecttest.NewMachine(), sharedFile, secrets.SharedProvider)
	if ok || err != nil || conf.Recipients != nil {
		t.Errorf("ReadConfig of no file gives %+v, %v, %v, want nothing and no error", conf, ok, err)
	}
}

func TestReadConfigRefusesRecipientsAsAList(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files[sharedFile] = []byte("[providers.age]\ntype = \"age\"\nrecipients = [\"" + backupKey + "\"]\n")
	_, _, err := secrets.ReadConfig(m, sharedFile, secrets.SharedProvider)
	if err == nil || !strings.Contains(err.Error(), sharedFile) || !strings.Contains(err.Error(), "list") {
		t.Errorf("a list of recipients gives %v, want an error naming the file and the list", err)
	}
}

func TestReadConfigRefusesAnotherProvider(t *testing.T) {
	for text, want := range map[string]string{
		"[providers.age]\ntype = \"1password\"\n": "1password",
		"[providers.host]\ntype = \"age\"\n":      "[providers.age]",
	} {
		m := effecttest.NewMachine()
		m.Files[sharedFile] = []byte(text)
		_, _, err := secrets.ReadConfig(m, sharedFile, secrets.SharedProvider)
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q gives %v, want an error naming %s", text, err, want)
		}
	}
}
