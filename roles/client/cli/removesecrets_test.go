package cli

import (
	"errors"
	"os/exec"
	"slices"
	"strings"
	"testing"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/client/macos/secrets"
	"github.com/zadykian/ws/roles/client/macos/secrets/secretstest"
)

// The bootstrap that remove runs last.
const bootstrapLine = "env MISE_LOCKED=1 mise -C /opt/ws/roles/client bootstrap"

// removeSecretsFixture is removeFixture on a Mac that ran init, whose known_hosts holds lab's key,
// and whose secrets repository holds files. rm only records its calls, so the clone's files stay
// for the test to read.
func removeSecretsFixture(files map[string]string) (enginecli.Env, *secretstest.Mac) {
	p, fake := removeFixture()
	fake.Files[fnoxKeyFile] = []byte("AGE-PLUGIN-YUBIKEY-1QQQ\n")
	fake.Files[knownHosts] = []byte("ws ssh-ed25519 AAAA\nlab " + labKey + "\n")
	fake.Results[findLab] = effecttest.Result{Out: []byte("lab " + labKey + "\n")}
	fake.Results[bootstrapLine] = effecttest.Result{}
	mac := secretstest.New(fake, files)
	p.Machine = mac
	return p, mac
}

func TestRemoveDropsTheKeyFromTheSecretsBeforeTheHost(t *testing.T) {
	p, mac := removeSecretsFixture(heldBy(t, labKey))
	mac.Keys = []secrets.DeployKey{{ID: 5, Title: "ws-a", Key: labKey, ReadOnly: true}}
	status, stdout, stderr := runWs(p, "client", "remove", "lab")
	if status != 0 {
		t.Fatalf("remove lab: status = %d, stderr = %q", status, stderr)
	}
	at := -1
	for _, part := range []string{findLab, keysLine, " fnox reencrypt -p age -f", secrets.GH + " pr create",
		secrets.GH + " pr merge", secrets.GH + " api -X DELETE repos/o/n/keys/5", "run ssh-keygen -R lab",
		"rm -rf /etc/ws/servers/lab"} {
		i := callAt(mac.Calls, part)
		if i <= at {
			t.Fatalf("%q comes at %d, after %d, in\n%q", part, i, at, mac.Calls)
		}
		at = i
	}
	want := secrets.Change{Topic: "remove-ws-a", Title: "chore: remove the host key of ws-a",
		Body: "Removed by ws client remove lab.", Files: []string{"fnox.toml", "fnox.ws-a.toml"}}
	if landed := mac.Landed(); len(landed) != 1 || !equalChange(landed[0], want) {
		t.Errorf("remove landed %+v, want %+v", landed, want)
	}
	wantFnox := [][]string{{"reencrypt", "-p", "age", "-f"}, {"-P", "ws-a", "reencrypt", "-p", "host", "-f"}}
	if !slices.EqualFunc(mac.Fnoxed(), wantFnox, slices.Equal) {
		t.Errorf("remove ran fnox %q, want %q", mac.Fnoxed(), wantFnox)
	}
	files := mac.Files()
	if profile, ok := files["fnox.ws-a.toml"]; !ok || strings.Contains(files["fnox.toml"]+profile, labKey) {
		t.Errorf("remove left the files %q, want ws-a's profile kept and lab's key out of both", files)
	}
	if !strings.HasPrefix(stdout, "opened "+secretstest.Pull) || strings.Contains(stdout, "By hand") {
		t.Errorf("stdout = %q, want the pull request first and nothing by hand", stdout)
	}
}

