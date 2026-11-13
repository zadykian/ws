package swap

import (
	"context"
	"embed"
	"fmt"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/host"
	"github.com/zadykian/ws/engine/core/host/file"
)

//go:embed files/99-swappiness.conf.tmpl
var sysctlFiles embed.FS

// sysctlFile is /etc/sysctl.d/99-swappiness.conf.
type sysctlFile struct {
	M          effect.Machine
	Swappiness int
}

// Plan checks the value, from 0 to 200, and plans the file.
func (f sysctlFile) Plan(ctx context.Context) ([]host.Change, error) {
	if f.Swappiness < 0 || f.Swappiness > 200 {
		return nil, fmt.Errorf("base_swappiness is %d: it takes a whole number from 0 to 200", f.Swappiness)
	}
	content, err := host.Render(sysctlFiles, "files/99-swappiness.conf.tmpl", f)
	if err != nil {
		return nil, err
	}
	return file.File{M: f.M, Path: "/etc/sysctl.d/99-swappiness.conf", Content: content, Mode: 0o644}.Plan(ctx)
}
