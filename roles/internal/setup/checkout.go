package setup

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/zadykian/ws/engine/core/check"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/step"
)

// Repo is where ws's releases are tagged.
const Repo = "https://github.com/zadykian/ws"

// ReleaseTag returns the tag that version names, which ws's build stamps in as its version. A
// pseudo-version and devel name no tag, so setup refuses them.
func ReleaseTag(version string) (string, error) {
	if !check.Release(version) {
		return "", fmt.Errorf("this ws is %s, which names no release tag: run a release's ws, "+
			"or pass --in-place to run the tree in %s as it stands", version, TreeDir)
	}
	return version, nil
}

// Checkout is the tree in Dir, at Tag of Repo and clean. A first run makes Dir an empty repository
// and fetches into it, as a later run does. A change to the tree, and a file git does not track,
// such as a trial's mise.local.toml that would load over the tag's tree, go. It plans nothing where the
// tree is at the tag and clean. With InPlace it plans nothing: the tree runs as it stands.
type Checkout struct {
	M       effect.Machine
	Dir     string
	Repo    string
	Tag     string
	InPlace bool
	// AsUser is set where ws runs as a user, and root owns the tree, as on a Mac. A read of the tree
	// names it safe to git, which refuses another user's repository, and each write runs through sudo.
	AsUser bool
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
		if _, err := t.write(ctx, "-c", "init.defaultBranch=ws", "init", "-q", t.Dir); err != nil {
			return fmt.Errorf("cannot create %s: %w", t.Dir, err)
		}
	}
	ref := "refs/tags/" + t.Tag
	for _, args := range [][]string{
		{"fetch", "-q", "--depth", "1", "--force", t.Repo, ref + ":" + ref},
		{"-c", "advice.detachedHead=false", "checkout", "-q", "--force", "--detach", ref},
		{"clean", "-q", "-ffdx"},
	} {
		if _, err := t.write(ctx, append([]string{"-C", t.Dir}, args...)...); err != nil {
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
	if !check.Release(t.Tag) {
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

// git reads the tree, as the user where AsUser is set.
func (t Checkout) git(ctx context.Context, args ...string) (string, error) {
	argv := append([]string{"-C", t.Dir}, args...)
	if t.AsUser {
		argv = append([]string{"-c", "safe.directory=" + t.Dir}, argv...)
	}
	out, err := t.M.Run(ctx, "git", argv...)
	return strings.TrimSpace(string(out)), err
}

// write runs git to change the tree, through sudo where AsUser is set.
func (t Checkout) write(ctx context.Context, args ...string) ([]byte, error) {
	if t.AsUser {
		return t.M.RunRoot(ctx, "git", args...)
	}
	return t.M.Run(ctx, "git", args...)
}
