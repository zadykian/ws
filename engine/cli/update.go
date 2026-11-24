package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine"
	"github.com/zadykian/ws/engine/core/check"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/update"
)

// Setups is the setup that ws update ends in.
type Setups struct {
	// Pick returns the arguments of ws for the setup that this machine's side runs, or nil where
	// the machine has no side yet.
	Pick func(m effect.Machine) ([]string, error)
	// Names are the commands that Pick may choose, which ws update names where it chose none.
	Names []string
	// Fixed is set where Pick never chooses none.
	Fixed bool
}

// FixedSetup is the setup of a system with one side: ws ARGS, whatever the machine holds.
func FixedSetup(args ...string) Setups {
	return Setups{Names: []string{"ws " + strings.Join(args, " ")}, Fixed: true,
		Pick: func(effect.Machine) ([]string, error) { return args, nil }}
}

// Update returns ws update, which brings ws to a release, and ends in the setup that s picks for
// the machine's side, or in none where it picks none. Where root is set, ws runs as root, as on a
// server, and else as the user who asks sudo for the write, as on a Mac.
func Update(p Env, root bool, s Setups) *cobra.Command {
	none := ""
	if !s.Fixed {
		none = ", and none where it has no side yet"
	}
	c := &cobra.Command{
		Use:   "update",
		Short: "update ws to a release, then run the setup of this system",
		Long: "Downloads the latest release's ws for this system, or the one that --version names, " +
			"checks it against the release's ws.sha256 and that it runs and prints the release, then " +
			"renames it over " + update.File + ". A signal before the rename leaves the old ws. Then it " +
			"runs the setup of that ws that this machine's side has, " + strings.Join(s.Names, " or ") +
			none + ".\n\n" +
			"The release of " + update.File + " counts, as the units run that file, whichever ws runs " +
			"the update. Where that file is the release already, nothing is downloaded: only its setup " +
			"runs. The latest release is refused where that file is a later release, and --version " +
			"names the older one. A ws that is no release, such as a trial's build, or one that is " +
			"missing or does not run, is older than any release, so that an update ends the trial.",
		Args: cobra.NoArgs,
	}
	version := c.Flags().String("version", "", "the release to install, X.Y.Z or X.Y.Z-rc.N, in place of the latest")
	c.RunE = func(c *cobra.Command, _ []string) error {
		tag, err := releaseArg(*version)
		if err != nil {
			return engine.UsageError{Err: err}
		}
		if (p.Euid == 0) != root {
			if root {
				return errors.New("run it as root")
			}
			return errors.New("run it as your user, not root: it asks sudo for what needs root")
		}
		r, err := runUpdate(c, p, tag)
		if err != nil {
			return err
		}
		if _, err := io.WriteString(c.OutOrStdout(), r.Report()); err != nil {
			return err
		}
		args, err := s.Pick(p.Machine)
		if err != nil {
			return err
		}
		if args == nil {
			_, err = fmt.Fprintf(c.OutOrStdout(), "No side is set up here: run %s\n", strings.Join(s.Names, " or "))
			return err
		}
		return setupAfter(p, r, args)
	}
	return c
}

// releaseArg returns the tag that --version names: X.Y.Z or X.Y.Z-rc.N, with or without the v, or
// empty where none is given.
func releaseArg(version string) (string, error) {
	if version == "" {
		return "", nil
	}
	tag := "v" + strings.TrimPrefix(version, "v")
	if !check.Release(tag) {
		return "", fmt.Errorf("--version %q is no release: use X.Y.Z or X.Y.Z-rc.N", version)
	}
	return tag, nil
}

// runUpdate runs the update until it returns. A signal before the rename cancels it, and leaves
// this ws unchanged. One after it changes nothing.
func runUpdate(c *cobra.Command, p Env, tag string) (update.Result, error) {
	ctx, cancel := context.WithCancelCause(c.Context())
	defer cancel(nil)
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signals)
	go func() {
		select {
		case s := <-signals:
			cancel(engine.SignalExit{Sig: s.(syscall.Signal)}) //nolint:forcetypeassert // Notify's signals are syscall.Signal
		case <-ctx.Done():
		}
	}()
	u := update.Updater{M: p.Machine, URL: p.Releases, File: update.File, Tag: tag,
		Binary: "ws-" + p.GOOS + "-" + p.GOARCH, PID: p.Pid}
	r, err := u.Run(ctx)
	if err != nil {
		var cut engine.SignalExit
		if errors.As(context.Cause(ctx), &cut) {
			return r, cut
		}
		return r, err
	}
	return r, nil
}

// setupAfter replaces this process with ws ARGS of the ws that the update leaves, the file that the
// units run.
func setupAfter(p Env, r update.Result, args []string) error {
	return p.Machine.Exec(r.File, args, nil)
}
