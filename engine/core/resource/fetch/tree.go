package fetch

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/file"
	"github.com/zadykian/ws/engine/core/step"
)

// treeMarker is the file in a tree that holds the URLs the tree came from, a line each, and so its
// probe. A tree with other URLs, a new version or architecture, or none, is unpacked again.
const treeMarker = ".ws-source"

// Tree is a directory unpacked from pinned archives, in their order, each less Strip leading parts
// of each name. A later archive's file replaces an earlier one's. The unpack goes to a hidden
// directory beside Path, which replaces Path whole, so that a run cut short leaves no half tree at
// Path. The archives wait in Cache, CacheDir where empty, and go with the install.
type Tree struct {
	M       effect.Machine
	Sources []PinnedFile // their Path and Mode are not used
	Path    string
	Strip   int
	Cache   string
}

// staging is where the tree is unpacked, beside Path so that the move is a rename.
func (t Tree) staging() string {
	return path.Join(path.Dir(t.Path), "."+path.Base(t.Path)+".staging")
}

func (t Tree) cache() string {
	if t.Cache == "" {
		return CacheDir
	}
	return t.Cache
}

type installTree struct {
	t     Tree
	files []string
	urls  []string
}

func (c installTree) String() string {
	return fmt.Sprintf("install %s in %s", strings.Join(c.urls, ", "), c.t.Path)
}

func (installTree) Kind() step.ChangeKind { return step.Install }

func (c installTree) Apply(ctx context.Context) error {
	staging := c.t.staging()
	if _, err := c.t.M.RunRoot(ctx, "rm", "-rf", "--", staging); err != nil {
		return err
	}
	for _, f := range c.files {
		if err := c.t.M.Unpack(ctx, f, staging, c.t.Strip); err != nil {
			return fmt.Errorf("tree %s: %w", c.t.Path, err)
		}
	}
	marker := effect.Attrs{Mode: 0o644}
	if err := c.t.M.WriteAtomic(path.Join(staging, treeMarker), []byte(markerText(c.urls)), marker); err != nil {
		return err
	}
	steps := [][]string{{"rm", "-rf", "--", c.t.Path}, {"mv", "-T", "--", staging, c.t.Path}}
	for _, f := range c.files {
		steps = append(steps, []string{"rm", "-f", "--", f})
	}
	for _, args := range steps {
		if _, err := c.t.M.RunRoot(ctx, args[0], args[1:]...); err != nil {
			return err
		}
	}
	return nil
}

// markerText is the marker's content for the URLs: a line each.
func markerText(urls []string) string {
	return strings.Join(urls, "\n") + "\n"
}

// cleaned plans the removal of what a run cut short left beside a current tree: its archives, and
// its staging directory. An archive's name needs the URL alone, so no source is asked for a sum.
func (t Tree) cleaned(ctx context.Context, urls []string) ([]resource.Change, error) {
	var changes []resource.Change
	leftovers := make([]string, 0, len(urls)+1)
	for _, url := range urls {
		leftovers = append(leftovers, path.Join(t.cache(), path.Base(url)))
	}
	leftovers = append(leftovers, t.staging())
	for _, p := range leftovers {
		more, err := file.Absent{M: t.M, Path: p}.Plan(ctx)
		if err != nil {
			return nil, err
		}
		changes = append(changes, more...)
	}
	return changes, nil
}

// Plan plans the removal of leftovers where the marker holds the pinned URLs. Otherwise it plans
// the directory, the fetch of each archive and the install. It asks the sources for their sums only
// for the fetch.
func (t Tree) Plan(ctx context.Context) ([]resource.Change, error) {
	if len(t.Sources) == 0 {
		return nil, fmt.Errorf("tree %s: no archive", t.Path)
	}
	urls := make([]string, 0, len(t.Sources))
	names := map[string]bool{}
	for _, s := range t.Sources {
		url, _, err := s.resolve(ctx)
		if err != nil {
			return nil, fmt.Errorf("tree %s: %w", t.Path, err)
		}
		if names[path.Base(url)] {
			return nil, fmt.Errorf("tree %s: two archives named %s would wait in one file", t.Path, path.Base(url))
		}
		urls, names[path.Base(url)] = append(urls, url), true
	}
	marker, err := t.M.ReadRootFile(path.Join(t.Path, treeMarker))
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	if err == nil && string(marker) == markerText(urls) {
		return t.cleaned(ctx, urls)
	}
	parts := []resource.Resource{file.Dir{M: t.M, Path: t.cache(), Mode: 0o755}}
	files := make([]string, 0, len(t.Sources))
	for _, s := range t.Sources {
		archive, err := s.cachedIn(ctx, t.cache())
		if err != nil {
			return nil, fmt.Errorf("tree %s: %w", t.Path, err)
		}
		parts, files = append(parts, archive), append(files, archive.Path)
	}
	var changes []resource.Change
	for _, r := range parts {
		more, err := r.Plan(ctx)
		if err != nil {
			return nil, err
		}
		changes = append(changes, more...)
	}
	return append(changes, installTree{t, files, urls}), nil
}
