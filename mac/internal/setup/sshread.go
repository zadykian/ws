package setup

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/zadykian/ws/mac/internal/effect"
	"github.com/zadykian/ws/mac/internal/sshcfg"
)

func (s SSH) stat(p string) (path, effect.Info, error) {
	info, err := s.M.Stat(p)
	if errors.Is(err, fs.ErrNotExist) {
		return path{}, effect.Info{}, nil
	}
	return path{Exists: true, Mode: info.Mode}, info, err
}

// read reads ~/.ssh as it is, and stops at a config.d/ws that is not ws's and at an Include below
// a setting.
func (s SSH) read() (sshFacts, error) {
	dir := s.Home + "/.ssh"
	f := sshFacts{Home: s.Home, Stamp: s.Now().Format("20060102150405")}
	var err error
	if f.Dir, _, err = s.stat(dir); err != nil {
		return f, err
	}
	var info effect.Info
	if f.ConfigDir, info, err = s.stat(dir + "/config.d"); err != nil {
		return f, err
	}
	f.ConfigDirLink = info.Link
	ws := dir + "/config.d/ws"
	if f.Ws, _, err = s.stat(ws); err != nil {
		return f, err
	}
	if f.Ws.Exists {
		if f.Ws.Text, err = s.M.ReadFile(ws); err != nil && !f.Ws.Mode.IsRegular() {
			err = nil // a link to nowhere, or a directory, is not ws's either
		}
		if err != nil {
			return f, err
		}
		if !strings.HasPrefix(firstLine(f.Ws.Text), sshcfg.Header) {
			return f, fmt.Errorf("%s is not mac/setup.sh's: move it away, as the script writes its own there", ws)
		}
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

func firstLine(text []byte) string {
	line, _, _ := bytes.Cut(text, []byte("\n"))
	return string(line)
}
