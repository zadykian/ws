package kind

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource/account"
)

// System is the operating system that a side's units run on, as the tools that find a home differ.
// The side sets it in Env, and Linux is the zero value.
type System int

const (
	// Linux finds a home in the passwd database, by getent.
	Linux System = iota
	// MacOS finds a home in Directory Service, by dscl, as macOS has no getent.
	MacOS
)

func (s System) String() string {
	if s == MacOS {
		return "macOS"
	}
	return "Linux"
}

// home is the home of name, as the system's database of accounts gives it.
func (s System) home(ctx context.Context, m effect.Machine, name string) (string, error) {
	var out []byte
	var err error
	dir := ""
	switch s {
	case MacOS:
		out, err = m.Run(ctx, "dscl", ".", "-read", "/Users/"+name, "NFSHomeDirectory")
		// dscl puts a long value on a line of its own, below the key.
		if rest, ok := strings.CutPrefix(strings.TrimSpace(string(out)), "NFSHomeDirectory:"); ok {
			dir = strings.TrimSpace(rest)
		}
	default:
		out, err = m.Run(ctx, "getent", "passwd", name)
		if fields := strings.Split(strings.TrimSpace(string(out)), ":"); len(fields) == 7 {
			dir = fields[5]
		}
	}
	if err != nil {
		return "", fmt.Errorf("find the home of %s: %w", name, err)
	}
	if !path.IsAbs(dir) || path.Clean(dir) == "/" {
		return "", fmt.Errorf("find the home of %s: %q names no home", name, out)
	}
	return path.Clean(dir), nil
}

// ownerOf is the account of name, root's for an empty one. Another account's Home is its home, as
// the system's database of accounts gives it.
func ownerOf(ctx context.Context, e Env, name string) (account.Account, error) {
	if name == "" || name == "root" {
		return account.Account{User: account.User{Name: "root"}}, nil
	}
	a, err := account.User{Name: name}.Lookup(ctx, e.M)
	if err != nil {
		return account.Account{}, err
	}
	a.Home, err = e.System.home(ctx, e.M, name)
	return a, err
}
