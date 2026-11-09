package ikeward

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/host"
	"github.com/zadykian/ws/internal/secrets"
	"github.com/zadykian/ws/internal/server/values"
)

// exitChanged is what ikeward install --exit-code exits with where install would change anything,
// and 0 where not.
const exitChanged = 4

// Install runs ikeward install from the profile of the secrets, where its dry run says it would
// change anything. A change is to install a package, write or remove a file, or take a step. So a
// new version, a new value or a file of ikeward's edited by hand has it run again. So has an
// install cut short, or a unit of ikeward's found stopped. Without systemd booted, or a profile in the secrets,
// it says so and plans nothing.
type Install struct {
	M       effect.Machine
	Out     effect.Out
	Values  values.Ikeward
	Secrets secrets.Secrets
}

func (i Install) profile() profile {
	return profile{m: i.M, name: i.Values.Profile, files: i.Secrets.File}
}

// args are install's arguments, from the decrypted files and the remote network's values. --probe
// stays ikeward's default, the first domain. The password goes on standard input, never here.
func (i Install) args(p profile) []string {
	args := []string{"install", "--p12", p.base() + ".p12", "--sswan", p.base() + ".sswan", "--dns", i.Values.DNS}
	for _, d := range i.Values.Domains {
		args = append(args, "--domain="+d)
	}
	for _, f := range i.Values.Floors {
		args = append(args, "--floor="+f)
	}
	return args
}

// run runs install with the flags, on the password. It returns install's exit code with its output,
// and an error for any other failure, which shows ikeward's error: it holds no key and no password.
func (i Install) run(ctx context.Context, p profile, flags ...string) (int, string, error) {
	out, err := i.M.RunInput(ctx, []byte(i.Secrets.Password.VPNP12), binary,
		append(i.args(p), append(flags, "--exit-code")...)...)
	var ee *exec.ExitError
	switch {
	case err == nil:
		return 0, string(out), nil
	case errors.As(err, &ee) && ee.ExitCode() == exitChanged:
		return exitChanged, string(out), nil
	}
	return 0, "", fmt.Errorf("ikeward install: %w", err)
}

type installNow struct {
	i      Install
	dryRun string
}

func (c installNow) String() string {
	return "run ikeward install, which would change:\n" + strings.TrimRight(c.dryRun, "\n")
}

func (c installNow) Apply(ctx context.Context) error {
	p := c.i.profile()
	return p.with(ctx, func() error {
		_, _, err := c.i.run(ctx, p)
		return err
	})
}

// Plan asks install what it would change, on the files of the run.
func (i Install) Plan(ctx context.Context) ([]host.Change, error) {
	if !host.Booted(i.M) {
		i.Out.Say("ikeward install: skipped, systemd is not running")
		return nil, nil
	}
	p := i.profile()
	if !p.present() {
		i.Out.Say("ikeward install: the secrets hold no profile, so the VPN stays as it is")
		return nil, nil
	}
	var changes []host.Change
	err := p.with(ctx, func() error {
		code, dry, err := i.run(ctx, p, "--dry-run")
		if code == exitChanged {
			changes = []host.Change{installNow{i, dry}}
		}
		return err
	})
	return changes, err
}
