package facts

import (
	"errors"
	"io/fs"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/internal/sshcfg"
	"github.com/zadykian/ws/macos/internal/mactest"
	"github.com/zadykian/ws/macos/values"
)

// env is the process's environment of a user v, with no WS_VALUES.
func env(k string) string {
	return map[string]string{"USER": "v", "HOME": "/Users/v"}[k]
}

// mac is a machine with values.yml, which names a file key for login and signing, and a file for
// each server, its text by name. ssh takes each server's block, and mise has no gh yet.
func mac(servers map[string]string) *effecttest.Machine {
	m := effecttest.NewMachine()
	m.Files["/etc/ws/values.yml"] = []byte(values.Text("WS_IDENTITY_PUB_KEY", "~/.ssh/id.pub",
		"WS_GIT_SIGNING_PUB_KEY", "~/.ssh/id.pub"))
	m.Files["/Users/v/.ssh/id.pub"] = []byte("ssh-ed25519 AAAA me\n")
	m.Results["ssh-keygen -l -f /Users/v/.ssh/id.pub"] = effecttest.Result{}
	m.Results["mktemp -d"] = effecttest.Result{Out: []byte("/tmp/x\n")}
	m.Results["rm -rf /tmp/x"] = effecttest.Result{}
	m.Dirs["/etc/ws/servers"] = 0o755
	for name, text := range servers {
		m.Dirs["/etc/ws/servers/"+name] = 0o755
		m.Files["/etc/ws/servers/"+name+"/values.yml"] = []byte(text)
		for _, file := range []string{"new", "whole"} {
			m.Results["ssh -T -G -F /tmp/x/"+file+" "+name] = effecttest.Result{}
		}
	}
	return m
}

func TestReadListsEachServer(t *testing.T) {
	m := mac(map[string]string{
		"ws":  values.Text("WS_SLOT", "0"),
		"lab": values.Text("WS_SLOT", "1", "WS_TARGET", "cloud", "WS_REMOTE", "198.51.100.7", "WS_REMOTE_PORT", "22"),
	})
	f, err := Read(t.Context(), m, env, "/p")
	if err != nil {
		t.Fatal(err)
	}
	if f.User != "v" || f.Home != "/Users/v" || len(f.Servers) != 2 {
		t.Fatalf("facts = %+v, want v, /Users/v and two servers", f)
	}
	lab, ws := f.Servers[0], f.Servers[1]
	if lab.Name != "lab" || lab.Slot != "1" || lab.Remote != "198.51.100.7" || ws.Name != "ws" {
		t.Errorf("servers = %+v, want lab at slot 1, then ws", f.Servers)
	}
	if want := "/Users/v/Library/Application Support/com.mitchellh.ghostty/config.ghostty"; f.Ghostty.Config != want {
		t.Errorf("Ghostty's config = %q, want %q, as none is there", f.Ghostty.Config, want)
	}
	if lab.IdentityPubKey != "~/.ssh/id.pub" {
		t.Errorf("lab's IdentityPubKey = %q, want the Mac's key joined in", lab.IdentityPubKey)
	}
}

func TestReadRefusesAClash(t *testing.T) {
	m := mac(map[string]string{"ws": values.Text("WS_SLOT", "1"), "lab": values.Text("WS_SLOT", "1")})
	if _, err := Read(t.Context(), m, env, "/p"); err == nil || !strings.Contains(err.Error(), "WS_SLOT=1") {
		t.Errorf("read = %v, want the clash of the slot", err)
	}
}

func TestReadRefusesABadValue(t *testing.T) {
	m := mac(map[string]string{"ws": values.Text("WS_SLOT", "99")})
	if _, err := Read(t.Context(), m, env, "/p"); err == nil || !strings.Contains(err.Error(), "server ws") {
		t.Errorf("read = %v, want server ws refused", err)
	}
}

// TestReadRefusesABadMacFileOnce checks that values.yml is read before any server. Two servers
// and a bad values.yml give one refusal, in that file's words.
func TestReadRefusesABadMacFileOnce(t *testing.T) {
	m := mac(map[string]string{"ws": values.Text("WS_SLOT", "0"), "lab": values.Text("WS_SLOT", "1")})
	m.Files["/etc/ws/values.yml"] = []byte(values.Text("WS_SLOT", "3"))
	_, err := Read(t.Context(), m, env, "/p")
	if err == nil || strings.Count(err.Error(), "/etc/ws/values.yml") != 1 || strings.Contains(err.Error(), "server ") ||
		strings.Contains(err.Error(), "fix its file") {
		t.Errorf("read = %v, want one refusal of values.yml, naming no server", err)
	}
	if _, err := Servers(m, "/etc/ws/values.yml"); err == nil || strings.Contains(err.Error(), "fix its file") {
		t.Errorf("servers = %v, want values.yml's refusal alone", err)
	}
}

