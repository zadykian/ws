package wizard

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/zadykian/ws/mac/internal/effect"
	"github.com/zadykian/ws/mac/internal/setup"
	"github.com/zadykian/ws/mac/internal/step"
)

// ghStage names the stage in its messages.
const ghStage = "gh on the Mac"

// ghScopes are the scopes of gh's login that gh ssh-key add needs, for a key that logs in and for
// one that signs.
const ghScopes = "write:public_key,write:ssh_signing_key"

// Gh is the stage of GitHub's CLI: from Homebrew where missing, and logged in to github.com with
// the scopes that add the Mac's keys. Its login's keys must hold the Mac's key for ws, which
// Ubuntu's installer imports for ws's user. setup's git step, which runs after, adds git's signing
// key with that login.
type Gh struct {
	// Conf is mac.conf's path, and Home the user's.
	Conf, Home string
	// Tty tells whether a terminal is there for gh's device flow.
	Tty bool
	// LookPath finds a program on PATH.
	LookPath func(string) (string, error)
	M        effect.Machine
	Out      effect.Out
}

// Name is the stage's name.
func (*Gh) Name() string { return ghStage }

// WaitsOn is the Mac key, whose .pub this stage adds to the login's keys.
func (*Gh) WaitsOn() string { return macKey }

// ghLogin is the change that has gh log in, or refresh the scopes of its login. Its device flow
// shows a code to enter in the browser, at github.com/login/device.
type ghLogin struct{ args []string }

func (c ghLogin) String() string {
	return fmt.Sprintf("%s: would run %s", ghStage, c.line())
}

func (c ghLogin) line() string {
	return "gh auth " + strings.Join(c.args, " ") + " -h github.com -s " + ghScopes
}

// ghAdd is the change that adds the Mac's key for ws to the login's keys.
type ghAdd struct{ pub, title string }

func (c ghAdd) line() string { return "gh ssh-key add " + c.pub + " --title " + c.title }

func (c ghAdd) String() string { return fmt.Sprintf("%s: would run %s", ghStage, c.line()) }

// ghPath returns the path of gh, and the change that installs it where PATH has none.
func (g *Gh) ghPath(ctx context.Context) (string, []step.Change, error) {
	path, install, err := setup.FindGh(ctx, g.M, g.look)
	if err != nil {
		return "", nil, fmt.Errorf("%s: %w", ghStage, err)
	}
	if install != nil {
		return path, []step.Change{install}, nil
	}
	return path, nil, nil
}

func (g *Gh) look(name string) (string, error) {
	if g.LookPath != nil {
		return g.LookPath(name)
	}
	return exec.LookPath(name)
}
