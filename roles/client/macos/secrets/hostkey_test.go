package secrets_test

import (
	"maps"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/roles/client/macos/secrets"
)

// newerKey is ws-a's key once ws-a is rebuilt.
const newerKey = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIHostC"

// profileWrote is ws-a's profile file as fnox leaves it once fnox -P ws-a set has added a value.
const profileWrote = `# ws-a's own values.
[providers.host]
type = "age"

[providers.host.recipients]
nano = "` + recipientB + `"
backup = "` + backupKey + `"
ws-a = "` + hostKey + `"

[secrets]
WS_IKEWARD_PASSWORD= { provider = "host", value = "fnox-age-batch-v1:YWdl" }
`

// repository is a clone's files once ws-a is added.
func repository() secrets.Files {
	return secrets.Files{secrets.SharedFile: []byte(fnoxWrote), "fnox.ws-a.toml": []byte(profileWrote)}
}

// recipientsOf reads a file's recipients, and fails the test where it cannot.
func recipientsOf(t *testing.T, text []byte, provider string) map[string]string {
	r, err := secrets.Recipients(text, provider)
	if err != nil {
		t.Fatalf("%v in\n%s", err, text)
	}
	return r
}

func TestSetHostKeyAddsTheKeyToBothLists(t *testing.T) {
	f, changed, err := secrets.SetHostKey(repository(), "ws-b", otherHost)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{secrets.SharedFile, "fnox.ws-b.toml"}; !slices.Equal(changed, want) {
		t.Errorf("SetHostKey changed %q, want %q", changed, want)
	}
	shared := recipientsOf(t, f[secrets.SharedFile], secrets.SharedProvider)
	if shared["ws-b"] != otherHost || shared["ws-a"] != hostKey || len(shared) != 4 {
		t.Errorf("fnox.toml's recipients are %v, want ws-b's beside the others", shared)
	}
	want := map[string]string{secrets.NanoLabel: recipientB, secrets.BackupLabel: backupKey, "ws-b": otherHost}
	if own := recipientsOf(t, f["fnox.ws-b.toml"], secrets.ProfileProvider); !maps.Equal(own, want) {
		t.Errorf("fnox.ws-b.toml's recipients are %v, want %v", own, want)
	}
	if has, err := secrets.HasSecrets(f["fnox.ws-b.toml"]); has || err != nil {
		t.Errorf("a new profile file holds a secret: %v, %v", has, err)
	}
	if !strings.Contains(string(f["fnox.ws-b.toml"]), "fnox -P ws-b set NAME --provider host") {
		t.Errorf("a new profile file says not how to set a value:\n%s", f["fnox.ws-b.toml"])
	}
}

func TestSetHostKeyReplacesTheHostsOlderKey(t *testing.T) {
	f, changed, err := secrets.SetHostKey(repository(), "ws-a", newerKey)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{secrets.SharedFile, "fnox.ws-a.toml"}; !slices.Equal(changed, want) {
		t.Errorf("SetHostKey changed %q, want %q", changed, want)
	}
	for name, provider := range map[string]string{secrets.SharedFile: secrets.SharedProvider, "fnox.ws-a.toml": secrets.ProfileProvider} {
		r := recipientsOf(t, f[name], provider)
		if r["ws-a"] != newerKey || len(r) != 3 || slices.Contains(slices.Collect(maps.Values(r)), hostKey) {
			t.Errorf("%s's recipients are %v, want ws-a's newer key in place of its older", name, r)
		}
	}
}

func TestSetHostKeyMovesAKeyFromAnotherLabel(t *testing.T) {
	f, changed, err := secrets.SetHostKey(repository(), "ws-b", hostKey)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{secrets.SharedFile, "fnox.ws-a.toml", "fnox.ws-b.toml"}; !slices.Equal(changed, want) {
		t.Errorf("SetHostKey changed %q, want %q", changed, want)
	}
	if shared := recipientsOf(t, f[secrets.SharedFile], secrets.SharedProvider); shared["ws-a"] != "" || shared["ws-b"] != hostKey {
		t.Errorf("fnox.toml's recipients are %v, want the key under ws-b alone", shared)
	}
	old := recipientsOf(t, f["fnox.ws-a.toml"], secrets.ProfileProvider)
	if _, ok := old["ws-a"]; ok || len(old) != 2 {
		t.Errorf("fnox.ws-a.toml's recipients are %v, want nano and backup alone", old)
	}
	if has, err := secrets.HasSecrets(f["fnox.ws-a.toml"]); !has || err != nil {
		t.Errorf("fnox.ws-a.toml lost its secrets: %v, %v", has, err)
	}
}

func TestSetHostKeyChangesNothingWhereTheKeyIsInPlace(t *testing.T) {
	in := repository()
	f, changed, err := secrets.SetHostKey(in, "ws-a", hostKey)
	if err != nil || len(changed) != 0 {
		t.Fatalf("SetHostKey changed %q, %v, want nothing", changed, err)
	}
	for name, text := range in {
		if string(f[name]) != string(text) {
			t.Errorf("%s changed:\n%s", name, f[name])
		}
	}
}

func TestSetHostKeyKeepsEveryOtherByte(t *testing.T) {
	in := repository()
	f, _, err := secrets.SetHostKey(in, "ws-a", newerKey)
	if err != nil {
		t.Fatal(err)
	}
	for name, text := range in {
		want := strings.ReplaceAll(string(text), hostKey, newerKey)
		if string(f[name]) != want {
			t.Errorf("%s is\n%s\nwant\n%s", name, f[name], want)
		}
	}
}

func TestSetHostKeyRefusesARepositoryWithoutNanoOrBackup(t *testing.T) {
	for name, f := range map[string]secrets.Files{
		"no fnox.toml": {},
		"no nano":      {secrets.SharedFile: []byte(strings.Replace(fnoxWrote, "nano = ", "yubi = ", 1))},
		"no backup":    {secrets.SharedFile: []byte(strings.Replace(fnoxWrote, "backup = ", "spare = ", 1))},
	} {
		if _, _, err := secrets.SetHostKey(f, "ws-b", otherHost); err == nil || !strings.Contains(err.Error(), "secrets init") {
			t.Errorf("%s: SetHostKey gives %v, want an error naming secrets init", name, err)
		}
	}
}

func TestSetHostKeyRefusesAHostnameOrKeyOfAnotherForm(t *testing.T) {
	for _, c := range [][2]string{{"ws.lan", otherHost}, {"nano", otherHost}, {"ws-b", "ssh-rsa AAAA"}, {"ws-b", `ssh-ed25519 A"`}} {
		if _, _, err := secrets.SetHostKey(repository(), c[0], c[1]); err == nil {
			t.Errorf("SetHostKey takes %q with %q", c[0], c[1])
		}
	}
}
