package git

import (
	"context"
	"strings"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/resource/account"
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

// gitFixture is a machine whose git answers by stdinMachine's tables, empty to begin with, and the
// shadow of the account ws on it.
func gitFixture() (*effecttest.Machine, stdinMachine, shadow) {
	m := effecttest.NewMachine()
	m.Results["git config --file - --name-only --list"] = effecttest.Result{}
	m.Results["git config --file - --type path --get-all include.path"] = effecttest.Result{}
	m.Results["git config --no-includes --file - --get-regexp "+programPattern] = effecttest.Result{}
	sm := stdinMachine{Machine: m, answers: map[string]string{}, paths: map[string]string{}, helpers: map[string]string{}}
	a := account.Account{User: account.User{Name: "ws", Home: "/home/ws"}, UID: 1000, GID: 1000}
	return m, sm, shadow{M: sm, A: a}
}
