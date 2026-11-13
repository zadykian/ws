package setup_test

import (
	"context"
	"io/fs"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/client/macos/internal/change"
	"github.com/zadykian/ws/roles/client/macos/internal/mactest"
	"github.com/zadykian/ws/roles/internal/sshcfg"
)

func TestGitSignsWithTheCopyOfTheAgentsKey(t *testing.T) {
	m, g := gitReady()
	home := g.Home
	data := home + "/" + sshcfg.SecretiveData
	line := mactest.AgentGit
	socket := data + "/socket.ssh"
	m.Files[socket], m.Modes[socket] = nil, fs.ModeSocket
	m.Results["env SSH_AUTH_SOCK="+socket+" ssh-add -L"] = effecttest.Result{Out: []byte(line)}
	key := sshcfg.AgentKeys(line)[0]
	pub := sshcfg.Expand(key.Pub(), home)
	// A mac.conf from before macOS 27, whose .pub ws may not read.
	in := data + "/PublicKeys/" + key.Name
	m.Root[in] = m.Root[socket]
	g.Conf.GitSigningPubKey = in
	p, err := g.Plan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	var copied, config bool
	for _, c := range p.Changes {
		copied = copied || c == change.KeyCopy{Path: pub, Text: line}
		config = config || strings.Contains(c.String(), `signingkey = "`+pub+`"`)
	}
	if !copied || !config {
		t.Errorf("Plan = %v; want ws's copy, named as git's key", p.Changes)
	}
}
