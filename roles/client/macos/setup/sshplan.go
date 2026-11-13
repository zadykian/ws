package setup

import (
	"io/fs"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/step"
)

// The modes of ssh's files.
const (
	sshDirMode  fs.FileMode = 0o700
	sshFileMode fs.FileMode = 0o600
)

// includeLine is the line that puts config.d/ws first in ~/.ssh/config.
const includeLine = "Include config.d/ws"

// sshFacts is what the ssh step read of the Mac.
type sshFacts struct {
	Home string
	// Dirs plans ~/.ssh and ~/.ssh/config.d, and DirLink is the last of them that is a link.
	Dirs       []step.Change
	DirLink    DirLink
	Ws         Found // ~/.ssh/config.d/ws
	UserConfig Found // ~/.ssh/config, whose text is the file a link points to
	// IncludeLine is the line of its Include of config.d/ws, or 0.
	IncludeLine int
	// Config plans ~/.ssh/config, or Stop says why a run can't. Include is how it stands.
	Config  []step.Change
	Stop    error
	Include step.Item
}

// sshChanges returns the changes that bring ~/.ssh to the new text of config.d/ws and to an
// Include of it first in ~/.ssh/config.
func sshChanges(f sshFacts, text []byte) []step.Change {
	cs := append([]step.Change(nil), f.Dirs...)
	dir := f.Home + "/.ssh"
	cs = append(cs, File{Path: dir + "/config.d/ws", Text: text, Mode: sshFileMode}.Change(f.Ws)...)
	return append(cs, f.Config...)
}

// sshDirs is ~/.ssh and its config.d. A run sets the mode of config.d, which holds the hosts.
func sshDirs(m effect.Machine, home string) Dirs {
	return Dirs{
		{M: m, Path: home + "/.ssh", Mode: sshDirMode},
		{M: m, Path: home + "/.ssh/config.d", Mode: sshDirMode, Fix: true},
	}
}
