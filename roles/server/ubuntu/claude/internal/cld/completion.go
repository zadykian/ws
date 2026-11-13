// Package cld sets up cld after its install: its update, its completion in bash, and its
// restore of sessions at boot.
package cld

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/host"
	"github.com/zadykian/ws/engine/core/host/account"
	"github.com/zadykian/ws/roles/server/ubuntu/claude/internal/asuser"
)

// Completion sets up cld's completion in bash. cld writes the script only where it differs.
type Completion struct {
	M    effect.Machine
	User account.User
}

type setupCompletion struct{ c Completion }

func (setupCompletion) String() string { return "set up cld's completion in bash" }

func (s setupCompletion) Apply(ctx context.Context) error {
	return s.c.User.Pass(ctx, s.c.M, s.c.User.Home+"/.local/bin/cld", "setup", "completion", "bash")
}

// Plan compares the script with what cld writes. Without cld, the install step reports the change.
func (c Completion) Plan(ctx context.Context) ([]host.Change, error) {
	cld := c.User.Home + "/.local/bin/cld"
	if _, err := c.M.Stat(cld); errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	want, err := c.User.Output(ctx, c.M, cld, "completion", "bash")
	if err != nil {
		return nil, fmt.Errorf("cld completion bash: %w", err)
	}
	have, err := asuser.Read(ctx, c.M, c.User, c.User.Home+"/.local/share/bash-completion/completions/cld")
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, err
	}
	if bytes.Equal(have, want) {
		return nil, nil
	}
	return []host.Change{setupCompletion{c}}, nil
}
