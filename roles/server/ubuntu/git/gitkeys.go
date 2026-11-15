package git

import (
	"context"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
)

// lines returns the non-empty lines of a command's output.
func lines(out []byte) []string {
	var l []string
	for _, line := range strings.Split(string(out), "\n") {
		if line != "" {
			l = append(l, line)
		}
	}
	return l
}

// gitNames lists the keys of a config's content, by git, without their values.
func gitNames(ctx context.Context, m effect.Machine, content []byte) ([]string, error) {
	out, err := m.RunInput(ctx, content, "git", "config", "--file", "-", "--name-only", "--list")
	return lines(out), err
}
