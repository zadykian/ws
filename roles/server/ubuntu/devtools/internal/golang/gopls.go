package golang

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/account"
)

// Gopls is gopls at the pinned version, built with devtools' Go. gopls type-checks with the Go it
// was built with, so a new Go needs a new gopls too.
type Gopls struct {
	M    effect.Machine
	User account.User
	// Path is where go install puts gopls, and where languages.toml names it.
	Path string
	// Version and GoVersion are the pins, without the v and the go.
	Version, GoVersion string
}

type buildGopls struct{ g Gopls }

func (c buildGopls) String() string {
	return fmt.Sprintf("build gopls v%s with Go %s at %s", c.g.Version, c.g.GoVersion, c.g.Path)
}

// Apply builds gopls where languages.toml names it.
func (c buildGopls) Apply(ctx context.Context) error {
	return goInstall(ctx, c.g.M, c.g.User, "gopls", c.g.Path, "golang.org/x/tools/gopls@v"+c.g.Version)
}

// Plan reads gopls's version and the Go that built it. A gopls that is missing fails the
// command, which counts as another version.
func (g Gopls) Plan(ctx context.Context) ([]resource.Change, error) {
	out, err := g.User.Output(ctx, g.M, g.Path, "version", "-json")
	if err == nil {
		var got struct{ Version, GoVersion string }
		if err := json.Unmarshal(out, &got); err != nil {
			return nil, fmt.Errorf("read %s version -json: %w", g.Path, err)
		}
		if got.Version == "v"+g.Version && got.GoVersion == "go"+g.GoVersion {
			return nil, nil
		}
	}
	return []resource.Change{buildGopls{g}}, nil
}
