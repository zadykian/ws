package secrets

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
)

// GH is gh where state.gh copies it, and where git's credential command runs it.
const GH = "/usr/local/bin/gh"

// checkWorkflow is the secrets repository's workflow of the checks values and gitleaks, which its
// ruleset requires before a merge.
const checkWorkflow = ".github/workflows/check.yml"

// inDir runs its second argument on in the directory of its first. fnox reads fnox.toml and a
// profile's file from its working directory alone, and macOS's env has no -C.
const inDir = `cd "$1" && shift && exec "$@"`

// Clone is a temporary clone of the secrets repository's main.
type Clone struct {
	Repo Repo
	// Dir is the clone's working tree.
	Dir string
	M   effect.Machine
	// FnoxEnv holds the assignments fnox runs with, its PATH and its key file.
	FnoxEnv []string

	// base is the temporary directory that holds the clone, which Close removes.
	base string
}

// Open clones the repository's main by HTTPS into a temporary directory, once gh reads it as
// private. A clone without the checks' workflow stops: a pull request's checks would never come.
func Open(ctx context.Context, m effect.Machine, r Repo) (*Clone, error) {
	out, err := m.Run(ctx, GH, "api", "repos/"+r.Slug(), "--jq", ".private")
	switch {
	case err != nil && strings.Contains(err.Error(), "HTTP 404"):
		return nil, fmt.Errorf("no repository %s that gh's login reads: create it once by gh repo create %s "+
			"--private, with its README, license and check workflow", r.Slug(), r.Slug())
	case err != nil:
		return nil, fmt.Errorf("cannot read the repository %s by gh: %w", r.Slug(), err)
	}
	switch private := strings.TrimSpace(string(out)); private {
	case "true":
	case "false":
		return nil, fmt.Errorf("%s is public: the secrets repository must be private", r.Slug())
	default:
		return nil, fmt.Errorf("gh reads %s as private %q, neither true nor false", r.Slug(), private)
	}
	out, err = m.Run(ctx, "mktemp", "-d")
	if err != nil {
		return nil, fmt.Errorf("cannot make a directory for the clone: %w", err)
	}
	base := strings.TrimSpace(string(out))
	if base == "" {
		return nil, errors.New("mktemp -d named no directory for the clone")
	}
	c := &Clone{Repo: r, Dir: base + "/repo", M: m, base: base}
	if _, err := m.Run(ctx, "env", "GIT_TERMINAL_PROMPT=0", "git", "clone", "-q", "--branch", "main",
		"--single-branch", r.HTTPS(), c.Dir); err != nil {
		c.Close(ctx)
		return nil, fmt.Errorf("cannot clone %s: %w", r.Slug(), err)
	}
	if _, err := m.Stat(c.Dir + "/" + checkWorkflow); err != nil {
		c.Close(ctx)
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("%s holds no %s on main: its checks values and gitleaks would never come",
				r.Slug(), checkWorkflow)
		}
		return nil, fmt.Errorf("cannot stat %s's %s: %w", r.Slug(), checkWorkflow, err)
	}
	return c, nil
}

// Close removes the clone and its directory. A clone holds fnox's ciphertexts alone, yet it goes
// on every path, as nothing reads it once the command ends.
func (c *Clone) Close(ctx context.Context) {
	if c.base != "" {
		_, _ = c.M.Run(ctx, "rm", "-rf", c.base)
	}
}

// Fnox runs fnox with args in the clone, with stdin as its input. It runs fnox from FnoxEnv's PATH
// with FnoxEnv's key file. It drops FNOX_AGE_KEY, which would beat any key file. It drops
// FNOX_PROFILE too, which would send fnox set to a profile's file. A value goes in stdin alone, as
// an error names the arguments.
func (c *Clone) Fnox(ctx context.Context, stdin []byte, args ...string) ([]byte, error) {
	return c.M.RunInput(ctx, stdin, "env", c.fnoxArgv(args)...)
}

// fnoxArgv is env's arguments that run fnox with args in the clone.
func (c *Clone) fnoxArgv(args []string) []string {
	argv := append([]string{"-u", "FNOX_AGE_KEY", "-u", "FNOX_PROFILE"}, c.FnoxEnv...)
	argv = append(argv, "sh", "-c", inDir, "sh", c.Dir, "fnox")
	return append(argv, args...)
}
