package shell

import (
	"context"
	"strings"

	"github.com/zadykian/ws/effect/effecttest"
)

// stdinMachine answers RunInput by the start of its standard input, as the fake's table has the
// command line alone.
type stdinMachine struct {
	*effecttest.Machine
	answers map[string]string // the start of the input to the output
}

func (m stdinMachine) RunInput(ctx context.Context, stdin []byte, name string, args ...string) ([]byte, error) {
	if _, err := m.Machine.RunInput(ctx, stdin, name, args...); err != nil {
		return nil, err
	}
	for start, out := range m.answers {
		if strings.HasPrefix(string(stdin), start) {
			return []byte(out), nil
		}
	}
	return nil, nil
}
