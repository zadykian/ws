// Package secrets sets up the secrets repository from the Mac. It reads the Nano's age
// identities, the server's inputs and fnox.toml. It sets and drops a server's host key in the fnox
// files, and reads it from known_hosts. It clones the repository, runs fnox in the clone, and lands
// a change by a pull request.
package secrets

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/pelletier/go-toml/v2"

	"github.com/zadykian/ws/engine/core/effect"
)

const (
	// Tree is where the Mac keeps ws's tree, whose mise.toml names the secrets repository.
	Tree = "/opt/ws"
	// RepoKey is the variable of mise.toml's [env] that names the secrets repository.
	RepoKey = "WS_SECRETS_REPO"
)

// Repo is a repository on GitHub.
type Repo struct {
	Owner, Name string
}

// Slug is OWNER/NAME, as gh names a repository.
func (r Repo) Slug() string {
	return r.Owner + "/" + r.Name
}

// HTTPS is the repository's URL by HTTPS. The Mac clones and pushes by it, as gh's credential
// command serves it and the Mac's ssh config names no host of GitHub's.
func (r Repo) HTTPS() string {
	return "https://github.com/" + r.Slug() + ".git"
}

// GitHub's names: an owner of letters, digits and hyphens, a repository of letters, digits, dots,
// hyphens and underscores. Each goes into gh's and git's arguments, so nothing else may.
var (
	ownerName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]*$`)
	repoName  = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)
)

// ParseRepo reads a repository from its SSH URL on GitHub, git@github.com:OWNER/NAME.git, the
// form a server fetches by. The .git is optional.
func ParseRepo(url string) (Repo, error) {
	path, ok := strings.CutPrefix(url, "git@github.com:")
	if !ok {
		return Repo{}, fmt.Errorf("%s is no SSH URL of GitHub's, git@github.com:OWNER/NAME.git", url)
	}
	owner, name, _ := strings.Cut(strings.TrimSuffix(path, ".git"), "/")
	if !ownerName.MatchString(owner) || !repoName.MatchString(name) || name == "." || name == ".." {
		return Repo{}, fmt.Errorf("%s names no repository OWNER/NAME of GitHub's", url)
	}
	return Repo{Owner: owner, Name: name}, nil
}

// RepoOf reads the secrets repository from the [env] of the tree's mise.toml. No mise runs ws on
// the Mac. So ws reads the variable from the file, where mise reads it for a server's task.
func RepoOf(m effect.Machine, tree string) (Repo, error) {
	file := tree + "/mise.toml"
	data, err := m.ReadFile(file)
	if err != nil {
		return Repo{}, fmt.Errorf("cannot read %s: %w", file, err)
	}
	var conf struct {
		Env map[string]any `toml:"env"`
	}
	if err := toml.Unmarshal(data, &conf); err != nil {
		return Repo{}, fmt.Errorf("cannot read %s: %w", file, err)
	}
	url, ok := conf.Env[RepoKey].(string)
	if !ok {
		return Repo{}, fmt.Errorf("%s sets no %s in its [env]: run a release of ws that names the secrets repository",
			file, RepoKey)
	}
	r, err := ParseRepo(url)
	if err != nil {
		return Repo{}, fmt.Errorf("%s of %s: %w", RepoKey, file, err)
	}
	return r, nil
}
