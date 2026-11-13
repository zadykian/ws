package gh

import (
	"io/fs"
	"strings"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/internal/sshcfg"
)

const (
	confPathForTest = "/h/.config/ws/mac.conf"
	// pubText is the text of a public key file.
	pubText  = "ssh-ed25519 AAAAC3 me@mac\n"
	p256     = "ecdsa-sha2-nistp256 AAAA me\n"
	socket   = "/h/" + sshcfg.SecretiveData + "/socket.ssh"
	listKeys = "env SSH_AUTH_SOCK=" + socket + " ssh-add -L"
)

// agent gives the Mac Secretive's socket, whose agent lists the lines.
func agent(m *effecttest.Machine, lines ...string) {
	m.Files[socket], m.Modes[socket] = nil, fs.ModeSocket
	m.Results[listKeys] = effecttest.Result{Out: []byte(strings.Join(lines, ""))}
	m.Results["mkdir -p /h/.config/ws"] = effecttest.Result{}
	m.Dirs["/h/.config"], m.Dirs["/h/.config/ws"], m.Dirs["/h/"+sshcfg.KeyCopies] = 0o755, 0o700, 0o755
}

// copyOf is the path of ws's copy of the line's key.
func copyOf(line string) string { return sshcfg.Expand(sshcfg.AgentKeys(line)[0].Pub(), "/h") }
