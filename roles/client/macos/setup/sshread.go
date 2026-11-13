package setup

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/roles/client/macos/internal/userfile"
	"github.com/zadykian/ws/roles/internal/sshcfg"
)

// read reads ~/.ssh as the directory stands, and stops at a config.d/ws that is not ws's. Its Stop is
// where ~/.ssh/config can't take the Include.
func (s SSH) read(ctx context.Context) (sshFacts, error) {
	dir := s.Home + "/.ssh"
	f := sshFacts{Home: s.Home}
	var err error
	if f.Dirs, f.DirLink, err = sshDirs(s.M, s.Home).Plan(); err != nil {
		return f, err
	}
	var ours bool
	if f.Ws, ours, err = s.wsFile(nil).Read(); err != nil {
		return f, err
	}
	if !ours {
		return f, fmt.Errorf("%s is not ws client macos apply's: move it away, as the script writes its own there", dir+"/config.d/ws")
	}
	lines := s.configLines()
	found, link, holds, err := lines.Read(ctx)
	if err != nil && !errors.Is(err, userfile.ErrMisplaced) {
		return f, err
	}
	f.UserConfig = found
	if found.Exists {
		_, f.IncludeLine = sshcfg.IncludeAt(string(found.Text), "config.d/ws", s.Home)
	}
	f.Include, _ = lines.Item(found, link, holds, err)
	f.Config, f.Stop = lines.Changes(found, link, holds, err)
	return f, nil
}

// configLines is the Include of config.d/ws at the top of ~/.ssh/config. An Include below a setting
// is out of place: ssh would take that setting first.
func (s SSH) configLines() userfile.Lines {
	return userfile.Lines{
		M: s.M, Path: s.Home + "/.ssh/config", Lines: includeLine + "\n", Shown: includeLine, Top: true,
		Mode: sshFileMode, Now: s.Now, Has: s.included,
	}
}

// included says whether the text of ~/.ssh/config has the Include of config.d/ws first.
func (s SSH) included(_ context.Context, _ effect.Machine, file string, text []byte) (bool, error) {
	place, line := sshcfg.IncludeAt(string(text), "config.d/ws", s.Home)
	if place == sshcfg.Below {
		return false, userfile.Misplaced("line %d of %s is the Include of config.d/ws, below other settings, which ssh "+
			"would take first: move it to the top", line, file)
	}
	return place == sshcfg.First, nil
}

func firstLine(text []byte) string {
	line, _, _ := bytes.Cut(text, []byte("\n"))
	return string(line)
}

// wsFile is config.d/ws, which holds text.
func (s SSH) wsFile(text []byte) userfile.File {
	return userfile.File{M: s.M, Path: s.Home + "/.ssh/config.d/ws", Text: text, Mode: sshFileMode, Own: sshOwn}
}

// sshOwn says whether config.d/ws's first line is ws's.
func sshOwn(have []byte) bool { return strings.HasPrefix(firstLine(have), sshcfg.Header) }