func TestReadWithoutServers(t *testing.T) {
	m := mac(nil)
	delete(m.Dirs, "/etc/ws/servers")
	f, err := Read(t.Context(), m, env, "/p")
	if err != nil || f.User != "v" || len(f.Servers) != 0 || strings.Contains(f.SSH.Config, "Host ") {
		t.Errorf("read = %+v, %v; want the user, no server and no host", f, err)
	}
	for _, c := range m.Calls {
		if strings.HasPrefix(c, "run ssh -T -G") {
			t.Errorf("read ran %q, want no check of no host", c)
		}
	}
}

func TestReadNeedsValuesYml(t *testing.T) {
	m := mac(nil)
	delete(m.Files, "/etc/ws/values.yml")
	if _, err := Read(t.Context(), m, env, "/p"); err == nil || !strings.Contains(err.Error(), "values.yml is missing") {
		t.Errorf("read = %v, want values.yml asked for", err)
	}
}

// TestSSHFactsReadNoKey pins that ssh's text names seward's login key whatever values.yml names:
// the key's .pub is made by a unit, after the facts are read.
func TestSSHFactsReadNoKey(t *testing.T) {
	m := mac(map[string]string{"ws": values.Text("WS_SLOT", "0")})
	m.Files["/etc/ws/values.yml"] = []byte(values.Text("WS_IDENTITY_PUB_KEY", "~/.ssh/other.pub",
		"WS_GIT_SIGNING_PUB_KEY", "~/.ssh/id.pub"))
	f, err := Read(t.Context(), m, env, "/p")
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"IdentityFile ~/.ssh/ws-auth.pub\n", "IdentityAgent \"~/.ssh/seward.sock\"\n"} {
		if !strings.Contains(f.SSH.Config, want) {
			t.Errorf("ssh_config lacks %q:\n%s", want, f.SSH.Config)
		}
	}
}

func TestSSHFactsMatchSshcfg(t *testing.T) {
	m := mac(map[string]string{"ws": values.Text("WS_SLOT", "0")})
	f, err := Read(t.Context(), m, env, "/p")
	if err != nil {
		t.Fatal(err)
	}
	want := sshcfg.Config(sshcfg.Inputs{
		Hosts:       []sshcfg.Host{{Name: "ws", Addr: f.Servers[0].TunnelSlot().Server()}},
		Patterns:    "UseKeychain",
		IdentityPub: "~/.ssh/ws-auth.pub", IdentityAgent: "~/.ssh/seward.sock",
	})
	if f.SSH.Config != want {
		t.Errorf("ssh_config = %q, want sshcfg's %q", f.SSH.Config, want)
	}
	if got := string(m.Files["/tmp/x/new"]); got != want {
		t.Errorf("ssh checked %q, want the text", got)
	}
}

func TestSSHFactsRefuseAnOwnHost(t *testing.T) {
	m := mac(map[string]string{"ws": values.Text("WS_SLOT", "0")})
	m.Files["/Users/v/.ssh/config"] = []byte("Host ws\n    User other\n")
	if _, err := Read(t.Context(), m, env, "/p"); err == nil || !strings.Contains(err.Error(), "line 1 of ~/.ssh/config") {
		t.Errorf("read = %v, want the own Host refused", err)
	}
}

func TestSSHFactsRefuseWhatSSHRefuses(t *testing.T) {
	m := mac(map[string]string{"ws": values.Text("WS_SLOT", "0")})
	m.Results["ssh -T -G -F /tmp/x/new ws"] = effecttest.Result{Err: errors.New("exit status 255: bad line")}
	if _, err := Read(t.Context(), m, env, "/p"); err == nil || !strings.Contains(err.Error(), "ssh refuses the config") {
		t.Errorf("read = %v, want ssh's refusal", err)
	}
}

func TestSSHFactsPredictCldsTabs(t *testing.T) {
	m := mac(map[string]string{"ws": values.Text("WS_SLOT", "0")})
	m.Dirs["/Applications/Ghostty.app"] = 0o755
	f, err := Read(t.Context(), m, env, "/p")
	if err != nil || !strings.Contains(f.SSH.Config, "restore tabs") {
		t.Errorf("read = %q, %v; want cld's watcher, as cld's unit installs cld", f.SSH.Config, err)
	}
}

func TestGitFactsNameUsrLocalGh(t *testing.T) {
	f, err := Read(t.Context(), mac(nil), env, "/p")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(f.Git.Config, "!/usr/local/bin/gh auth git-credential") || f.Git.Keygen != "" {
		t.Errorf("gitconfig = %q, keygen %q; want the credential command at /usr/local/bin/gh, and no script for a file key",
			f.Git.Config, f.Git.Keygen)
	}
	if !strings.Contains(f.Git.Signers, "ssh-ed25519 AAAA") {
		t.Errorf("allowed_signers = %q, want the key", f.Git.Signers)
	}
}

