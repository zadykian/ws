package devtools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
)

// goplsTool is gopls at the pinned version, built with devtools' Go. gopls type-checks with the Go it
// was built with, so a new Go needs a new gopls too.
type goplsTool struct {
	M    effect.Machine
	User host.User
	// Path is where go install puts gopls, and where languages.toml names it.
	Path string
	// Version and GoVersion are the pins, without the v and the go. Empty means the package's and
	// devtools'.
	Version, GoVersion string
}

func (g goplsTool) version() string {
	if g.Version == "" {
		return goplsVersion
	}
	return g.Version
}

func (g goplsTool) goVersion() string {
	if g.GoVersion == "" {
		return GoVersion
	}
	return g.GoVersion
}

type buildGopls struct{ g goplsTool }

func (c buildGopls) String() string {
	return fmt.Sprintf("build gopls v%s with Go %s at %s", c.g.version(), c.g.goVersion(), c.g.Path)
}

// Apply builds gopls where languages.toml names it.
func (c buildGopls) Apply(ctx context.Context) error {
	return goInstall(ctx, c.g.M, c.g.User, "gopls", c.g.Path, "golang.org/x/tools/gopls@v"+c.g.version())
}

// Plan reads gopls's version and the Go it was built with. A gopls that is missing fails the
// command, which counts as another version.
func (g goplsTool) Plan(ctx context.Context) ([]host.Change, error) {
	out, err := g.User.Output(ctx, g.M, g.Path, "version", "-json")
	if err == nil {
		var got struct{ Version, GoVersion string }
		if err := json.Unmarshal(out, &got); err != nil {
			return nil, fmt.Errorf("read %s version -json: %w", g.Path, err)
		}
		if got.Version == "v"+g.version() && got.GoVersion == "go"+g.goVersion() {
			return nil, nil
		}
	}
	return []host.Change{buildGopls{g}}, nil
}
