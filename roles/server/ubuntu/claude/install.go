package claude

import (
	"context"
	"errors"
	"io/fs"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/account"
)

// Install installs claude and cld where they are missing. claude updates itself in the background,
// and cld is updated by its own step, so a command that is there is left alone. Each installer
// runs as the user, with HOME set, as it writes under it.
type Install struct {
	M    effect.Machine
	User account.User
}

func (i Install) claude() string { return i.User.Home + "/.local/bin/claude" }

func (i Install) cld() string { return i.User.Home + "/.local/bin/cld" }

// installer is a command that is installed by a script from a URL.
type installer struct {
	i     Install
	name  string
	path  string // where the installer puts the command
	url   string
	shell string   // the interpreter of the script
	env   []string // the installer's environment
}

func (c installer) String() string { return "install " + c.name + " from " + c.url }

// Apply fetches the script and runs it, so that a failed fetch is an error, as with pipefail.
// The installer of claude refuses root where SUDO_USER is set, as under sudo -i, from which the
// README runs ws server ubuntu apply; root is meant here. cld's installer would take the caller's
// CLD_INSTALL_DIR over ~/.local/bin, so the step sets it. --retry rides out a 5xx.
func (c installer) Apply(ctx context.Context) error {
	script, err := c.i.User.Output(ctx, c.i.M, "curl", "-fsSL", "--retry", "3", c.url)
	if err != nil {
		return err
	}
	return c.i.User.PassInput(ctx, c.i.M, script, "env", append(c.env, c.shell)...)
}

// missing reports whether the path is absent.
func (i Install) missing(path string) (bool, error) {
	_, err := i.M.Stat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return true, nil
	}
	return false, err
}

// Plan looks for the two commands.
func (i Install) Plan(context.Context) ([]resource.Change, error) {
	var changes []resource.Change
	for _, c := range []installer{
		{i: i, name: "claude", path: i.claude(), url: "https://claude.ai/install.sh", shell: "bash",
			env: []string{"CLAUDE_INSTALL_ALLOW_SUDO=1"}},
		{i: i, name: "cld", path: i.cld(), url: "https://github.com/zadykian/cld/releases/latest/download/install.sh",
			shell: "sh", env: []string{"CLD_INSTALL_DIR=" + i.User.Home + "/.local/bin"}},
	} {
		gone, err := i.missing(c.path)
		if err != nil {
			return nil, err
		}
		if gone {
			changes = append(changes, c)
		}
	}
	return changes, nil
}
