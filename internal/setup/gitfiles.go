package setup

import (
	"fmt"
	"io/fs"

	"github.com/zadykian/ws/internal/gitcfg"
	"github.com/zadykian/ws/internal/sshcfg"
	"github.com/zadykian/ws/step"
)

// Pending is a change that waits on another, such as the key: the line says on what.
type Pending string

func (p Pending) String() string { return string(p) }

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

// gitFile is a file of ~/.config/ws.
type gitFile struct {
	path, text string
	mode       fs.FileMode
}

// list returns the files of ~/.config/ws: the script for a key of Secretive's, the allowed signers
// once the key is there, and gitconfig.
func (g *Git) list(key gitKey, gh string) []gitFile {
	dir := g.Home + "/.config/ws"
	in := gitcfg.Inputs{Name: g.Conf.GitName, Email: g.Conf.GitEmail, Key: key.Pub, Gh: gh, Dir: dir}
	var list []gitFile
	if sshcfg.SecretiveKey(key.Pub, g.Home) {
		in.Program = dir + "/secretive-ssh-keygen"
		list = append(list, gitFile{in.Program, gitcfg.Wrapper(g.Home + "/" + sshcfg.SecretiveData), 0o755})
	}
	if key.Public != "" {
		list = append(list, gitFile{dir + "/allowed_signers", gitcfg.AllowedSigners(g.Conf.GitEmail, key.Public), 0o644})
	}
	return append(list, gitFile{dir + "/gitconfig", gitcfg.Config(in), 0o644})
}

// files plans ~/.config/ws and the files in it. Under --check, before the key is made, there are no
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
	cs = append(cs, PlanCopy(g.M, key.Pub, key.Text, g.Home)...)
	if key.Public == "" {
		cs = append(cs, Pending("would write "+dir+"/allowed_signers once the key is made"))
	}
	for _, f := range g.list(key, gh) {
		c, err := g.put(f.path, f.text, f.mode)
		if err != nil {
			return nil, err
		}
		if c != nil {
			cs = append(cs, c)
		}
	}
	return cs, nil
}
