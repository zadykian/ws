package setup

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

// gh returns the path of GitHub's CLI, which git runs in whatever directory it works in, so the path
// must be absolute, not one that a relative directory of PATH gives. Where PATH has none, it
// returns the path Homebrew would give it, and the change that installs it.
func (g *Git) gh(ctx context.Context) (string, *Run, error) {
	path, err := g.look("gh")
	if err != nil {
		if _, err := g.look("brew"); err != nil {
			return "", nil, errors.New("install Homebrew first: https://brew.sh")
		}
		prefix, err := g.M.Run(ctx, "brew", "--prefix")
		if err != nil {
			return "", nil, fmt.Errorf("brew --prefix failed: %w", err)
		}
		return strings.TrimSpace(string(prefix)) + "/bin/gh", &Run{Name: "brew", Args: []string{"install", "gh"}}, nil
	}
	if !filepath.IsAbs(path) {
		return "", nil, fmt.Errorf("PATH gives gh as %s, which git would look for in each repository: "+
			"drop PATH's relative directories", path)
	}
	return path, nil, nil
}
