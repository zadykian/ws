package kind

import (
	"errors"
	"fmt"
	"path"
	"slices"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/account"
	"github.com/zadykian/ws/engine/core/resource/file"
)

// checkLink refuses what a link cannot take. A link has no content, mode or group of its own, so
// only its path, its target, after and when mean anything. Root owns it, as the account could
// point a link of its own anywhere.
func (f filePart) checkLink() error {
	switch {
	case f.Template != "" || f.Source != "" || f.Absent || f.Line != "" || len(f.Has) > 0 ||
		f.Secret || len(f.Check) > 0 || f.CheckBooted || f.Mode != 0 || f.Group != "" ||
		f.Ours || f.Backup || f.AbsentWhenEmpty || f.Seed:
		return errors.New("a link takes a path, a target, after and when alone")
	case f.Owner != "" && f.Owner != "root":
		return errors.New("a link is root's")
	case !path.IsAbs(f.Link) || path.Clean(f.Link) != f.Link:
		return fmt.Errorf("link %q is not a clean absolute path", f.Link)
	}
	return nil
}

// linked is the link's resources: its missing parents, root's at 0755, then the link itself.
func (f filePart) linked(e Env) built {
	link := file.Link{M: e.M, Path: f.Path, Target: f.Link}
	return built{file: link, all: append(parents(e.M, f.Path), link),
		owner: account.Account{User: account.User{Name: "root"}}}
}

// parents makes each missing directory above p, root's at 0755, from the top down. An existing one
// is left alone.
func parents(m effect.Machine, p string) []resource.Resource {
	var dirs []resource.Resource
	for d := path.Dir(p); d != "/"; d = path.Dir(d) {
		dirs = append(dirs, file.Dir{M: m, Path: d, Mode: 0o755, CreateOnly: true})
	}
	slices.Reverse(dirs)
	return dirs
}
