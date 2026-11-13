package devtools

import (
	"context"
	"errors"
	"io/fs"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/account"
)

// installRustup is rustup's own installer, which curl pipes into sh. `--no-modify-path` leaves PATH
// to the shell role's .profile, which sources rustup's env file.
const installRustup = "set -o pipefail && " +
	"curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | " +
	"sh -s -- -y --no-modify-path --default-toolchain stable"

// Rustup installs rustup and Rust's stable toolchain for User, in ~/.cargo and ~/.rustup. rustup
// updates itself and its toolchains, so the step installs it only where missing.
type Rustup struct {
	M    effect.Machine
	User string
	Home string
}

type runRustup struct{ r Rustup }

func (c runRustup) String() string {
	return "install rustup and Rust's stable toolchain for " + c.r.User
}

// Apply runs the installer as the user, with the user's HOME.
func (c runRustup) Apply(ctx context.Context) error {
	r := c.r
	return account.User{Name: r.User, Home: r.Home}.Pass(ctx, r.M, "bash", "-c", installRustup)
}

// Plan installs rustup where ~/.cargo/bin/rustup is missing.
func (r Rustup) Plan(context.Context) ([]resource.Change, error) {
	_, err := r.M.Stat(r.Home + "/.cargo/bin/rustup")
	if errors.Is(err, fs.ErrNotExist) {
		return []resource.Change{runRustup{r}}, nil
	}
	return nil, err
}
