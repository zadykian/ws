package setup

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/step"
)

// Repo is where ws's releases are tagged.
const Repo = "https://github.com/zadykian/ws"

// releaseTag matches a release's tag, vX.Y.Z or vX.Y.Z-rc.N: numbers with no leading zero.
var releaseTag = regexp.MustCompile(`^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-rc\.[1-9][0-9]*)?$`)

// ReleaseTag returns the tag that version names, which ws's build stamps in as its version. A
// pseudo-version and devel name no tag, so setup refuses them.
func ReleaseTag(version string) (string, error) {
	if !releaseTag.MatchString(version) {
		return "", fmt.Errorf("this ws is %s, which names no release tag: run a release's ws, "+
			"or pass --in-place to run the tree in %s as it stands", version, TreeDir)
	}
	return version, nil
}

// Checkout is the tree in Dir, at Tag of Repo and clean. A first run makes Dir an empty repository
// and fetches into it, as a later run does. A change to the tree, and a file git does not track,
// such as a trial's mise.local.toml that would pin another ws, go. It plans nothing where the
// tree is at the tag and clean. With InPlace it plans nothing: the tree runs as it stands.
type Checkout struct {
	M       effect.Machine
	Dir     string
	Repo    string
	Tag     string
	InPlace bool
}

type checkOut struct {
	c     Checkout
	fresh bool
}

func (c checkOut) String() string {
	return fmt.Sprintf("check out %s of %s in %s", c.c.Tag, c.c.Repo, c.c.Dir)
}

func (checkOut) Kind() step.ChangeKind { return step.Install }

// Apply makes the repository where none stands. It fetches the tag, moves the tree to it and
// cleans the tree. A step that fails leaves the tree for the next run, which fetches again.
func (c checkOut) Apply(ctx context.Context) error {
	t := c.c
	if c.fresh {
		if _, err := t.M.Run(ctx, "git", "-c", "init.defaultBranch=ws", "init", "-q", t.Dir); err != nil {
			return fmt.Errorf("cannot create %s: %w", t.Dir, err)
		}
	}
	ref := "refs/tags/" + t.Tag
	for _, args := range [][]string{
		{"fetch", "-q", "--depth", "1", "--force", t.Repo, ref + ":" + ref},
		{"-c", "advice.detachedHead=false", "checkout", "-q", "--force", "--detach", ref},
		{"clean", "-q", "-ffdx"},
	} {
		if _, err := t.M.Run(ctx, "git", append([]string{"-C", t.Dir}, args...)...); err != nil {
			return fmt.Errorf("cannot check %s out in %s: %w", t.Tag, t.Dir, err)
		}
	}
	return nil
}

// Plan reads the tree, and plans the checkout where the tree is missing, not at the tag or not
// clean. A path that is no checkout fails, which the tree's owner removes by hand.
func (t Checkout) Plan(ctx context.Context) ([]resource.Change, error) {
	if t.InPlace {
		return nil, nil
	}
	if !releaseTag.MatchString(t.Tag) {
		return nil, fmt.Errorf("%q is no release tag", t.Tag)
	}
	if _, err := t.M.Run(ctx, "git", "--version"); err != nil {
		return nil, fmt.Errorf("git is needed to check the tree of %s out: %w", t.Tag, err)
	}
	if _, err := t.M.Stat(t.Dir); errors.Is(err, fs.ErrNotExist) {
		return []resource.Change{checkOut{t, true}}, nil
	} else if err != nil {
		return nil, err
	}
	if _, err := t.M.Stat(path.Join(t.Dir, ".git")); err != nil {
		return nil, fmt.Errorf("%s holds no checkout: remove it, or pass --in-place to run it", t.Dir)
	}
	if t.current(ctx) {
		return nil, nil
	}
	return []resource.Change{checkOut{t, false}}, nil
}

// current says whether the tree is at the tag, with nothing changed, untracked or ignored in it.
// Any failure to read says no, for the checkout to mend.
func (t Checkout) current(ctx context.Context) bool {
	head, err := t.git(ctx, "rev-parse", "--verify", "-q", "HEAD^{commit}")
	if err != nil {
		return false
	}
	tagged, err := t.git(ctx, "rev-parse", "--verify", "-q", "refs/tags/"+t.Tag+"^{commit}")
	if err != nil || head != tagged {
		return false
	}
	left, err := t.git(ctx, "--no-optional-locks", "status", "--porcelain", "--ignored")
	return err == nil && left == ""
}

func (t Checkout) git(ctx context.Context, args ...string) (string, error) {
	out, err := t.M.Run(ctx, "git", append([]string{"-C", t.Dir}, args...)...)
	return strings.TrimSpace(string(out)), err
}
