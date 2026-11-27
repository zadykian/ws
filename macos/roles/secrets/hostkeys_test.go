package secrets_test

import (
	"bytes"
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/macos/roles/secrets"
	"github.com/zadykian/ws/macos/roles/secrets/secretstest"
)

// labKnown is the known_hosts line of lab.
const labKnown = "lab " + labKey + "\n"

const (
	labKey         = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAILab"
	olderKey       = "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOld"
	otherKey       = "ssh-ed25519 AAAA"
	deployKeysLine = secrets.GH + " api --paginate repos/o/n/keys"
	postKey        = secrets.GH + " api -X POST repos/o/n/keys -f title=ws-a -f key=" + labKey + " -F read_only=true"
	sealed         = "fnox-age-batch-v1:c2VhbGVk"
)

// startText is fnox.toml as init leaves it, with one value, and recipients that follow the nano's
// and the backup's.
const startText = `root = true
if_missing = "error"

[providers.age]
type = "age"

[providers.age.recipients]
nano = "age1yubikey1qnano"
backup = "age1qbackup"
%s
[secrets]
S1 = { provider = "age", value = "` + sealed + `" }
`

// profileText is ws-a's file, holding a value, with recipients that follow the nano's and the backup's.
const profileText = `[providers.host]
type = "age"

[providers.host.recipients]
nano = "age1yubikey1qnano"
backup = "age1qbackup"
%s
[secrets]
A1 = { provider = "host", value = "` + sealed + `" }
`

// holding is the repository's files once ws-a's recipient is key and each of more is a recipient of
// fnox.toml too, as "label = key" lines.
func holding(key string, more ...string) map[string]string {
	own := "ws-a = \"" + key + "\"\n"
	return map[string]string{
		secrets.SharedFile:          sprintf(startText, own+strings.Join(more, "\n")),
		secrets.ProfileFile("ws-a"): sprintf(profileText, own),
	}
}

func sprintf(format, arg string) string { return strings.Replace(format, "%s", arg, 1) }

// addMac is a Mac whose known_hosts holds the lines of known, and whose secrets repository holds files.
func addMac(files map[string]string, known string) *secretstest.Mac {
	m := effecttest.NewMachine()
	m.Files["/h/.ssh/known_hosts"] = []byte(known)
	return secretstest.New(m, files)
}

// add runs the add of labKey as hostname on mac, and gives what it said and its lines by hand.
func add(mac *secretstest.Mac, hostname string) (out string, byHand []string, err error) {
	var buf bytes.Buffer
	h := secrets.HostKeys{M: mac, Getenv: func(k string) string { return map[string]string{"HOME": "/h"}[k] },
		Out: &buf, KeyFile: "/h/.config/fnox/age.txt"}
	byHand, err = h.Add(context.Background(), "lab", hostname, labKey)
	return buf.String(), byHand, err
}

func callAt(calls []string, part string) int {
	return slices.IndexFunc(calls, func(c string) bool { return strings.Contains(c, part) })
}

func TestAddRunsGitGhAndFnoxInOrder(t *testing.T) {
	mac := addMac(map[string]string{secrets.SharedFile: sprintf(startText, "")}, labKnown)
	out, _, err := add(mac, "ws-a")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	at := -1
	for _, part := range []string{" git clone ", deployKeysLine, " fnox reencrypt -p age -f", postKey,
		secrets.GH + " pr create", secrets.GH + " pr merge"} {
		i := callAt(mac.Calls, part)
		if i <= at {
			t.Fatalf("%q comes at %d, after %d, in\n%q", part, i, at, mac.Calls)
		}
		at = i
	}
	want := secrets.Change{Topic: "add-ws-a", Title: "chore: add the host key of ws-a",
		Body: "Added by ws macos secrets add lab.", Files: []string{"fnox.toml", "fnox.ws-a.toml"}}
	landed := mac.Landed()
	if len(landed) != 1 || landed[0].Topic != want.Topic || landed[0].Title != want.Title ||
		landed[0].Body != want.Body || !slices.Equal(landed[0].Files, want.Files) {
		t.Errorf("Add landed %+v, want %+v", landed, want)
	}
	if fnoxed := mac.Fnoxed(); len(fnoxed) != 1 {
		t.Errorf("Add ran fnox %q, want one reencrypt of fnox.toml: the new profile holds no value", fnoxed)
	}
	if !strings.Contains(mac.Files()["fnox.ws-a.toml"], labKey) || !strings.Contains(out, secretstest.Pull) {
		t.Errorf("Add left the profile %q and said %q, want the key there and the link", mac.Files(), out)
	}
}

