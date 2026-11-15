package ssh

import (
	"context"
	"errors"
	"github.com/zadykian/ws/engine/core/resource/file"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/roles/internal/sshcfg"
)

// read reads ~/.ssh as the directory stands. Its Stop is where ~/.ssh/config can't take the Include.
func (s SSH) read(ctx context.Context) (sshFacts, error) {
	f := sshFacts{M: s.M, Out: s.Out, Home: s.Home}
	var err error
	if f.Dirs, _, err = sshDirs(s.M, s.Out, s.Home).Plan(); err != nil {
		return f, err
	}
	lines := s.configLines()
	found, link, holds, err := lines.Read(ctx)
	if err != nil && !errors.Is(err, file.ErrMisplaced) {
		return f, err
	}
	f.UserConfig = found
	if found.Exists {
		_, f.IncludeLine = sshcfg.IncludeAt(string(found.Text), sshcfg.File)
	}
	f.Include, _ = lines.Item(found, link, holds, err)
	f.Config, f.Stop = lines.Changes(found, link, holds, err)
	return f, nil
}

// configLines is the Include of /etc/ws/ssh_config at the top of ~/.ssh/config. An Include below a setting
// is out of place: ssh would take that setting first.
func (s SSH) configLines() file.UserLines {
	return file.UserLines{
		M: s.M, Out: s.Out, Path: s.Home + "/.ssh/config", Lines: includeLine + "\n", Shown: includeLine, Top: true,
		Mode: sshFileMode, Now: s.Now, Has: s.included,
	}
}

// included says whether the text of ~/.ssh/config has the Include of /etc/ws/ssh_config first.
func (s SSH) included(_ context.Context, _ effect.Machine, path string, text []byte) (bool, error) {
	place, line := sshcfg.IncludeAt(string(text), sshcfg.File)
	if place == sshcfg.Below {
		return false, file.Misplaced("line %d of %s is the Include of /etc/ws/ssh_config, below other settings, which ssh "+
			"would take first: move it to the top", line, path)
	}
	return place == sshcfg.First, nil
}
