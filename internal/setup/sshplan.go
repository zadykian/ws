package setup

import (
	"io/fs"

	"github.com/zadykian/ws/step"
)

// The modes of ssh's files.
const (
	sshDirMode  fs.FileMode = 0o700
	sshFileMode fs.FileMode = 0o600
)

// includeLine is the line that puts config.d/ws first in ~/.ssh/config.
const includeLine = "Include config.d/ws"

// path is what a step read of one path: whether it exists, its mode and its text.
type path struct {
	Exists bool
	Mode   fs.FileMode
	Text   []byte
}

// sshFacts is what the ssh step read of the Mac.
type sshFacts struct {
	Home       string
	Dir        path // ~/.ssh
	ConfigDir  path // ~/.ssh/config.d
	Ws         path // ~/.ssh/config.d/ws
	UserConfig path // ~/.ssh/config, whose text is the file a link points to
	// UserConfigNeedsInclude is whether it lacks the Include.
	UserConfigNeedsInclude bool
	// IncludeLine is the line of its Include of config.d/ws, or 0.
	IncludeLine int
	// ConfigDirLink is where ~/.ssh/config.d points, or empty.
	ConfigDirLink string
	// Stamp is the time suffix of the copy of the old ~/.ssh/config.
	Stamp string
}

// sshChanges returns the changes that bring ~/.ssh to the new text of config.d/ws and to an
// Include of it first in ~/.ssh/config.
func sshChanges(f sshFacts, text []byte) []step.Change {
	var cs []step.Change
	dir := f.Home + "/.ssh"
	if !f.Dir.Exists {
		cs = append(cs, MakeDir{dir, sshDirMode})
	}
	switch {
	case !f.ConfigDir.Exists:
		cs = append(cs, MakeDir{dir + "/config.d", sshDirMode})
	case f.ConfigDir.Mode.Perm() != sshDirMode:
		cs = append(cs, SetMode{dir + "/config.d", sshDirMode})
	}
	cs = append(cs, userFile{Path: dir + "/config.d/ws", Text: text, Mode: sshFileMode}.change(f.Ws)...)
	user := dir + "/config"
	switch {
	case !f.UserConfig.Exists:
		cs = append(cs, Write{user, nil, []byte(includeLine + "\n"), false, sshFileMode})
	case f.UserConfigNeedsInclude:
		cs = append(cs, Backup{user, user + "." + f.Stamp + "~"},
			Write{user, f.UserConfig.Text, userConfigText(f), true, f.UserConfig.Mode.Perm()})
	}
	return cs
}

// userConfigText returns the text of ~/.ssh/config that the run leaves: the Include and a blank
// line above the file's text where it lacks the Include.
func userConfigText(f sshFacts) []byte {
	if !f.UserConfig.Exists || !f.UserConfigNeedsInclude {
		return f.UserConfig.Text
	}
	return append([]byte(includeLine+"\n\n"), f.UserConfig.Text...)
}
