package git

import (
	"context"
	"io/fs"

	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/file"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/internal/keys"
	"github.com/zadykian/ws/roles/internal/etcws"
	"github.com/zadykian/ws/roles/internal/gitcfg"
	"github.com/zadykian/ws/roles/internal/sshcfg"
)

// part is a file or a directory that a run plans and a report reads.
type part interface {
	resource.Resource
	step.Reporter
}

// list returns the script for a key of Secretive's, root's allowed signers and gitconfig. The
// signers lie in /etc/ws, which a run writes through sudo. Where ws keeps a copy of the key's .pub,
// the copy's parts make /etc/ws, and the list leaves it out, as a plan holds it once.
func (g *Git) list(key keys.NamedKey, gh string) []part {
	dir := g.Home + "/.config/ws"
	in := gitcfg.Inputs{Key: key.File, Gh: gh}
	userFile := func(path, text string, mode fs.FileMode) file.UserFile {
		return file.UserFile{M: g.M, Out: g.Out, Path: path, Text: []byte(text), Mode: mode}
	}
	var list []part
	if sshcfg.SecretiveKey(key.File, g.Home) {
		in.Program = dir + "/secretive-ssh-keygen"
		list = append(list, userFile(in.Program, wrapper(g.Home+"/"+sshcfg.SecretiveData), 0o755))
	}
	signers := etcws.Files
	if keys.Copied(key.File, key.Text) {
		signers = etcws.Inside
	}
	for _, r := range signers(g.M, gitcfg.Signers, []byte(gitcfg.AllowedSigners(header, key.Key.String()))) {
		list = append(list, resource.Said{Out: g.Out, Part: r})
	}
	return append(list, userFile(dir+"/gitconfig", config(in), 0o644))
}

// files plans ~/.config/ws, ws's copy of the key's .pub where git signs with a copy, and the files in
// ~/.config/ws.
func (g *Git) files(ctx context.Context, key keys.NamedKey, gh string) ([]step.Change, error) {
	cs, link, err := file.WsDirs(g.M, g.Out, g.Home).Plan()
	if err != nil {
		return nil, err
	}
	copied, err := keys.PlanCopy(ctx, g.M, g.Out, key.File, key.Text)
	if err != nil {
		return nil, err
	}
	cs = append(cs, copied...)
	for _, f := range g.list(key, gh) {
		if cs, err = resource.Append(ctx, cs, f); err != nil {
			return nil, err
		}
	}
	link.Warn(g.Out, len(cs), "ws's files go there")
	return cs, nil
}

// report says how gh, the signing key, the files and ~/.gitconfig's include stand. The key shows by
// its .pub alone, and ~/.gitconfig by the include, as the file is the user's.
func (g *Git) report(ctx context.Context) ([]step.Item, error) {
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
	parts := []step.Reporter{file.Fixed{tool}, file.WsDirs(g.M, g.Out, g.Home)}
	for _, f := range g.list(key, gh) {
		parts = append(parts, f)
	}
	var pub []step.Reporter
	for _, part := range keys.CopyParts(g.M, g.Out, key.File, key.Text) {
		pub = append(pub, part)
	}
	if len(pub) == 0 {
		pub = []step.Reporter{file.UserPath{M: g.M, Kind: step.FileItem, Path: key.File}}
	}
	return file.Gather(ctx, append(append(parts, pub...), g.gitconfigLines())...)
}
