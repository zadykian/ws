package devtools

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io/fs"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
)

const npmBin = "/usr/bin/npm"

// serversDir is where the npm servers install. The steps and the settings both name it.
func serversDir(home string) string { return home + "/.local/lib/language-servers" }

// formatterLink puts sql-formatter on every PATH, root's, so a project's settings name the bare
// command. The target exists only after npm ci: Link compares the link alone, so a --check before
// the first install plans it.
const formatterLink = "/usr/local/bin/sql-formatter"

// serverBin is a bin of the npm servers.
func serverBin(home, name string) string { return serversDir(home) + "/node_modules/.bin/" + name }

// serverFiles returns the directories and the two files of the lock, the install they feed and
// sql-formatter's link.
// ~/.local and ~/.local/lib are made only where missing, as root's would break the user's own
// tools, such as claude and pip --user.
func serverFiles(m effect.Machine, user host.User, uid, gid int) ([]host.Resource, error) {
	dir := serversDir(user.Home)
	parts := []host.Resource{
		host.Plain{M: m, Home: user.Home, Paths: []string{
			dir + "/package.json", dir + "/package-lock.json", dir + "/node_modules/.ws-sha256",
		}},
		host.Dir{M: m, Path: user.Home + "/.local", Mode: 0o755, UID: uid, GID: gid, CreateOnly: true},
		host.Dir{M: m, Path: user.Home + "/.local/lib", Mode: 0o755, UID: uid, GID: gid, CreateOnly: true},
		host.Dir{M: m, Path: dir, Mode: 0o755, UID: uid, GID: gid},
	}
	sum := sha256.New()
	for _, name := range []string{"package.json", "package-lock.json"} {
		content, err := serversLock.ReadFile("servers/" + name)
		if err != nil {
			return nil, err
		}
		sum.Write(content)
		parts = append(parts, host.File{M: m, Path: dir + "/" + name, Content: content, Mode: 0o644,
			UID: uid, GID: gid})
	}
	return append(parts, npmInstall{M: m, User: user, UID: uid, GID: gid, Digest: fmt.Sprintf("%x", sum.Sum(nil))},
		host.Link{M: m, Path: formatterLink, Target: serverBin(user.Home, "sql-formatter")}), nil
}

// npmInstall runs npm ci. Plan cannot see the files written earlier in the step, so a stamp of
// their digest in node_modules, which npm ci removes, tells it the install is current.
type npmInstall struct {
	M        effect.Machine
	User     host.User
	UID, GID int
	Digest   string
}

func (n npmInstall) stamp() string { return serversDir(n.User.Home) + "/node_modules/.ws-sha256" }

type runNpm struct{ n npmInstall }

func (c runNpm) String() string { return "npm ci --ignore-scripts in " + serversDir(c.n.User.Home) }

func (c runNpm) Apply(ctx context.Context) error {
	n := c.n
	if _, err := n.M.Stat(npmBin); err != nil {
		return fmt.Errorf("the servers install with npm, from the Node package: %w", err)
	}
	if err := n.User.Pass(ctx, n.M, npmBin, "ci", "--prefix", serversDir(n.User.Home), "--ignore-scripts",
		"--no-audit", "--no-fund"); err != nil {
		return err
	}
	return n.M.WriteAtomic(n.stamp(), []byte(n.Digest+"\n"), effect.Attrs{UID: n.UID, GID: n.GID, Mode: 0o644})
}

// Plan finds a change where npm is missing or the stamp is missing or another digest.
func (n npmInstall) Plan(context.Context) ([]host.Change, error) {
	if _, err := n.M.Stat(npmBin); errors.Is(err, fs.ErrNotExist) {
		return []host.Change{runNpm{n}}, nil
	} else if err != nil {
		return nil, err
	}
	got, err := n.M.ReadFile(n.stamp())
	if errors.Is(err, fs.ErrNotExist) || (err == nil && string(got) != n.Digest+"\n") {
		return []host.Change{runNpm{n}}, nil
	}
	return nil, err
}
