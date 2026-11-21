package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine"
	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/engine/core/resource/file"
	"github.com/zadykian/ws/roles/client/values"
	"github.com/zadykian/ws/roles/internal/setup"
)

const (
	// miseTarget names mise's release file for the Mac: ws sets up macOS on arm64 alone.
	miseTarget = "macos-arm64"
	// macPluginDir is mise's system directory of plugins on macOS, which mise loads a plugin from as
	// it stands.
	macPluginDir = "/usr/local/share/mise/plugins"
)

// clientRoles is where mise's bootstrap runs: the Mac's roles, in the tree.
var clientRoles = path.Join(setup.TreeDir, "roles", "client")

// macDirs are the directories that setup writes below, which a Mac may lack. Setup makes each
// one that lacks, root's, at 755, in order.
var macDirs = []string{"/usr/local", "/usr/local/bin", "/usr/local/share", "/opt"}

// aboveTree lists the directories above the tree and mise's system plugin, which mise ignores the
// plugin below where anyone but root or the user may write one.
var aboveTree = []string{"/", "/usr", "/usr/local", "/usr/local/share", "/opt"}

// setupConfig is what ws client setup takes from the build, which the tests replace.
type setupConfig struct {
	// Mise is the release of mise that setup installs.
	Mise setup.MiseRelease
	// Repo is where the tree's tag is fetched from.
	Repo string
	// Poll is the wait between two looks for the command line tools.
	Poll time.Duration
	// Enter readies the process. It sets umask 022, so that the tree that sudo's git makes is
	// readable by the user's mise. It changes to /, as mise loads the config of the directory it runs
	// in, and a checkout there is not the tree. Nil does neither.
	Enter func() error
}

// liveSetup is the config of a run on the Mac.
func liveSetup() setupConfig {
	return setupConfig{Mise: setup.PinnedMise, Repo: setup.Repo, Poll: 3 * time.Second, Enter: func() error {
		syscall.Umask(0o022)
		return os.Chdir("/")
	}}
}

