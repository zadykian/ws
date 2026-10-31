package setup

import (
	"fmt"
	"io/fs"

	"github.com/zadykian/ws/mac/internal/gitcfg"
	"github.com/zadykian/ws/mac/internal/step"
)

// put returns the change that makes the user's file hold text, with mode, or nil.
func (g *Git) put(file string, text string, mode fs.FileMode) (step.Change, error) {
	info, err := g.M.Stat(file)
	if missing, err := notExist(err); err != nil {
		return nil, err
	} else if missing {
		return Write{file, nil, []byte(text), false, mode}, nil
	}
	old, err := g.M.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", file, err)
	}
	switch {
	case string(old) != text:
		return Write{file, old, []byte(text), true, mode}, nil
	case info.Mode.Perm() != mode:
		return SetMode{file, mode}, nil
	}
	return nil, nil //nolint:nilnil // nothing to change
}

// files plans ~/.config/ws and the files in it: the script for a key of Secretive's, the allowed
// signers once the key is there, and gitconfig. Under --check, before the key is made, there are no
// allowed signers.
func (g *Git) files(key gitKey, gh string) ([]step.Change, error) {
	dir := g.Home + "/.config/ws"
	var cs []step.Change
	for _, d := range []struct {
		path string
		mode fs.FileMode
	}{{g.Home + "/.config", 0o755}, {dir, 0o700}} {
		if _, err := g.M.Stat(d.path); err != nil {
			if missing, err := notExist(err); err != nil || !missing {
				return nil, err
			}
			cs = append(cs, MakeDir{d.path, d.mode})
		}
	}
	in := gitcfg.Inputs{Name: g.Conf.GitName, Email: g.Conf.GitEmail, Key: key.Pub, Gh: gh, Dir: dir}
	c, err := g.put(dir+"/gitconfig", gitcfg.Config(in), 0o644)
	if c != nil {
		cs = append(cs, c)
	}
	return cs, err
}