func TestAddOpensNoPullRequestWhereNothingChanged(t *testing.T) {
	mac := addMac(holding(labKey), labKnown)
	mac.Keys = []secrets.DeployKey{{ID: 5, Title: "ws-a", Key: labKey, ReadOnly: true}}
	out, _, err := add(mac, "ws-a")
	if err != nil || !strings.Contains(out, "the secrets already hold the host key of ws-a") {
		t.Fatalf("Add of a key in place: %v, said %q", err, out)
	}
	if len(mac.Fnoxed()) != 0 || len(mac.Landed()) != 0 || callAt(mac.Calls, " -X ") >= 0 {
		t.Errorf("Add of a key in place ran fnox, landed or changed a deploy key: %q", mac.Calls)
	}
}

func TestAddReplacesARebuiltServersKey(t *testing.T) {
	mac := addMac(holding(olderKey), labKnown)
	mac.Keys = []secrets.DeployKey{{ID: 7, Title: "ws-a", Key: olderKey, ReadOnly: true}}
	if _, _, err := add(mac, "ws-a"); err != nil {
		t.Fatalf("Add of a rebuilt server: %v", err)
	}
	deleted, posted := callAt(mac.Calls, secrets.GH+" api -X DELETE repos/o/n/keys/7"), callAt(mac.Calls, postKey)
	pull := callAt(mac.Calls, secrets.GH+" pr create")
	if deleted < 0 || posted < deleted || pull < posted {
		t.Errorf("the older deploy key goes at %d, the new one comes at %d, the pull request opens at %d, want "+
			"that order", deleted, posted, pull)
	}
	want := [][]string{{"reencrypt", "-p", "age", "-f"}, {"-P", "ws-a", "reencrypt", "-p", "host", "-f"}}
	if !slices.EqualFunc(mac.Fnoxed(), want, slices.Equal) {
		t.Errorf("Add ran fnox %q, want %q", mac.Fnoxed(), want)
	}
	if files := mac.Files(); strings.Contains(files["fnox.toml"]+files["fnox.ws-a.toml"], olderKey) {
		t.Errorf("the older key stays in %q", files)
	}
}

// lab comes back as ws-b with labKey. ws-a's olderKey stays in the secrets, which known_hosts no
// longer holds, and the other key is in known_hosts, so that it stays out of the list.
func TestAddListsTheKeysThatKnownHostsLacks(t *testing.T) {
	mac := addMac(holding(olderKey, "ws-c = \""+otherKey+"\""), "ws "+otherKey+"\n"+labKnown)
	mac.Keys = []secrets.DeployKey{{ID: 7, Title: "ws-a", Key: olderKey, ReadOnly: true},
		{ID: 8, Title: "ws-c", Key: otherKey, ReadOnly: true}}
	_, byHand, err := add(mac, "ws-b")
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	want := "By hand: the secrets hold " + olderKey + " under ws-a, a key of no host in known_hosts: where no " +
		"server holds it now, as after a rebuild, drop it from the secrets' recipients and deploy keys"
	if len(byHand) != 1 || byHand[0] != want {
		t.Errorf("Add lists %q, want the one line %q", byHand, want)
	}
}
