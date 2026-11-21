package cli

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine"
	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/fetch"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/internal/setup"
)

// rootRoles is where mise's bootstrap runs: the server's roles, in the tree.
var rootRoles = path.Join(setup.TreeDir, "roles", "server")

// cloneDir is the clone of ws's secrets, which the task secrets makes on its first run.
const cloneDir = "/var/lib/ws/secrets"

// aboveTree lists the directories above the tree and mise's system plugin, which mise ignores the
// plugin below where anyone but root may write one.
var aboveTree = []string{"/", "/usr", "/usr/share", "/opt"}

// setupConfig is what ws server setup takes from the build, which the tests replace.
type setupConfig struct {
	// Mise is the release of mise that setup installs.
	Mise setup.MiseRelease
	// Repo is where the tree's tag is fetched from.
	Repo string
	// Enter readies the process. It sets umask 022, so that the tree stays root's alone for the
	// guard of the next run. It changes to /, as mise loads the config of the directory it runs
	// in, and a checkout there is not the tree. Nil does neither.
	Enter func() error
}

// liveSetup is the config of a run on the machine.
func liveSetup() setupConfig {
	return setupConfig{Mise: setup.PinnedMise, Repo: setup.Repo, Enter: func() error {
		syscall.Umask(0o022)
		return os.Chdir("/")
	}}
}

// newSetupCmd returns ws server setup, which sets the server up as root. It checks the system and
// installs mise at the release that ws pins. It checks the tree of this ws's release out behind
// the guard, links the plugin and runs mise's bootstrap.
func newSetupCmd(p enginecli.Env, cfg setupConfig) *cobra.Command {
	c := &cobra.Command{
		Use:   "setup",
		Short: "set the server up, as root",
		Long: "It refuses a user but root, a system but Ubuntu 26.04, and a tree or a directory above it " +
			"that others may write, before any write. It installs mise at the version ws pins, checked " +
			"against its SHA-256, where mise --version differs, and writes " + setup.MisercPath + " and " +
			setup.ConfigPath + ". It checks this ws's release tag out in " + setup.TreeDir + ", cleans the tree, " +
			"links its plugin into mise's, and runs mise's bootstrap in the server's roles. " +
			"A second run changes nothing.\n\n" +
			"--secrets runs the task secrets in its place, which clones the secrets on its first run and runs " +
			"the bootstrap under them. Without it, setup runs the task where " + cloneDir + " stands.\n\n" +
			"A ws that names no release tag, devel included, is refused: pass --in-place to run the tree " +
			"in " + setup.TreeDir + " as it stands, after the guard.\n\n" +
			"--plan plans these steps, and mise's bootstrap, or the task where setup would run it, where none " +
			"of them is due. The task fetches the secrets first, and the plan changes nothing else. " +
			"It exits 0 where nothing would change and 2 where something would.",
		Args: cobra.NoArgs,
	}
	inPlace := c.Flags().Bool("in-place", false, "run the tree in "+setup.TreeDir+" as it stands, after the guard")
	plan := c.Flags().Bool("plan", false, "plan setup's steps, and mise's where none is due: exit 2 where a change is due, and change nothing")
	secrets := c.Flags().Bool("secrets", false, "run the task secrets, which clones the secrets on its first run")
	c.RunE = func(c *cobra.Command, _ []string) error {
		if p.Euid != 0 {
			return errors.New("run it as root")
		}
		if err := ubuntu(p.Machine); err != nil {
			return err
		}
		tag := ""
		if !*inPlace {
			var err error
			if tag, err = setup.ReleaseTag(p.Version); err != nil {
				return err
			}
		}
		if cfg.Enter != nil {
			if err := cfg.Enter(); err != nil {
				return err
			}
		}
		self, err := p.Self()
		if err != nil {
			return err
		}
		if err := checkValues(p.Machine); err != nil {
			return err
		}
		steps, err := setupSteps(c.Context(), p.Machine, cfg, tag)
		if err != nil {
			return err
		}
		term := effect.NewTerminal(c.ErrOrStderr())
		defer term.Flush()
		out := linesOf(term, c.CommandPath())
		if *plan {
			return planSetup(c.Context(), p.Machine, out, steps, self, *secrets)
		}
		if err := settle(c.Context(), out, steps, false); err != nil {
			return err
		}
		return runBootstrap(c.Context(), p.Machine, self, *secrets, nil)
	}
	return c
}

