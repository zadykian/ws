package setup

import (
	"context"
	"io/fs"

	"github.com/zadykian/ws/internal/gitcfg"
	"github.com/zadykian/ws/internal/sshcfg"
	"github.com/zadykian/ws/step"
)

// Pending is a change that waits on another, such as gh's install: the line says on what.
type Pending string

func (p Pending) String() string { return string(p) }

// list returns the files of ~/.config/ws: the script for a key of Secretive's, the allowed signers
// and gitconfig.
func (g *Git) list(key NamedKey, gh string) []userFile {
	dir := g.Home + "/.config/ws"
	in := gitcfg.Inputs{Name: g.Conf.GitName, Email: g.Conf.GitEmail, Key: key.File, Gh: gh, Dir: dir}
	file := func(path, text string, mode fs.FileMode) userFile {
		return userFile{M: g.M, Path: path, Text: []byte(text), Mode: mode}
	}
	var list []userFile
	if sshcfg.SecretiveKey(key.File, g.Home) {
		in.Program = dir + "/secretive-ssh-keygen"
		list = append(list, file(in.Program, gitcfg.Wrapper(g.Home+"/"+sshcfg.SecretiveData), 0o755))
	}
	list = append(list, file(dir+"/allowed_signers", gitcfg.AllowedSigners(g.Conf.GitEmail, key.Key.String()), 0o644))
	return append(list, file(dir+"/gitconfig", gitcfg.Config(in), 0o644))
}

// dirs is ws's directory, with the directory of ws's copies where git signs with a copy.
func (g *Git) dirs(pub, text string) userDirs {
	if copied(pub, text, g.Home) {
		return copyDirs(g.M, g.Home)
	}
	return wsDirs(g.M, g.Home)
}

// files plans ~/.config/ws and the files in it.
func (g *Git) files(key NamedKey, gh string) ([]step.Change, error) {
	cs, link, err := g.dirs(key.File, key.Text).plan()
	if err != nil {
		return nil, err
	}
	if copied(key.File, key.Text, g.Home) {
		cs = append(cs, copyChange(g.M, key.File, key.Text)...)
	}
	for _, f := range g.list(key, gh) {
		planned, err := f.plan()
		if err != nil {
			return nil, err
		}
		cs = append(cs, planned...)
	}
	link.warn(g.Out, len(cs), "ws's files go there")
	return cs, nil
}

// Report says how gh, the signing key, the files and ~/.gitconfig's include stand. The key shows by
// its .pub alone, and ~/.gitconfig by the include, as the file is the user's.
func (g *Git) Report(ctx context.Context) ([]step.Item, error) {
	if g.Conf.GitName == "" {
		return nil, nil
	}
	in, err := g.include(ctx)
	if err != nil {
		return nil, err
	}
	key, err := ReadKey(ctx, g.M, g.Conf, g.Home, SigningKey)
	if err != nil {
		return nil, err
	}
	gh, install, err := g.gh(ctx)
	if err != nil {
		return nil, err
	}
	tool := step.Item{Kind: step.PackageItem, Name: "gh", Want: stateInstalled, Found: stateInstalled}
	if install != nil {
		tool.Found = stateMissing
	}
	parts := []step.Reporter{fixed{tool}, g.dirs(key.File, key.Text)}
	for _, f := range g.list(key, gh) {
		parts = append(parts, f)
	}
	pub := keyCopy(g.M, key.File, key.Text, g.Home)
	if len(pub) == 0 {
		pub = []step.Reporter{userPath{M: g.M, Kind: step.FileItem, Path: key.File}}
	}
	parts = append(append(parts, pub...), fixed{lineItem(in.Path, "the include of ~/.config/ws/gitconfig", in.Exists, in.Included)})
	return gather(ctx, parts...)
}
