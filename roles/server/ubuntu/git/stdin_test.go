package git

import (
	"context"
	"strings"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

// stdinMachine answers RunInput by the start of its standard input, as the fake's table has the
// command line alone. paths answers git's include paths, helpers its credential helpers, and answers its
// names.
type stdinMachine struct {
	*effecttest.Machine
	answers map[string]string // the start of the input to the names
	paths   map[string]string // the start of the input to the include paths
	helpers map[string]string // the start of the input to the credential helpers, as --get-regexp prints them
}

func (m stdinMachine) RunInput(ctx context.Context, stdin []byte, name string, args ...string) ([]byte, error) {
	if _, err := m.Machine.RunInput(ctx, stdin, name, args...); err != nil {
		return nil, err
	}
	table := m.answers
	switch joined := strings.Join(args, " "); {
	case strings.Contains(joined, "--get-all"):
		table = m.paths
	case strings.Contains(joined, "--get-regexp"):
		table = m.helpers
	}
	for start, out := range table {
		if strings.HasPrefix(string(stdin), start) {
			return []byte(out), nil
		}
	}
	return nil, nil
}
