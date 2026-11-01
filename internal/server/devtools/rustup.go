package devtools

import (
	"context"
	"errors"
	"io/fs"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
)

// installRustup is rustup's own installer, which curl pipes into sh. --no-modify-path leaves PATH
// to the shell role's .profile, which sources rustup's env file.
const installRustup = "set -o pipefail && " +
	"curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | " +
	"sh -s -- -y --no-modify-path --default-toolchain stable"

// Rustup installs rustup and Rust's stable toolchain for User, in ~/.cargo and ~/.rustup. rustup
// updates itself and its toolchains, so it is installed only where it is missing.
type Rustup struct {
	M    effect.Machine
	User string
	Home string
}

type runRustup struct{ r Rustup }

func (c runRustup) String() string {
	return "install rustup and Rust's stable toolchain for " + c.r.User
}

// Apply runs the installer as the user. HOME is set, as without the user's login it would be the
// caller's. root needs no switch of user, and a container has no sudo.
func (c runRustup) Apply(ctx context.Context) error {
	r := c.r
	run := []string{"env", "HOME=" + r.Home, "bash", "-c", installRustup}
	if r.User == "root" {
		return r.M.Pass(ctx, false, run[0], run[1:]...)
	}
	return r.M.Pass(ctx, true, "runuser", append([]string{"-u", r.User, "--"}, run...)...)
}

// Plan installs rustup where ~/.cargo/bin/rustup is missing.
func (r Rustup) Plan(context.Context) ([]host.Change, error) {
	_, err := r.M.Stat(r.Home + "/.cargo/bin/rustup")
	if errors.Is(err, fs.ErrNotExist) {
		return []host.Change{runRustup{r}}, nil
	}
	return nil, err
}
