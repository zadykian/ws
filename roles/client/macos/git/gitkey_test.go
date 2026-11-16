package git_test

import (
	"context"
	"io/fs"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/git"
	"github.com/zadykian/ws/roles/client/macos/internal/mactest"
	"github.com/zadykian/ws/roles/internal/sshcfg"
)

// copyKey makes the agent hold the key and the step name its copy, and returns the copy's path.
func copyKey(m *effecttest.Machine, g *git.Git) string {
	data := g.Home + "/" + sshcfg.SecretiveData
	socket := data + "/socket.ssh"
	m.Files[socket], m.Modes[socket] = nil, fs.ModeSocket
	m.Results["env SSH_AUTH_SOCK="+socket+" ssh-add -L"] = effecttest.Result{Out: []byte(mactest.AgentGit)}
	pub := sshcfg.AgentKeys(mactest.AgentGit)[0].Pub()
	g.Conf.GitSigningPubKey = pub
	return pub
}

func TestGitMakesEtcWsOnceWhereItKeepsACopy(t *testing.T) {
	m, g := gitReady()
	delete(m.Dirs, "/etc/ws")
	pub := copyKey(m, g)
	p, err := g.Step().Plan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var made int
	for _, c := range p.Changes {
		if strings.HasPrefix(c.String(), "create directory /etc/ws,") {
			made++
		}
	}
	if made != 1 {
		t.Errorf("Plan = %v; want /etc/ws made once", p.Changes)
	}
	if err := g.Step().Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	if m.Files[pub] == nil || m.Files["/etc/ws/allowed_signers"] == nil {
		t.Errorf("Apply left no copy or no signers: %v", m.Files)
	}
	reports, err := step.Read(context.Background(), []step.Step{g.Step()})
	if err != nil {
		t.Fatal(err)
	}
	made = 0
	for _, it := range reports[0].Items {
		if it.Name == "/etc/ws" {
			made++
		}
	}
	if made != 1 {
		t.Errorf("the report holds /etc/ws %d times, want once", made)
	}
}

func TestGitSignsWithTheCopyOfTheAgentsKey(t *testing.T) {
	m, g := gitReady()
	home := g.Home
	data := home + "/" + sshcfg.SecretiveData
	line := mactest.AgentGit
	socket := data + "/socket.ssh"
	m.Files[socket], m.Modes[socket] = nil, fs.ModeSocket
	m.Results["env SSH_AUTH_SOCK="+socket+" ssh-add -L"] = effecttest.Result{Out: []byte(line)}
	key := sshcfg.AgentKeys(line)[0]
	pub := key.Pub()
	// A values.yml from before macOS 27, whose .pub ws may not read.
	in := data + "/PublicKeys/" + key.Name
	m.Root[in] = m.Root[socket]
	g.Conf.GitSigningPubKey = in
	p, err := g.Step().Plan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var copied, config bool
	for _, c := range p.Changes {
		// CI's mac job compares the run's lines in order: the copy comes before the files.
		if strings.Contains(c.String(), "secretive-ssh-keygen") && !copied {
			t.Errorf("Plan = %v; want ws's copy before the wrapper", p.Changes)
		}
		copied = copied || strings.HasPrefix(c.String(), "create "+pub+"\n")
		config = config || strings.Contains(c.String(), `signingkey = "`+pub+`"`)
	}
	if !copied || !config {
		t.Errorf("Plan = %v; want ws's copy, named as git's key", p.Changes)
	}
}
