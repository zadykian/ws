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
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/local"
	"github.com/zadykian/ws/engine/core/resource"
	"github.com/zadykian/ws/macos/roles/tunnel"
	"github.com/zadykian/ws/macos/roles/tunnel/daemon"
	"github.com/zadykian/ws/macos/roles/tunnel/daemon/settings"
	"github.com/zadykian/ws/macos/roles/tunnel/daemon/uapi"
	"github.com/zadykian/ws/macos/roles/tunnel/daemon/wgdev"
	"github.com/zadykian/ws/macos/values"
)

// errNotRoot is the refusal of the commands that read root's files.
var errNotRoot = errors.New("run it as root, with sudo")

// watchWait is how long the watch of network changes waits before it starts notifyutil again.
const watchWait = 5 * time.Second

// statusWait is how long status waits for the device's socket.
const statusWait = 5 * time.Second

// newTunnelCmd returns ws macos tunnel.
func newTunnelCmd(p enginecli.Env, server *string) *cobra.Command {
	return enginecli.Group("tunnel", "the Mac's tunnel to ws",
		newTunnelRunCmd(p, server), newTunnelStatusCmd(p, server), newTunnelKeyCmd(p), newTunnelConfCmd(p), newTunnelLoadedCmd(p, server),
		newTunnelRoutesCmd(p, server))
}

// newTunnelRunCmd returns ws macos tunnel run.
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

// newTunnelStatusCmd returns ws macos tunnel status.
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

// newTunnelKeyCmd returns ws macos tunnel key.
func newTunnelKeyCmd(p enginecli.Env) *cobra.Command {
	return &cobra.Command{
		Use:   "key",
		Short: "make the Mac's key where none is, and print the public key, as root",
		Long: "The private key stays in root's file, which only root can write, so the key's unit runs this as root. " +
			"/etc/ws/tunnel.pub holds the public key, readable by all.",
		Args: cobra.NoArgs,
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
func tunnelRun(ctx context.Context, srv tunnel.Daemon, stdout io.Writer) error {
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
// stays in root's file, which only root can write, so the key's unit runs this as root.
func tunnelKey(ctx context.Context, p enginecli.Env) (string, error) {
	if p.Euid != 0 {
		return "", errNotRoot
	}
	return tunnel.MakeKey(ctx, p.Machine, rand.Reader)
}

// newTunnelConfCmd returns ws macos tunnel conf, which the unit of the daemons' files runs.
func newTunnelConfCmd(p enginecli.Env) *cobra.Command {
	return enginecli.Procedure("conf", "write each server's tunnel.conf, and take a gone server's daemon away",
		"Each server with WS_PUBLIC_KEY gets its directory under /etc/ws/tunnel, routes.d and tunnel.conf, "+
			"written through sudo and readable by all. A daemon whose server has no directory beside "+
			"values.yml is booted out, its files go, and the daemons left read their files again.",
		func(_ context.Context, out effect.Out) ([]resource.Resource, error) {
			servers, err := serversOf(p)
			if err != nil {
				return nil, err
			}
			return tunnel.Conf{Servers: servers, MacConf: values.Path(p.Getenv, p.Pwd), M: p.Machine, Out: out}.Resources(), nil
		})
}
