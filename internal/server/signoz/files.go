package signoz

import (
	"context"
	"io/fs"
	"path"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
)

// Files is the step that copies the forged compose files into the project's directory. Compose
// resolves their relative bind mounts against it. The containers read them as users other than
// root, so they stay world-readable, as foundryctl writes them.
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
	g.Parts = append(g.Parts, copies...)
	return g
}

// failed is a resource whose plan fails: the embedded files cannot be read.
type failed struct{ err error }

func (f failed) Plan(context.Context) ([]host.Change, error) { return nil, f.err }
