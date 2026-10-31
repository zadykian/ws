package setup

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/zadykian/ws/mac/internal/gitcfg"
)

// gitAdd is the command that adds the key to the signing keys of gh's login.
func gitAdd(key gitKey, title string) string {
	return "gh ssh-key add " + key.Pub + " --type signing --title " + title
}

// ghFailure splits the error of a gh command into its exit status and its errors.
func ghFailure(err error) (int, string) {
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		return ee.ExitCode(), string(ee.Stderr)
	}
	return 1, err.Error()
}

// listed reports whether the signing keys of gh's login hold the key, and gh's failure, if any.
func (g *Git) listed(ctx context.Context, gh, public string) (bool, error) {
	out, err := g.M.Run(ctx, gh, "api", "--paginate", "user/ssh_signing_keys", "--jq", ".[].key")
	if err != nil {
		return false, err
	}
	for _, line := range strings.Split(string(out), "\n") {
		if public != "" && line == public {
			return true, nil
		}
	}
	return false, nil
}

// noteUnverified keeps the warning for a failure of gh's.
func (g *Git) noteUnverified(err error, pub string) {
	status, stderr := ghFailure(err)
	g.note("%s. Until a later run adds %s, GitHub shows the commits it signs as unverified",
		gitcfg.Why(status, stderr), pub)
}

// register plans to add the key to the signing keys of gh's login where they lack it, titled with
// the Mac's name, so that GitHub verifies the commits it signs. Without a login or its
// write:ssh_signing_key scope, or on another error of gh's, it warns, at each run until one adds the
// key. Before gh or the key is there, it says what it would add once they are.
func (g *Git) register(ctx context.Context, key gitKey, gh string, installing bool) (string, error) {
	title, err := g.M.Run(ctx, "hostname", "-s")
	if err != nil {
		return "", fmt.Errorf("hostname failed: %w", err)
	}
	add := gitAdd(key, strings.TrimSpace(string(title)))
	if installing {
		return "would run " + add + " once gh is installed, where it has a login that lacks the key", nil
	}
	listed, err := g.listed(ctx, gh, key.Public)
	switch {
	case err != nil:
		g.noteUnverified(err, key.Pub)
	case listed:
	case key.Public == "":
		return "would run " + add + " once the key is made", nil
	default:
		return "would run " + add, nil
	}
	return "", nil
}
