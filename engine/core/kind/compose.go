package kind

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"regexp"
	"slices"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/step"
)

// composeSums holds a file per project: the SHA-256 of its directory's files at its last good up.
// It sits outside the directory, which the sum covers.
const composeSums = "/var/lib/ws/compose"

// projectName is a compose project's name as compose takes it.
var projectName = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

// composeSpec is a unit of the kind compose: a project of Docker Compose, up on the system's daemon,
// whose compose.yaml and other files sit in Dir. A file that something else writes there, such as
// mise, recreates the containers at the next run, which they would not see otherwise.
type composeSpec struct {
	base
	Dir     string `toml:"dir"`
	Project string `toml:"project"`
}

func (s composeSpec) parts(e Env, _ fs.FS) (func(context.Context) ([]resource.Resource, error), error) {
	if !path.IsAbs(s.Dir) || path.Clean(s.Dir) != s.Dir || s.Dir == "/" {
		return nil, fmt.Errorf("dir %q is no clean absolute path below /", s.Dir)
	}
	if !projectName.MatchString(s.Project) {
		return nil, fmt.Errorf("project %q is no compose project's name: lowercase letters, digits, '_' and '-'", s.Project)
	}
	r := composeProject{m: e.M, dir: s.Dir, project: s.Project}
	return func(context.Context) ([]resource.Resource, error) { return []resource.Resource{r}, nil }, nil
}

// composeProject is a compose unit's resource. Compose's up runs the one-shot services again, a
// schema's migration among them. So it ups only where the files differ from those of the last good
// up, or where a service that keeps running is down.
type composeProject struct {
	m       effect.Machine
	dir     string
	project string
}

// sumPath is the project's file in composeSums.
func (p composeProject) sumPath() string { return path.Join(composeSums, p.project) }

// composeUp is the up of a project, which recreates every container where force is set.
type composeUp struct {
	p     composeProject
	force bool
}

func (c composeUp) String() string {
	if c.force {
		return "recreate the containers of " + c.p.project
	}
	return "start the containers of " + c.p.project + " that are down"
}

func (composeUp) Kind() step.ChangeKind { return step.Restart }

// args is compose's command line of the up.
func (c composeUp) args() []string {
	args := []string{"compose", "--project-directory", c.p.dir, "--project-name", c.p.project,
		"up", "--detach", "--quiet-pull", "--remove-orphans"}
	if c.force {
		args = append(args, "--force-recreate")
	}
	return args
}

// Apply sums the files, ups the project, and keeps the sum once the up succeeds. A failed up keeps
// the old sum, so that the next run recreates again: the old containers may still run the old files.
func (c composeUp) Apply(ctx context.Context) error {
	sum, err := projectSum(c.p.m, c.p.dir)
	if err != nil {
		return err
	}
	if err := c.p.m.Pass(ctx, true, "docker", c.args()...); err != nil {
		return err
	}
	if _, err := c.p.m.RunRoot(ctx, "mkdir", "-p", "--", composeSums); err != nil {
		return err
	}
	return c.p.m.WriteRootFile(c.p.sumPath(), []byte(sum+"\n"), 0o600)
}

// Plan recreates where the files' sum is not the kept one, and ups where a service that keeps
// running is down.
func (p composeProject) Plan(ctx context.Context) ([]resource.Change, error) {
	sum, err := projectSum(p.m, p.dir)
	if err != nil {
		return nil, err
	}
	kept, err := p.m.ReadRootFile(p.sumPath())
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	if err != nil || strings.TrimSpace(string(kept)) != sum {
		return []resource.Change{composeUp{p, true}}, nil
	}
	compose, err := p.m.ReadRootFile(path.Join(p.dir, composeFile))
	if err != nil {
		return nil, err
	}
	want, err := keepRunning(ctx, p.project, compose)
	if err != nil {
		return nil, err
	}
	out, err := p.m.RunRoot(ctx, "docker", "compose", "--project-name", p.project, "ps", "--status", "running", "--services")
	if err != nil {
		return nil, err
	}
	running := strings.Fields(string(out))
	if slices.ContainsFunc(want, func(s string) bool { return !slices.Contains(running, s) }) {
		return []resource.Change{composeUp{p, false}}, nil
	}
	return nil, nil
}

// projectSum returns the SHA-256, in hex, of the files below dir. It covers each regular file's path
// and content, read as root, as a password's file may be root's alone. It covers each link's path and
// target, unfollowed. An empty directory counts for nothing.
func projectSum(m effect.Machine, dir string) (string, error) {
	h := sha256.New()
	if err := sumBelow(m, h, dir, "."); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

// sumBelow writes the files below dir/rel to h, in the order of their names. Each entry starts with
// its path and a NUL. The length of a file's content or the word link follows, so that no two trees
// write the same bytes.
func sumBelow(m effect.Machine, h io.Writer, dir, rel string) error {
	names, err := m.ReadDir(path.Join(dir, rel))
	if err != nil {
		return err
	}
	for _, name := range names {
		r := path.Join(rel, name)
		at := path.Join(dir, r)
		info, err := m.Stat(at)
		if err != nil {
			return err
		}
		switch {
		case info.Link != "":
			_, err = fmt.Fprintf(h, "%s\x00link\x00%s\x00", r, info.Link)
		case info.Mode.IsDir():
			err = sumBelow(m, h, dir, r)
		case info.Mode.IsRegular():
			var data []byte
			if data, err = m.ReadRootFile(at); err == nil {
				_, err = fmt.Fprintf(h, "%s\x00%d\x00%s", r, len(data), data)
			}
		}
		if err != nil {
			return err
		}
	}
	return nil
}
