package ikeward

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/roles/server/values"
)

// binary is where the role's binary unit puts ikeward, and where its services, the routing's and
// strongSwan run it: install refuses to run as any other program.
const binary = "/usr/local/sbin/ikeward"

// exitChanged is what ikeward install --exit-code exits with where install would change anything,
// and 0 where not.
const exitChanged = 4

// Install runs ikeward install from the profile of the run's env, where its dry run says it would
// change anything. A change is to install a package, write or remove a file, or take a step. So a
// new version, a new value or a file of ikeward's edited by hand has it run again. So has an
// install cut short, or a unit of ikeward's found stopped. In a run without the secrets, it says
// so and plans nothing. InRun is ProfileFrom's set.
type Install struct {
	M       effect.Machine
	Out     effect.Out
	Values  values.Ikeward
	Profile Profile
	InRun   bool
}

// Ready fails where the run gives the secrets but not both of the profile's files, which install
// reads. A run without them leaves the VPN alone, so it passes. The error names the profile's
// path and the variables, and holds no value.
func (i Install) Ready() error {
	if i.Values.Profile == "" || !i.InRun || i.profile().present() {
		return nil
	}
	return fmt.Errorf("WS_IKEWARD_PROFILE is %s, and %s or %s is empty", i.Values.Profile, P12Var, SswanVar)
}

func (i Install) profile() profile {
	return profile{m: i.M, name: i.Values.Profile, files: i.Profile}
}

// args are install's arguments, from the profile's files and the remote network's values. --probe
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
	out, err := i.M.RunInput(ctx, []byte(i.Profile.Password), binary,
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
func (i Install) Plan(ctx context.Context) ([]resource.Change, error) {
	if !i.InRun {
		i.Out.Say("ikeward install: the run holds no secrets, so the VPN stays as it is")
		return nil, nil
	}
	if err := i.Ready(); err != nil {
		return nil, err
	}
	p := i.profile()
	var changes []resource.Change
	err := p.with(ctx, func() error {
		code, dry, err := i.run(ctx, p, "--dry-run")
		if code == exitChanged {
			changes = []resource.Change{installNow{i, dry}}
		}
		return err
	})
	return changes, err
}
