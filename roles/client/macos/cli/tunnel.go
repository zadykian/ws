package cli

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect/local"
	"github.com/zadykian/ws/roles/client/macos/internal/launchd"
	"github.com/zadykian/ws/roles/client/macos/tunnel/daemon"
	"github.com/zadykian/ws/roles/client/macos/tunnel/daemon/settings"
	"github.com/zadykian/ws/roles/client/macos/tunnel/daemon/uapi"
	"github.com/zadykian/ws/roles/client/macos/tunnel/daemon/wgdev"
)

// errNotRoot is the refusal of the commands that read root's files.
var errNotRoot = errors.New("run it as root, with sudo")

// watchWait is how long the watch of network changes waits before it starts notifyutil again.
const watchWait = 5 * time.Second

// statusWait is how long status waits for the device's socket.
const statusWait = 5 * time.Second

// newTunnelCmd returns ws client macos tunnel.
func newTunnelCmd(p enginecli.Env, server *string) *cobra.Command {
	return enginecli.Group("tunnel", "the Mac's tunnel to ws",
		newTunnelRunCmd(p, server), newTunnelStatusCmd(p, server), newTunnelKeyCmd(p), newTunnelLoadedCmd(p, server),
		newTunnelRoutesCmd(p))
}

// newTunnelRunCmd returns ws client macos tunnel run.
func newTunnelRunCmd(p enginecli.Env, server *string) *cobra.Command {
	return &cobra.Command{
		Use:   "run",
		Short: "keep the tunnel to a server up, as root",
		Long:  "The daemon logs to stdout, launchd's log file. SIGHUP reads the settings again, and SIGTERM stops it.",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			d, err := daemonOf(p.Machine, *server)
			if err != nil {
				return err
			}
			return tunnelRun(c.Context(), d, c.OutOrStdout())
		},
	}
}

// newTunnelStatusCmd returns ws client macos tunnel status.
func newTunnelStatusCmd(p enginecli.Env, server *string) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "print the way in use and the peer, as root",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			text, err := tunnelStatus(c.Context(), p, *server)
			if err != nil {
				return err
			}
			_, err = io.WriteString(c.OutOrStdout(), text)
			return err
		},
	}
}

// newTunnelKeyCmd returns ws client macos tunnel key.
func newTunnelKeyCmd(p enginecli.Env) *cobra.Command {
	return &cobra.Command{
		Use:   "key",
		Short: "make the Mac's key where none is, and print the public key, as root",
		Long:  "The private key stays in root's file, which only root can write, so setup runs this through sudo.",
		Args:  cobra.NoArgs,
		RunE: func(c *cobra.Command, _ []string) error {
			public, err := tunnelKey(c.Context(), p)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(c.OutOrStdout(), public)
			return err
		},
	}
}

// tunnelRun keeps the tunnel up as root: the device on a utun, and the daemon's loop, which logs to
// stdout, launchd's log file. SIGHUP reads the settings again, and SIGTERM stops.
func tunnelRun(ctx context.Context, srv launchd.Daemon, stdout io.Writer) error {
	// Subscribe before anything else: launchd gives each signal its default action, which ends the
	// daemon on a SIGHUP.
	sigs := make(chan os.Signal, 4)
	signal.Notify(sigs, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGINT)
	tn, err := wgdev.Open(stdout)
	if err != nil {
		return err
	}
	defer tn.Close()
	d := daemon.New(local.OSMachine{}, tn, tn.Name(), daemonPaths(srv), stdout)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go d.Signals(ctx, sigs)
	go d.Watch(ctx, watchWait, daemon.Child(daemon.NetworkWatch...))
	go d.Exit(ctx, tn.Done())
	return d.Run(ctx)
}

// tunnelStatus returns the way in use and the peer's handshake and bytes. The state file and the
// device's socket are root's, so it needs root, as ws-tunnel status did.
func tunnelStatus(ctx context.Context, p enginecli.Env, server string) (string, error) {
	if p.Euid != 0 {
		return "", errNotRoot
	}
	d, err := daemonOf(p.Machine, server)
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(ctx, statusWait)
	defer cancel()
	return settings.Status(ctx, p.Machine, uapi.Get, daemonPaths(d), time.Now())
}

// tunnelKey makes the Mac's private key where none is, and returns the public key. The private key
// stays in root's file, which only root can write, so setup runs this through sudo.
func tunnelKey(ctx context.Context, p enginecli.Env) (string, error) {
	if p.Euid != 0 {
		return "", errNotRoot
	}
	return settings.Key(ctx, p.Machine, rand.Reader, launchd.Dir+"/private.key")
}