func TestGitFactsWithSecretive(t *testing.T) {
	m := mac(nil)
	socket := "/Users/v/" + sshcfg.SecretiveData + "/socket.ssh"
	m.Files[socket], m.Modes[socket] = nil, fs.ModeSocket
	m.Results["env SSH_AUTH_SOCK="+socket+" ssh-add -L"] = effecttest.Result{Out: []byte(mactest.AgentGit)}
	pub := sshcfg.AgentKeys(mactest.AgentGit)[0].Pub()
	m.Files["/etc/ws/values.yml"] = []byte(values.Text("WS_IDENTITY_PUB_KEY", "~/.ssh/id.pub", "WS_GIT_SIGNING_PUB_KEY", pub))
	f, err := Read(t.Context(), m, env, "/p")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(f.Git.Keygen, "SSH_AUTH_SOCK='/Users/v/"+sshcfg.SecretiveData+"/socket.ssh'") ||
		!strings.Contains(f.Git.Config, "/etc/ws/secretive-ssh-keygen") {
		t.Errorf("keygen = %q, gitconfig = %q; want the script, named as git's program", f.Git.Keygen, f.Git.Config)
	}
}

func TestGitFactsGhFromMise(t *testing.T) {
	m := mac(nil)
	if f, err := Read(t.Context(), m, env, "/p"); err != nil || f.Git.GhFrom != "" {
		t.Errorf("read = %q, %v; want no gh, and no error, where mise has none", f.Git.GhFrom, err)
	}
	gh := "/Users/v/.local/share/mise/installs/gh/2.104.0/gh_2.104.0_macOS_arm64/bin/gh"
	m.Results["mise -C /opt/ws/macos which gh"] = effecttest.Result{Out: []byte(gh + "\n")}
	if f, err := Read(t.Context(), m, env, "/p"); err != nil || f.Git.GhFrom != gh {
		t.Errorf("read = %q, %v; want mise's gh", f.Git.GhFrom, err)
	}
}

func TestSecretsFactsFromMise(t *testing.T) {
	m := mac(nil)
	if f, err := Read(t.Context(), m, env, "/p"); err != nil || f.Secrets != (Secrets{}) {
		t.Errorf("read = %+v, %v; want no tool, and no error, where mise has none", f.Secrets, err)
	}
	fnox := "/Users/v/.local/share/mise/installs/github-jdx-fnox/1.50.0/fnox"
	plugin := "/Users/v/.local/share/mise/installs/age-plugin-yubikey/0.5.1/age-plugin-yubikey"
	m.Results["mise -C /opt/ws/macos which fnox"] = effecttest.Result{Out: []byte(fnox + "\n")}
	m.Results["mise -C /opt/ws/macos which age-plugin-yubikey"] = effecttest.Result{Out: []byte(plugin + "\n")}
	f, err := Read(t.Context(), m, env, "/p")
	if want := (Secrets{FnoxFrom: fnox, PluginFrom: plugin}); err != nil || f.Secrets != want {
		t.Errorf("read = %+v, %v; want mise's %+v", f.Secrets, err, want)
	}
}

func TestSSHFactsSewardFromMise(t *testing.T) {
	m := mac(nil)
	if f, err := Read(t.Context(), m, env, "/p"); err != nil || f.SSH.SewardFrom != "" {
		t.Errorf("read = %q, %v; want no seward, and no error, where mise has none", f.SSH.SewardFrom, err)
	}
	seward := "/Users/v/.local/share/mise/installs/github-zadykian-seward/0.1.0/seward"
	m.Results["mise -C /opt/ws/macos which seward"] = effecttest.Result{Out: []byte(seward + "\n")}
	if f, err := Read(t.Context(), m, env, "/p"); err != nil || f.SSH.SewardFrom != seward {
		t.Errorf("read = %q, %v; want mise's seward", f.SSH.SewardFrom, err)
	}
}

func TestVMFactIsTheHypervisorBit(t *testing.T) {
	for out, want := range map[string]bool{"1\n": true, "0\n": false, "": false} {
		m := mac(nil)
		m.Results["sysctl -n kern.hv_vmm_present"] = effecttest.Result{Out: []byte(out)}
		if f, err := Read(t.Context(), m, env, "/p"); err != nil || f.VM != want {
			t.Errorf("sysctl prints %q: VM = %v, %v; want %v", out, f.VM, err, want)
		}
	}
	if f, err := Read(t.Context(), mac(nil), env, "/p"); err != nil || f.VM {
		t.Errorf("with no sysctl answer: VM = %v, %v; want false and no error", f.VM, err)
	}
}

func TestReadNeedsTheUser(t *testing.T) {
	if _, err := Read(t.Context(), effecttest.NewMachine(), func(string) string { return "" }, "/p"); err == nil {
		t.Error("read with no USER and HOME = nil, want a refusal")
	}
}
