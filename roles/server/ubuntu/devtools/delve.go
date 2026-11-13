package devtools

import (
	"context"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/host"
)

// delveTool is Delve at a version, built with a Go version, for Zed's debugger. The
// versions are the pins, without the v and the go.
type delveTool struct {
	M                  effect.Machine
	User               host.User
	Path               string // where go install puts dlv
	Version, GoVersion string
}

type buildDelve struct{ d delveTool }

func (c buildDelve) String() string {
	return "build Delve v" + c.d.Version + " with Go " + c.d.GoVersion + " at " + c.d.Path
}

// Apply builds Delve where Dlv names it.
func (c buildDelve) Apply(ctx context.Context) error {
	return goInstall(ctx, c.d.M, c.d.User, "Delve", c.d.Path, delveModule+"/cmd/dlv@v"+c.d.Version)
}

// Plan reads the binary's build info. go version -m only opens the file, so the caller runs it and
// a dry run does too. A failed read counts as no Delve.
func (d delveTool) Plan(ctx context.Context) ([]host.Change, error) {
	out, err := d.M.Run(ctx, goBin, "version", "-m", d.Path)
	if delveCurrent(out, err, d.GoVersion, d.Version) {
		return nil, nil
	}
	return []host.Change{buildDelve{d}}, nil
}
