package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/internal/update"
)

// runningFile returns the file this ws runs from, the file a symbolic link leads to.
func runningFile() (string, error) {
	file, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.EvalSymlinks(file)
}

// newUpdateCmd returns ws update, which replaces this ws and changes nothing else, as cld update
// does for cld. It runs on any system that ws is released for, as any user who may write the file.
func newUpdateCmd(p process) *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "update ws to the latest release",
		Long: "Downloads the latest release's ws for this system, checks it against the release's " +
			"ws.sha256 and that it runs, then replaces the file ws runs from, the file a symbolic " +
			"link leads to.\n\nWhere ws is the latest release already, or newer, nothing changes. " +
			"A ws built from a clone is refused.",
		Args: cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error { return runUpdate(c, p) },
	}
}

// runUpdate runs the update until it returns. A signal before the replace cancels it, and leaves
// this ws unchanged. One after the replace changes nothing.
func runUpdate(c *cobra.Command, p process) error {
	file, err := p.self()
	if err != nil {
		return fmt.Errorf("finding the file ws runs from: %w", err)
	}
	ctx, cancel := context.WithCancelCause(c.Context())
	defer cancel(nil)
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signals)
	go func() {
		select {
		case s := <-signals:
			cancel(signalExit{s.(syscall.Signal)}) //nolint:forcetypeassert // Notify's signals are syscall.Signal
		case <-ctx.Done():
		}
	}()
	u := update.Updater{M: p.machine, Version: p.version, URL: p.releases, File: file,
		Binary: "ws-" + p.goos + "-" + p.goarch, PID: p.pid}
	r, err := u.Run(ctx)
	if err != nil {
		var cut signalExit
		if errors.As(context.Cause(ctx), &cut) {
			return cut
		}
		return err
	}
	_, err = io.WriteString(c.OutOrStdout(), r.Report())
	return err
}
