package shell

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/host"
	"github.com/zadykian/ws/internal/server/values"
)

// checkClones refuses a repo with a user name or token in a URL of http:// or https://. Such a URL
// would stay in the clone's config and in the run's output. It also refuses a dest with *, ? or [,
// which the clone's check could take for a pattern. A blank repo or dest counts as missing. The
// message names no repo.
func checkClones(clones []values.GitClone) error {
	for _, c := range clones {
		if c.Repo == "" || strings.Contains(c.Repo, "\n") || hasUserInfo(c.Repo) ||
			c.Dest == "" || strings.ContainsAny(c.Dest, "*?[") {
			return errors.New("each entry of shell_git_clones needs a repo, with no user name or token in a URL of http:// or https://, and a dest without *, ? or [")
		}
	}
	return nil
}

// hasUserInfo reports whether the repo is an http(s) URL whose authority holds an @.
func hasUserInfo(repo string) bool {
	for _, scheme := range []string{"http://", "https://"} {
		if rest, ok := strings.CutPrefix(repo, scheme); ok {
			authority, _, _ := strings.Cut(rest, "/")
			return strings.Contains(authority, "@")
		}
	}
	return false
}

// clones clones each repository whose dest is missing, once: a clone is never pulled or reset, as
// once made it belongs to the user. It runs git clone, not a module that asks the server where dest
// is missing, so that a dry run lists each missing dest offline. It runs as the user, whose
// ~/.gitconfig gives the tokens, in the user's home.
type clones struct {
	M      effect.Machine
	A      host.Account
	Clones []values.GitClone
}

type cloneRepo struct {
	c    clones
	repo string
	dest string
}

func (c cloneRepo) String() string { return fmt.Sprintf("clone %s into %s", c.repo, c.dest) }

func (c cloneRepo) Apply(ctx context.Context) error {
	return c.c.A.Pass(ctx, c.c.M, "git", "-C", c.c.A.Home, "clone", "--", c.repo, c.dest)
}

// Plan checks the clones, then lists those whose dest is missing, a relative one under the home.
func (c clones) Plan(context.Context) ([]host.Change, error) {
	if err := checkClones(c.Clones); err != nil {
		return nil, err
	}
	var changes []host.Change
	for _, cl := range c.Clones {
		abs := cl.Dest
		if !strings.HasPrefix(abs, "/") {
			abs = c.A.Home + "/" + abs
		}
		_, err := c.M.RootAttrs(abs)
		if errors.Is(err, fs.ErrNotExist) {
			changes = append(changes, cloneRepo{c, cl.Repo, cl.Dest})
		} else if err != nil {
			return nil, err
		}
	}
	return changes, nil
}
