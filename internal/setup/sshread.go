package setup

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/internal/sshcfg"
)

func (s SSH) stat(p string) (path, effect.Info, error) {
	info, err := s.M.Stat(p)
	if errors.Is(err, fs.ErrNotExist) {
		return path{}, effect.Info{}, nil
	}
	return path{Exists: true, Mode: info.Mode}, info, err
}

// read reads ~/.ssh as the directory stands, and stops at a config.d/ws that is not ws's and at an
// Include below a setting.
func (s SSH) read() (sshFacts, error) {
	dir := s.Home + "/.ssh"
	f := sshFacts{Home: s.Home, Stamp: s.Now().Format("20060102150405")}
	var err error
	if f.Dirs, f.DirLink, err = sshDirs(s.M, s.Home).plan(); err != nil {
		return f, err
	}
	var ours bool
	if f.Ws, ours, err = s.wsFile(nil).read(); err != nil {
		return f, err
	}
	if !ours {
		return f, fmt.Errorf("%s is not ws mac apply's: move it away, as the script writes its own there", dir+"/config.d/ws")
	}
	user := dir + "/config"
	var link effect.Info
	if f.UserConfig, link, err = s.stat(user); err != nil {
		return f, err
	}
	if f.UserConfig.Exists {
		if f.UserConfig.Text, err = s.M.ReadFile(user); err != nil {
			return f, fmt.Errorf("cannot read %s: %w", user, err)
		}
		place, line := sshcfg.IncludeAt(string(f.UserConfig.Text), "config.d/ws", s.Home)
		switch {
		case place == sshcfg.Below:
			return f, fmt.Errorf("line %d of %s is the Include of config.d/ws, below other settings, which ssh "+
				"would take first: move it to the top", line, user)
		case place == sshcfg.Missing && link.Link != "":
			return f, fmt.Errorf("%s is a link to %s: put 'Include config.d/ws' at the top of that file", user, link.Link)
		}
		f.UserConfigNeedsInclude, f.IncludeLine = place == sshcfg.Missing, line
	}
	return f, nil
}

// configLines is the Include of config.d/ws at the top of ~/.ssh/config. An Include below a setting
// is out of place: ssh would take that setting first.
func (s SSH) configLines() userLines {
	return userLines{
		M: s.M, Path: s.Home + "/.ssh/config", Lines: includeLine + "\n", Shown: includeLine, Top: true,
		Mode: sshFileMode, Now: s.Now, Has: s.included,
	}
}

// included says whether the text of ~/.ssh/config has the Include of config.d/ws first.
func (s SSH) included(_ context.Context, _ effect.Machine, file string, text []byte) (bool, error) {
	place, line := sshcfg.IncludeAt(string(text), "config.d/ws", s.Home)
	if place == sshcfg.Below {
		return false, misplaced("line %d of %s is the Include of config.d/ws, below other settings, which ssh "+
			"would take first: move it to the top", line, file)
	}
	return place == sshcfg.First, nil
}

func firstLine(text []byte) string {
	line, _, _ := bytes.Cut(text, []byte("\n"))
	return string(line)
}

// wsFile is config.d/ws, which holds text.
func (s SSH) wsFile(text []byte) userFile {
	return userFile{M: s.M, Path: s.Home + "/.ssh/config.d/ws", Text: text, Mode: sshFileMode, Own: sshOwn}
}

// sshOwn says whether config.d/ws's first line is ws's.
func sshOwn(have []byte) bool { return strings.HasPrefix(firstLine(have), sshcfg.Header) }
