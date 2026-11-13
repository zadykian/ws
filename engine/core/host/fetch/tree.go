package fetch

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/host"
	"github.com/zadykian/ws/engine/core/host/file"
	"github.com/zadykian/ws/engine/core/step"
)

// treeMarker is the file in a tree that holds the URL the tree came from, and so its probe. A
// tree with another URL, a new version or architecture, or none, is unpacked again.
const treeMarker = ".ws-source"

// Tree is a directory unpacked from a pinned archive, less Strip leading parts of each name. The
// unpack goes to a hidden directory beside Path, which replaces Path whole, so that a run cut
// short leaves no half tree at Path. The archive waits in CacheDir and goes with the install.
type Tree struct {
	M      effect.Machine
	Source PinnedFile // its Path and Mode are not used
	Path   string
	Strip  int
}

// staging is where the tree is unpacked, beside Path so that the move is a rename.
func (t Tree) staging() string {
	return path.Join(path.Dir(t.Path), "."+path.Base(t.Path)+".staging")
}

type installTree struct {
	t    Tree
	file string
	url  string
}

func (c installTree) String() string { return fmt.Sprintf("install %s in %s", c.url, c.t.Path) }

func (installTree) Kind() step.ChangeKind { return step.Install }

func (c installTree) Apply(ctx context.Context) error {
	staging := c.t.staging()
	if _, err := c.t.M.RunRoot(ctx, "rm", "-rf", "--", staging); err != nil {
		return err
	}
	if err := c.t.M.Unpack(ctx, c.file, staging, c.t.Strip); err != nil {
		return fmt.Errorf("tree %s: %w", c.t.Path, err)
	}
	marker := effect.Attrs{Mode: 0o644}
	if err := c.t.M.WriteAtomic(path.Join(staging, treeMarker), []byte(c.url+"\n"), marker); err != nil {
		return err
	}
	for _, args := range [][]string{{"rm", "-rf", "--", c.t.Path}, {"mv", "-T", "--", staging, c.t.Path},
		{"rm", "-f", "--", c.file}} {
		if _, err := c.t.M.RunRoot(ctx, args[0], args[1:]...); err != nil {
			return err
		}
	}
	return nil
}

// cleaned plans the removal of what a run cut short left beside a current tree: its archive, and
// its staging directory. The archive's name needs the URL alone, so the source is not asked for a
// sum.
func (t Tree) cleaned(ctx context.Context, url string) ([]host.Change, error) {
	var changes []host.Change
	for _, p := range []string{path.Join(CacheDir, path.Base(url)), t.staging()} {
		more, err := file.Absent{M: t.M, Path: p}.Plan(ctx)
		if err != nil {
			return nil, err
		}
		changes = append(changes, more...)
	}
	return changes, nil
}

// Plan plans the removal of leftovers where the marker holds the pinned URL. Otherwise it plans the
// directory, the fetch of the archive and the install. It asks the source for its sum only for the
// fetch.
func (t Tree) Plan(ctx context.Context) ([]host.Change, error) {
	url, _, err := t.Source.resolve(ctx)
	if err != nil {
		return nil, fmt.Errorf("tree %s: %w", t.Path, err)
	}
	marker, err := t.M.ReadRootFile(path.Join(t.Path, treeMarker))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	if err == nil && string(marker) == url+"\n" {
		return t.cleaned(ctx, url)
	}
	archive, err := t.Source.Cached(ctx)
	if err != nil {
		return nil, fmt.Errorf("tree %s: %w", t.Path, err)
	}
	var changes []host.Change
	for _, r := range []host.Resource{file.Dir{M: t.M, Path: CacheDir, Mode: 0o755}, archive} {
		more, err := r.Plan(ctx)
		if err != nil {
			return nil, err
		}
		changes = append(changes, more...)
	}
	return append(changes, installTree{t, archive.Path, url}), nil
}