// runBootstrap runs mise's bootstrap, or the task secrets where secrets is set or the clone of the
// secrets stands, so that a later run stays under them. The task passes args to the bootstrap whole.
func runBootstrap(ctx context.Context, m effect.Machine, self string, secrets bool, args []string) error {
	if !secrets {
		_, err := m.Stat(cloneDir)
		switch {
		case err == nil:
			secrets = true
		case !errors.Is(err, fs.ErrNotExist):
			return err
		}
	}
	if !secrets {
		return runMise(ctx, m, self, append([]string{"bootstrap"}, args...)...)
	}
	if len(args) > 0 {
		args = append([]string{"--"}, args...)
	}
	return runMise(ctx, m, self, append([]string{"run", "secrets"}, args...)...)
}

// runMise runs mise in the server's roles, with WS_SELF naming this ws. The frame's check of the
// values runs that ws, which a first run has not installed in mise's path yet.
func runMise(ctx context.Context, m effect.Machine, self string, args ...string) error {
	argv := append([]string{"WS_SELF=" + self, setup.MisePath, "-C", rootRoles}, args...)
	return m.PassInput(ctx, nil, "env", argv...)
}

// setupSteps returns the resources of setup, in order. Each one plans before it writes. The guard
// comes first, as mise's config trusts the tree, and nothing is written before the tree passes. An
// empty tag takes the tree as it stands.
func setupSteps(ctx context.Context, m effect.Machine, cfg setupConfig, tag string) ([]resource.Resource, error) {
	target, err := miseTarget(ctx, m)
	if err != nil {
		return nil, err
	}
	holds := ""
	if tag == "" {
		holds = "mise.toml"
	}
	steps := []resource.Resource{
		setup.Guard{M: m, Dir: setup.TreeDir, Above: aboveTree, Holds: holds},
		setup.MiseBinary{M: m, Release: cfg.Mise, Target: target},
	}
	steps = append(steps, setup.SystemMise(m, "server")...)
	steps = append(steps, setup.Checkout{M: m, Dir: setup.TreeDir, Repo: cfg.Repo, Tag: tag, InPlace: tag == ""})
	return append(steps, setup.Plugin(m, setup.TreeDir, setup.PluginDir, 0)...), nil
}

// planSetup says each change that setup's steps would make, and exits 2 where there is one. Where
// there is none, it asks mise's bootstrap, or the task secrets where setup would run it, for its
// plan, whose exit 2 is setup's. The tree and mise are there to plan by only once the steps have
// run.
func planSetup(ctx context.Context, m effect.Machine, out effect.Out, steps []resource.Resource, self string, secrets bool) error {
	due := false
	for _, r := range steps {
		changes, err := r.Plan(ctx)
		if err != nil {
			return err
		}
		for _, c := range changes {
			out.Say("would %s", step.Headline(c))
			due = true
		}
	}
	if due {
		return engine.ExitStatus{Code: 2}
	}
	err := runBootstrap(ctx, m, self, secrets, []string{"plan", "--detailed-exitcode"})
	var exit interface{ ExitCode() int }
	if errors.As(err, &exit) && exit.ExitCode() == 2 {
		return engine.ExitStatus{Code: 2}
	}
	return err
}

// miseTarget names mise's binary for this machine by dpkg's architecture: ws sets up x86_64 and
// aarch64 alone.
func miseTarget(ctx context.Context, m effect.Machine) (string, error) {
	arch, err := fetch.Architecture(ctx, m)
	if err != nil {
		return "", err
	}
	switch arch {
	case "amd64":
		return "linux-x64", nil
	case "arm64":
		return "linux-arm64", nil
	}
	return "", fmt.Errorf("ws sets up Linux on x86_64 and aarch64, not on %s", arch)
}
