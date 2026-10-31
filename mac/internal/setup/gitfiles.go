package setup

import (
	"fmt"
	"io/fs"
	"strings"

	"github.com/zadykian/ws/mac/internal/gitcfg"
	"github.com/zadykian/ws/mac/internal/sshcfg"
	"github.com/zadykian/ws/mac/internal/step"
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
	secretive := g.Home + "/" + sshcfg.SecretiveData
	type file struct {
		path, text string
		mode       fs.FileMode
	}
	var list []file
	if strings.HasPrefix(key.Pub, secretive+"/PublicKeys/") && strings.HasSuffix(key.Pub, ".pub") {
		in.Program = dir + "/secretive-ssh-keygen"
		list = append(list, file{in.Program, gitcfg.Wrapper(secretive), 0o755})
	}
	if key.Public != "" {
		list = append(list, file{dir + "/allowed_signers", gitcfg.AllowedSigners(g.Conf.GitEmail, key.Public), 0o644})
	} else {
		cs = append(cs, Pending("would write "+dir+"/allowed_signers once the key is made"))
	}
	list = append(list, file{dir + "/gitconfig", gitcfg.Config(in), 0o644})
	for _, f := range list {
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
