package signoz

import (
	"context"
	"io/fs"
	"path"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/host"
)

// Files is the step that copies the forged compose files into the project's directory. Compose
// resolves their relative bind mounts against it. The containers read them as users other than
// root, so they stay world-readable, as foundryctl writes them. Compose recreates a container
// whose settings changed, but not one whose bind-mounted file did, so a change of the files
// marks all containers for recreation.
func Files(m effect.Machine, dir string) host.Group {
	g := host.Group{Title: "signoz files", Parts: []host.Resource{host.Dir{M: m, Path: dir, Mode: 0o755}}}
	var copies []host.Resource
	err := fs.WalkDir(deployment, "deployment", func(p string, d fs.DirEntry, err error) error {
		if err != nil || p == "deployment" {
			return err
		}
		to := path.Join(dir, p[len("deployment/"):])
		if d.IsDir() {
			copies = append(copies, host.Dir{M: m, Path: to, Mode: 0o755})
			return nil
		}
		content, err := deployment.ReadFile(p)
		copies = append(copies, host.File{M: m, Path: to, Content: content, Mode: 0o644})
		return err
	})
	if err != nil {
		copies = []host.Resource{failed{err}}
	}
	g.Parts = append(g.Parts, mark{M: m, Path: markPath(dir), Watch: copies})
	g.Parts = append(g.Parts, copies...)
	return g
}

// failed is a resource whose plan fails: the embedded files cannot be read.
type failed struct{ err error }

func (f failed) Plan(context.Context) ([]host.Change, error) { return nil, f.err }