func TestRemoveOfAKeyThatIsNoRecipientLandsNothing(t *testing.T) {
	p, mac := removeSecretsFixture(nil)
	mac.Keys = []secrets.DeployKey{{ID: 9, Title: "ws-old", Key: labKey, ReadOnly: true},
		{ID: 4, Title: "ws-b", Key: olderKey, ReadOnly: true}}
	status, stdout, stderr := runWs(p, "client", "remove", "lab")
	if status != 0 || !strings.Contains(stdout, "the secrets' recipients hold no host key of lab") {
		t.Fatalf("remove of no recipient: status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}
	if len(mac.Fnoxed()) != 0 || len(mac.Landed()) != 0 {
		t.Errorf("remove of no recipient ran fnox %q or landed %+v", mac.Fnoxed(), mac.Landed())
	}
	if callAt(mac.Calls, "keys/9") < 0 || callAt(mac.Calls, "keys/4") >= 0 {
		t.Errorf("remove must delete the deploy key of lab's key alone, whatever its title: %q", mac.Calls)
	}
}

func TestRemoveKeepsTheServerWhereTheSecretsFail(t *testing.T) {
	p, mac := removeSecretsFixture(heldBy(t, labKey))
	mac.Keys = []secrets.DeployKey{{ID: 5, Title: "ws-a", Key: labKey, ReadOnly: true}}
	mac.Results[secrets.GH+" pr checks "+secretstest.Pull+" --json name,bucket"] = effecttest.Result{
		Out: []byte(`[{"name":"values","bucket":"fail"},{"name":"gitleaks","bucket":"pass"}]`)}
	status, stdout, stderr := runWs(p, "client", "remove", "lab")
	if status != 1 || !strings.Contains(stderr, secretstest.Pull) || !strings.Contains(stderr, "lab stays on the Mac") {
		t.Errorf("a failed landing: status = %d, stderr = %q; want 1, the link and the server kept", status, stderr)
	}
	if strings.Contains(stdout, "removed") {
		t.Errorf("stdout = %q; want no removal", stdout)
	}
	for _, part := range []string{"ssh-keygen -R lab", "rm -rf /etc/ws/servers/lab", " -X DELETE "} {
		if callAt(mac.Calls, part) >= 0 {
			t.Errorf("a failed landing ran %q", part)
		}
	}
	if _, ok := mac.Machine.Files["/etc/ws/servers/lab/values.yml"]; !ok {
		t.Error("the server's file is gone")
	}
}

func TestRemoveWithoutTheMacsFnoxKeyLeavesItByHand(t *testing.T) {
	p, m := removeFixture()
	m.Results[bootstrapLine] = effecttest.Result{}
	status, stdout, stderr := runWs(p, "client", "remove", "lab")
	want := "removed lab\nBy hand: this Mac has no fnox key, " + fnoxKeyFile + ": from a Mac that has one, drop " +
		"lab's host key, ssh-ed25519 BBBB, from the secrets' recipients and deploy keys\n"
	if status != 0 || stdout != want {
		t.Errorf("status = %d, stdout = %q, stderr = %q; want 0 and %q", status, stdout, stderr, want)
	}
	if cloned(m.Calls) {
		t.Errorf("remove without fnox's key cloned the secrets: %q", m.Calls)
	}
}

func TestRemoveWithNoKeyInKnownHostsLeavesTheSecretsByHand(t *testing.T) {
	p, mac := removeSecretsFixture(heldBy(t, labKey))
	mac.Results[findLab] = effecttest.Result{Err: exec.Command("sh", "-c", "exit 1").Run()}
	status, stdout, stderr := runWs(p, "client", "remove", "lab")
	want := "removed lab\nBy hand: known_hosts holds no host key of lab: where add ran for it, drop its " +
		"hostname's recipients and deploy key from the secrets\n"
	if status != 0 || stdout != want {
		t.Errorf("status = %d, stdout = %q, stderr = %q; want 0 and %q", status, stdout, stderr, want)
	}
	if cloned(mac.Calls) || callAt(mac.Calls, secrets.GH) >= 0 {
		t.Errorf("remove of a server with no key in known_hosts touched the secrets: %q", mac.Calls)
	}
}

func TestRemoveOutOfReachOfTheSecretsSaysHowToGoOn(t *testing.T) {
	p, mac := removeSecretsFixture(heldBy(t, labKey))
	mac.Results[secrets.GH+" api repos/o/n --jq .private"] = effecttest.Result{Err: errors.New("exit status 1")}
	status, stdout, stderr := runWs(p, "client", "remove", "lab")
	for _, part := range []string{"by git, gh and the Nano", "--skip-secrets", "lab stays on the Mac: run remove again"} {
		if status != 1 || !strings.Contains(stderr, part) {
			t.Errorf("remove offline: status = %d, stderr = %q; want 1 and %q", status, stderr, part)
		}
	}
	if strings.Contains(stdout, "removed") || callAt(mac.Calls, "ssh-keygen -R lab") >= 0 {
		t.Errorf("remove offline went on: stdout = %q, calls %q", stdout, mac.Calls)
	}
}

func TestRemoveBySkipSecretsLeavesTheKeyByHand(t *testing.T) {
	p, mac := removeSecretsFixture(heldBy(t, labKey))
	status, stdout, stderr := runWs(p, "client", "remove", "lab", "--skip-secrets")
	want := "removed lab\nBy hand: remove skipped the secrets: drop lab's host key, " + labKey +
		", from the secrets' recipients and deploy keys\n"
	if status != 0 || stdout != want {
		t.Errorf("status = %d, stdout = %q, stderr = %q; want 0 and %q", status, stdout, stderr, want)
	}
	if cloned(mac.Calls) || callAt(mac.Calls, secrets.GH) >= 0 || len(mac.Fnoxed()) != 0 {
		t.Errorf("remove --skip-secrets touched the secrets: %q", mac.Calls)
	}
	if callAt(mac.Calls, "rm -rf /etc/ws/servers/lab") < 0 || callAt(mac.Calls, "ssh-keygen -R lab") < 0 {
		t.Errorf("remove --skip-secrets left lab on the Mac: %q", mac.Calls)
	}
}
