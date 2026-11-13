package git

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"slices"
	"strings"

	"github.com/zadykian/ws/roles/client/macos/internal/gitcfg"
	"github.com/zadykian/ws/roles/client/macos/internal/keys"
)

// Register adds the key to the signing keys of gh's login.
type Register struct {
	Git                    *Git
	Gh, Pub, Public, Title string
}

func (r Register) command() string {
	return "gh ssh-key add " + r.Pub + " --type signing --title " + r.Title
}

// String is the line that --check prints.
func (r Register) String() string { return "would run " + r.command() }

// Apply adds the key. A failure is a warning, and the next run tries again.
func (r Register) Apply(ctx context.Context) error {
	r.Git.add(ctx, r)
	return nil
}

// waiting is the registration that waits for gh's install. It says what it would run, and once gh
// is there it plans the registration again and makes it, with the lines that follow.
type waiting struct {
	Git  *Git
	Key  keys.NamedKey
	Gh   string
	Text string
	// Notes are the warnings that the plan took before the registration's, which come first.
	Notes []string
}

func (w waiting) String() string { return w.Text }

// Apply plans the registration with gh installed, adds the key where the plan does, and says the
// lines that the plan keeps, the warnings taken before the registration's first.
func (w waiting) Apply(ctx context.Context) error {
	reg, _, err := w.Git.register(ctx, w.Key, w.Gh, false)
	if err != nil {
		return err
	}
	if reg != nil {
		w.Git.add(ctx, *reg)
	}
	w.Git.notes = append(slices.Clone(w.Notes), w.Git.notes...)
	w.Git.withNotes(nil)
	return nil
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

// unverified is the warning for a failure of gh's.
func unverified(err error, pub string) string {
	status, stderr := ghFailure(err)
	return fmt.Sprintf("%s. Until a later run adds %s, GitHub shows the commits it signs as unverified",
		gitcfg.Why(status, stderr), pub)
}

// register plans to add the key to the signing keys of gh's login where they lack it, titled with
// the Mac's name. That way GitHub verifies the commits the key signs. Without a login or its
// write:ssh_signing_key scope, or on another error of gh's, it warns, at each run until one adds
// the key. Before gh is installed, it says what it would add with gh there.
func (g *Git) register(ctx context.Context, key keys.NamedKey, gh string, installing bool) (*Register, string, error) {
	title, err := g.M.Run(ctx, "hostname", "-s")
	if err != nil {
		return nil, "", fmt.Errorf("hostname failed: %w", err)
	}
	r := Register{g, gh, key.File, key.Key.String(), strings.TrimSpace(string(title))}
	if installing {
		return nil, "would run " + r.command() + " once gh is installed, where it has a login that lacks the key", nil
	}
	listed, err := g.listed(ctx, gh, key.Key.String())
	switch {
	case err != nil:
		g.note("%s", unverified(err, key.File))
	case listed:
		if !g.added {
			g.know("git's key is among GitHub's signing keys already")
		}
	case g.addErr != nil:
		// The add failed in this run: planning it again would end the step in an error, where the
		// warning is all that a later run can change.
		g.note("%s", unverified(g.addErr, key.File))
	default:
		return &r, "", nil
	}
	return nil, "", nil
}

// add adds the key, then checks that the signing keys list it, and warns where they do not.
func (g *Git) add(ctx context.Context, r Register) {
	if _, err := g.M.Run(ctx, r.Gh, "ssh-key", "add", r.Pub, "--type", "signing", "--title", r.Title); err != nil {
		g.addErr = err
		g.say(unverified(err, r.Pub))
		return
	}
	listed, err := g.listed(ctx, r.Gh, r.Public)
	if err == nil && !listed {
		err = errors.New("gh api user/ssh_signing_keys lacks the key after " + r.command())
	}
	if err != nil {
		g.addErr = err
		g.say(unverified(err, r.Pub))
		return
	}
	g.added = true
	g.Out.Say("ran %s", r.command())
}