// newSetupCmd returns ws client setup, which sets the Mac up as its user, and asks sudo for the
// writes that root's paths need. It installs the command line tools and mise at the release that ws
// pins. It checks the tree of this ws's release out behind the guard, links the plugin and runs
// mise's bootstrap.
func newSetupCmd(p enginecli.Env, server *string, cfg setupConfig) *cobra.Command {
	c := &cobra.Command{
		Use:   "setup",
		Short: "set the Mac up, as its user",
		Long: "It refuses root, a system but macOS on arm64, and bad values, before any write. It refuses a tree " +
			"or a directory above it that others may write. It installs the command line tools where " +
			"xcode-select -p finds none: Apple's dialog opens, and setup waits until the tools stand, which " +
			"Ctrl-C stops. It installs mise at the version ws pins, checked against its SHA-256, where mise " +
			"--version differs, and writes " + setup.MisercPath + " and " + setup.ConfigPath + ". It checks this ws's " +
			"release tag out in " + setup.TreeDir + ", cleans the tree, links its plugin into mise's, and runs mise's " +
			"bootstrap in the Mac's roles. Where values.yml names no WS_IDENTITY_PUB_KEY, it installs Secretive by mise's " +
			"pin first, checks it by spctl, opens it and waits for a P-256 key made in it, which it names in values.yml " +
			"for ws's login and git's signing, and takes no key that was there before: --key NAME names one. It asks sudo for what root's paths need, and for no more where a " +
			"run changes nothing, which a second run does.\n\n" +
			"A ws that names no release tag, devel included, is refused: pass --in-place to run the tree " +
			"in " + setup.TreeDir + " as it stands, after the guard.\n\n" +
			"--plan plans these steps, and mise's bootstrap where none of them is due. Where the command " +
			"line tools are due, it plans no more, as git needs them. It exits 0 where nothing would " +
			"change and 2 where something would.",
		Args: cobra.NoArgs,
	}
	inPlace := c.Flags().Bool("in-place", false, "run the tree in "+setup.TreeDir+" as it stands, after the guard")
	pick := c.Flags().String("key", "", "take this key of Secretive's for ws's login and git's signing, by its NAME or fingerprint")
	plan := c.Flags().Bool("plan", false, "plan setup's steps, and mise's where none is due: exit 2 where a change is due, and change nothing")
	c.RunE = func(c *cobra.Command, _ []string) error {
		if p.Euid == 0 {
			return errors.New("run it as your user, not root: it asks sudo for what root's paths need")
		}
		if *server != "" {
			return errors.New("setup takes no --server: it sets the Mac up for every server")
		}
		if err := macOnARM(c.Context(), p.Machine); err != nil {
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
		file := values.Path(p.Getenv, p.Pwd)
		if err := values.Check(p.Machine, file); err != nil {
			return err
		}
		conf, err := values.LoadMac(p.Machine, file)
		if err != nil {
			return err
		}
		if *pick != "" && conf.IdentityPubKey != "" {
			return fmt.Errorf("%s names WS_IDENTITY_PUB_KEY already, so setup takes no --key", file)
		}
		term := effect.NewTerminal(c.ErrOrStderr())
		defer term.Flush()
		out := linesOf(term, c.CommandPath())
		first, rest := setupSteps(p.Machine, out, cfg, tag)
		rest = append(rest, secretiveKey{M: p.Machine, Out: out, Home: p.Getenv("HOME"), Values: file, Conf: conf,
			Pick: *pick, Poll: cfg.Poll})
		if *plan {
			return planSetup(c.Context(), p.Machine, out, first, rest)
		}
		if err := enginecli.Settle(c.Context(), out, slices.Concat(first, rest), false); err != nil {
			return err
		}
		return runBootstrap(c.Context(), p.Machine)
	}
	return c
}

// macOnARM refuses a machine but a Mac on arm64, the one that the roles and the pin of mise are
// for.
func macOnARM(ctx context.Context, m effect.Machine) error {
	out, err := m.Run(ctx, "uname", "-sm")
	if err != nil {
		return err
	}
	if got := strings.TrimSpace(string(out)); got != "Darwin arm64" {
		return fmt.Errorf("ws sets up macOS on arm64 alone, not %q", got)
	}
	return nil
}

// setupSteps returns the resources of setup, in two parts. The first, the guard and the command line
// tools, plans and runs before the rest, as the rest runs git, which the tools give. The guard
// comes first, as mise's config trusts the tree, and nothing is written before the tree passes. An
// empty tag takes the tree as it stands.
func setupSteps(m effect.Machine, out effect.Out, cfg setupConfig, tag string) (first, rest []resource.Resource) {
	holds := ""
	if tag == "" {
		holds = "mise.toml"
	}
	first = []resource.Resource{
		setup.Guard{M: m, Dir: setup.TreeDir, Above: aboveTree, Holds: holds},
		commandLineTools{M: m, Out: out, Poll: cfg.Poll},
	}
	for _, dir := range macDirs {
		rest = append(rest, file.Dir{M: m, Path: dir, Mode: 0o755, CreateOnly: true})
	}
	rest = append(rest, setup.MiseBinary{M: m, Release: cfg.Mise, Target: miseTarget})
	rest = append(rest, setup.SystemMise(m, "mac")...)
	rest = append(rest, setup.Checkout{M: m, Dir: setup.TreeDir, Repo: cfg.Repo, Tag: tag, InPlace: tag == "", AsUser: true})
	return first, append(rest, setup.Plugin(m, setup.TreeDir, macPluginDir, 0)...)
}

// planSetup says each change that setup's steps would make, and exits 2 where there is one. The
// steps after the command line tools plan once the tools stand. Where there is none, it asks mise's
// bootstrap for its plan, whose exit 2 is setup's.
func planSetup(ctx context.Context, m effect.Machine, out effect.Out, first, rest []resource.Resource) error {
	for _, steps := range [][]resource.Resource{first, rest} {
		due, err := setup.PlanSteps(ctx, out, steps)
		if err != nil {
			return err
		}
		if due {
			return engine.ExitStatus{Code: 2}
		}
	}
	err := runMise(ctx, m, "bootstrap", "plan", "--detailed-exitcode")
	var exit interface{ ExitCode() int }
	if errors.As(err, &exit) && exit.ExitCode() == 2 {
		return engine.ExitStatus{Code: 2}
	}
	return err
}

// runBootstrap runs mise's bootstrap in the Mac's roles.
func runBootstrap(ctx context.Context, m effect.Machine) error {
	return runMise(ctx, m, "bootstrap")
}

// runMise runs the mise that setup installs, in the Mac's roles, as the user.
func runMise(ctx context.Context, m effect.Machine, args ...string) error {
	return m.PassInput(ctx, nil, setup.MisePath, append([]string{"-C", clientRoles}, args...)...)
}
