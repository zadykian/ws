package cld

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"regexp"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/host"
	"github.com/zadykian/ws/engine/core/host/systemd"
)

// Restore has the user's systemd restore cld's sessions as it starts, which brings back the
// sessions a reboot ended. cld setup restore writes the user unit. The rest is left to Restore and
// the steps around it. One is lingering, so that the user's systemd starts at boot rather than at
// the first login. Another is a shell where systemctl --user reaches it. A container has no
// systemd, so Restore plans nothing there.
type Restore struct {
	M       effect.Machine
	Account host.Account
	Out     effect.Out
}

func (r Restore) cld() string { return r.Account.Home + "/.local/bin/cld" }

// loginShell returns the command line of a login shell of the user's, which runs the arguments
// after it. The unit keeps the PATH cld setup restore runs with, where cld restore sessions finds
// tmux, and its TMUX_TMPDIR, XDG_STATE_HOME and CLD_IDLE_DAYS where set. So the shell starts from a
// clean environment, not the caller's: /etc/environment, as PAM gives a login, then bash's login
// files. With the user's bus, systemctl --user finds the user's systemd.
func (r Restore) loginShell(args ...string) (string, []string) {
	argv := append([]string{"-i", "HOME=" + r.Account.Home, "USER=" + r.Account.Name, "LOGNAME=" + r.Account.Name,
		"SHELL=/bin/bash"}, systemd.SessionEnv(r.Account.UID)...)
	argv = append(argv, "/bin/bash", "-c", `set -a && . /etc/environment && exec /bin/bash -l "$@"`, "bash")
	return "env", append(argv, args...)
}

// compare prints missing or differs where the unit is not what cld writes, in the lines that come
// from cld's environment and path, each quoted as cld quotes it. The rest of the unit is cld's own
// text, which may change with a newer cld: so a unit older than cld differs too, and setupRestore
// touches it. Its command is restore sessions where the cld installed has it. With an older cld,
// which exits 2 on help restore sessions as cld 0.13.0 does, the command is plain restore. $0 is
// cld.
const compare = `unit=$HOME/.config/systemd/user/cld-restore.service
quote() {
    local word=${1//\\/\\\\}
    word=${word//\"/\\\"}
    printf '"%s"' "${word//%/%%}"
}
want=$(
    for name in PATH TMUX_TMPDIR XDG_STATE_HOME CLD_IDLE_DAYS; do
        if [ "$name" = PATH ] || [ -n "${!name-}" ]; then
            printf 'Environment=%s\n' "$(quote "$name=${!name-}")"
        fi
    done
    cld=$(readlink -f "$0")
    command=restore
    if "$0" help restore sessions >/dev/null 2>&1; then
        command='restore sessions'
    fi
    printf 'ExecStart=%s %s\n' "$(quote "${cld//\$/\$\$}")" "$command"
)
if [ ! -f "$unit" ]; then
    echo missing
elif [ "$(grep -E '^(Environment|ExecStart)=' "$unit")" != "$want" ] || [ "$unit" -ot "$0" ]; then
    echo differs
fi
`

var (
	lingering = regexp.MustCompile(`(?m)^Lingering is on: `)
	// stale matches compare's verdict, apart from what a login shell prints.
	stale = regexp.MustCompile(`(?m)^(missing|differs)$`)
)

// setup runs cld setup restore, then touches the unit, so that compare finds it newer than cld.
const setup = `"$0" setup restore && touch "$HOME/.config/systemd/user/cld-restore.service"`

type setupRestore struct{ r Restore }

func (setupRestore) String() string { return "have the user's systemd restore cld's sessions" }

// Apply runs cld setup restore in the login shell. It exits 0 where lingering is off or unknown,
// and only says so, so it fails unless cld says lingering is on.
func (s setupRestore) Apply(ctx context.Context) error {
	name, args := s.r.loginShell("-c", setup, s.r.cld())
	out, err := s.r.Account.Output(ctx, s.r.M, name, args...)
	if err != nil {
		return err
	}
	if !lingering.Match(out) {
		return fmt.Errorf("cld setup restore did not say that lingering is on:\n%s", strings.TrimSpace(string(out)))
	}
	return nil
}

// Plan compares cld-restore.service with what cld writes. It also looks for the link that
// systemctl --user enable makes, as cld enables the unit at every run but reports only whether it
// wrote its file.
func (r Restore) Plan(ctx context.Context) ([]host.Change, error) {
	if !systemd.Booted(r.M) {
		r.Out.Say("Claude restore: skipped, systemd is not running")
		return nil, nil
	}
	if _, err := r.M.Stat(r.cld()); errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	name, args := r.loginShell("-c", compare, r.cld())
	out, err := r.Account.Output(ctx, r.M, name, args...)
	if err != nil {
		return nil, err
	}
	if stale.Match(out) {
		return []host.Change{setupRestore{r}}, nil
	}
	_, err = r.M.Stat(r.Account.Home + "/.config/systemd/user/default.target.wants/cld-restore.service")
	if errors.Is(err, fs.ErrNotExist) {
		return []host.Change{setupRestore{r}}, nil
	}
	return nil, err
}
