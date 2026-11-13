package git

import (
	"context"
	"github.com/zadykian/ws/engine/core/resource/file"
	"io/fs"

	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/internal/gitcfg"
	"github.com/zadykian/ws/roles/client/macos/internal/keys"
	"github.com/zadykian/ws/roles/internal/sshcfg"
)

// list returns the files of ~/.config/ws: the script for a key of Secretive's, the allowed signers
// and gitconfig.
func (g *Git) list(key keys.NamedKey, gh string) []file.UserFile {
	dir := g.Home + "/.config/ws"
	in := gitcfg.Inputs{Name: g.Conf.GitName, Email: g.Conf.GitEmail, Key: key.File, Gh: gh, Dir: dir}
	userFile := func(path, text string, mode fs.FileMode) file.UserFile {
		return file.UserFile{M: g.M, Out: g.Out, Path: path, Text: []byte(text), Mode: mode}
	}
	var list []file.UserFile
	if sshcfg.SecretiveKey(key.File, g.Home) {
		in.Program = dir + "/secretive-ssh-keygen"
		list = append(list, userFile(in.Program, gitcfg.Wrapper(g.Home+"/"+sshcfg.SecretiveData), 0o755))
	}
	list = append(list, userFile(dir+"/allowed_signers", gitcfg.AllowedSigners(g.Conf.GitEmail, key.Key.String()), 0o644))
	return append(list, userFile(dir+"/gitconfig", gitcfg.Config(in), 0o644))
}

// dirs is ws's directory, with the directory of ws's copies where git signs with a copy.
func (g *Git) dirs(pub, text string) file.UserDirs {
	if keys.Copied(pub, text, g.Home) {
		return keys.CopyDirs(g.M, g.Out, g.Home)
	}
	return file.WsDirs(g.M, g.Out, g.Home)
}

// files plans ~/.config/ws and the files in it.
func (g *Git) files(key keys.NamedKey, gh string) ([]step.Change, error) {
	cs, link, err := g.dirs(key.File, key.Text).Plan()
	if err != nil {
		return nil, err
	}
	if keys.Copied(key.File, key.Text, g.Home) {
		cs = append(cs, keys.CopyChange(g.M, g.Out, key.File, key.Text)...)
	}
	for _, f := range g.list(key, gh) {
		planned, err := f.Plan()
		if err != nil {
			return nil, err
		}
		cs = append(cs, planned...)
	}
	link.Warn(g.Out, len(cs), "ws's files go there")
	return cs, nil
}

// Report says how gh, the signing key, the files and ~/.gitconfig's include stand. The key shows by
// its .pub alone, and ~/.gitconfig by the include, as the file is the user's.
func (g *Git) Report(ctx context.Context) ([]step.Item, error) {
	if g.Conf.GitName == "" {
		return nil, nil
	}
	key, err := keys.ReadKey(ctx, g.M, g.Conf, g.Home, keys.SigningKey)
	if err != nil {
		return nil, err
	}
	gh, install, err := g.gh(ctx)
	if err != nil {
		return nil, err
	}
	tool := step.Item{Kind: step.PackageItem, Name: "gh", Want: file.StateInstalled, Found: file.StateInstalled}
	if install != nil {
		tool.Found = file.StateMissing
	}
	parts := []step.Reporter{file.Fixed{tool}, g.dirs(key.File, key.Text)}
	for _, f := range g.list(key, gh) {
		parts = append(parts, f)
	}
	pub := keys.CopyReport(g.M, g.Out, key.File, key.Text, g.Home)
	if len(pub) == 0 {
		pub = []step.Reporter{file.UserPath{M: g.M, Kind: step.FileItem, Path: key.File}}
	}
	return file.Gather(ctx, append(append(parts, pub...), g.gitconfigLines())...)
}
