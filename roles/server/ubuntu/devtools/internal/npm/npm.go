// Package npm installs the language servers that npm serves, from a lock that pins each one.
package npm

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io/fs"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/host"
	"github.com/zadykian/ws/engine/core/host/account"
)

const npmBin = "/usr/bin/npm"

// serversDir is where the npm servers install. The steps and the settings both name it.
func serversDir(home string) string { return home + "/" + serversRel }

// serversRel is serversDir below the home.
const serversRel = ".local/lib/language-servers"

// formatterLink puts sql-formatter on every PATH, root's, so a project's settings name the bare
// command. The target exists only after npm ci: Link compares the link alone, so a --check before
// the first install plans it.
const formatterLink = "/usr/local/bin/sql-formatter"

// Bin is a bin of the npm servers.
func Bin(home, name string) string { return serversDir(home) + "/node_modules/.bin/" + name }

// Files returns the two files of the lock, the install they feed and sql-formatter's link.
// Their directories are made as the user's, so root's would not break the user's own tools, such as
// claude and pip --user.
func Files(m effect.Machine, a account.Account) ([]host.Resource, error) {
	sum := sha256.New()
	var parts []host.Resource
	for _, name := range []string{"package.json", "package-lock.json"} {
		content, err := serversLock.ReadFile("servers/" + name)
		if err != nil {
			return nil, err
		}
		sum.Write(content)
		parts = append(parts, a.File(m, serversRel+"/"+name, content, 0o644))
	}
	return append(parts, npmInstall{M: m, Account: a, Digest: fmt.Sprintf("%x", sum.Sum(nil))},
		host.Link{M: m, Path: formatterLink, Target: Bin(a.Home, "sql-formatter")}), nil
}

// npmInstall runs npm ci. Plan cannot see the files written earlier in the step, so a stamp of
// their digest in node_modules, which npm ci removes, tells it the install is current.
type npmInstall struct {
	M       effect.Machine
	Account account.Account
	Digest  string
}

// stampRel is the stamp below the home, which npm ci removes with node_modules.
const stampRel = serversRel + "/node_modules/.ws-sha256"

func (n npmInstall) stamp() string { return n.Account.Home + "/" + stampRel }

type runNpm struct{ n npmInstall }

func (c runNpm) String() string { return "npm ci --ignore-scripts in " + serversDir(c.n.Account.Home) }

func (c runNpm) Apply(ctx context.Context) error {
	n := c.n
	if _, err := n.M.Stat(npmBin); err != nil {
		return fmt.Errorf("the servers install with npm, from the Node package: %w", err)
	}
	if err := n.Account.Pass(ctx, n.M, npmBin, "ci", "--prefix", serversDir(n.Account.Home), "--ignore-scripts",
		"--no-audit", "--no-fund"); err != nil {
		return err
	}
	return n.M.WriteAtomic(n.stamp(), []byte(n.Digest+"\n"), effect.Attrs{UID: n.Account.UID, GID: n.Account.GID, Mode: 0o644})
}

// Plan finds a change where npm is missing or the stamp is missing or another digest.
func (n npmInstall) Plan(context.Context) ([]host.Change, error) {
	if _, err := n.M.Stat(npmBin); errors.Is(err, fs.ErrNotExist) {
		return []host.Change{runNpm{n}}, nil
	} else if err != nil {
		return nil, err
	}
	got, err := n.Account.ReadHome(n.M, stampRel)
	if errors.Is(err, fs.ErrNotExist) || (err == nil && string(got) != n.Digest+"\n") {
		return []host.Change{runNpm{n}}, nil
	}
	return nil, err
}
