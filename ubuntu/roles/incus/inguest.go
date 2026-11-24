package incus

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/zadykian/ws/engine/core/effect"
)

// wsInGuest is where the host puts ws's binary in a guest: the path of a host's own.
const wsInGuest = "/usr/local/bin/ws"

// A script that runs in a guest by sh -c, with its path as $1. Each exits 3 where the path is
// missing, which no other failure of the command uses.
const (
	// sumScript prints the SHA-256 of a file.
	sumScript = `test -e "$1" || exit 3; sha256sum -- "$1"`
	// planScript plans the guest's own setup, with the flags after the path of ws.
	planScript = `test -x "$1" || exit 3; ws=$1; shift; exec "$ws" ubuntu guest setup --plan "$@"`
	// readyScript waits for the boot to end, then for cloud-init, whose seed names the guest, makes
	// its host keys and writes its netplan. Its status is cloud-init's.
	readyScript = `systemctl is-system-running --wait >/dev/null 2>&1; cloud-init status --wait >/dev/null`
)

// missing is the exit status of a script whose path is missing.
const missing = 3

// A guest's root runs what ws asks of it, so each exec is bounded: one that hangs would stall the
// guests and the units after it. readyTimeout covers a first boot, cloud-init and its packages.
var (
	readyTimeout = 15 * time.Minute
	probeTimeout = time.Minute
	planTimeout  = 15 * time.Minute
	setupTimeout = time.Hour
)

// shown is an error that a terminal may print. A guest's root controls what the error quotes of its
// standard error, so the text keeps no control character.
type shown struct{ err error }

func (e shown) Error() string { return printable(e.err.Error()) }
func (e shown) Unwrap() error { return e.err }

// fromGuest returns err as an error that a terminal may print.
func fromGuest(err error) error {
	if err == nil {
		return nil
	}
	return shown{err}
}

// printable drops the control characters of s but the tab: an escape sequence from a guest, such as
// one that writes the operator's clipboard, then prints as plain text.
func printable(s string) string {
	return strings.Map(func(r rune) rune {
		if r != '\t' && unicode.IsControl(r) {
			return -1
		}
		return r
	}, s)
}

// execIn runs the script in the guest, and stops it after limit.
func execIn(ctx context.Context, m effect.Machine, limit time.Duration, name, script string, args ...string) ([]byte, error) {
	bounded, cancel := context.WithTimeout(ctx, limit)
	defer cancel()
	out, err := m.Run(bounded, "incus", inGuest(name, script, args...)...)
	if err != nil && ctx.Err() == nil && bounded.Err() != nil {
		return out, fmt.Errorf("the guest %s did not answer in %s, so ws stopped waiting: %w", name, limit, err)
	}
	return out, err
}

// inGuest returns the arguments of incus that run the script in the guest, with args after it as
// $1 on.
func inGuest(name, script string, args ...string) []string {
	return append([]string{"exec", name, "--", "sh", "-c", script, "sh"}, args...)
}

// exitCode returns the exit status of a command's error, where it has one.
func exitCode(err error) (int, bool) {
	var exit interface{ ExitCode() int }
	if errors.As(err, &exit) {
		return exit.ExitCode(), true
	}
	return 0, false
}

// running reports whether the guest is an instance that Incus runs.
func running(ctx context.Context, m effect.Machine, name string) (bool, error) {
	all, err := Instances(ctx, m)
	if err != nil {
		return false, err
	}
	for _, i := range all {
		if i.Name == name {
			return i.Status == "Running", nil
		}
	}
	return false, nil
}

// waitReady returns where the guest's boot and cloud-init are over. A cloud-init that ended with a
// recoverable error, status 2, is over too. A first boot is not, so the plan of a guest's setup
// waits for it, as a plan before it would read a guest with no network.
func waitReady(ctx context.Context, m effect.Machine, name string) error {
	_, err := execIn(ctx, m, readyTimeout, name, readyScript)
	if code, ok := exitCode(err); ok && code == 2 {
		return nil
	}
	if err != nil {
		return fmt.Errorf("the guest %s is not ready, as cloud-init failed in it: %w", name, fromGuest(err))
	}
	return nil
}
