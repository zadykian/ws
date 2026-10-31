// Package wgdev runs wireguard-go's device in the daemon's process, on a utun, with its UAPI socket.
package wgdev

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"regexp"

	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/ipc"
	"golang.zx2c4.com/wireguard/tun"
)

// mtu is wireguard-go's own default for a tunnel.
const mtu = device.DefaultMTU

var utun = regexp.MustCompile(`^utun[0-9]{1,3}$`)

// Tunnel is a device on a tun, and the socket that wg-style tools read it through.
type Tunnel struct {
	dev      *device.Device
	name     string
	listener net.Listener
	logger   *device.Logger
}

// Open makes a utun and a device on it, and serves the device's UAPI at /var/run/wireguard/NAME.sock
// as wireguard-go's own main does. Only root may make a utun. logOut gets the device's messages.
func Open(logOut io.Writer) (*Tunnel, error) {
	t, err := tun.CreateTUN("utun", mtu)
	if err != nil {
		return nil, fmt.Errorf("cannot make a utun: %w", err)
	}
	name, err := t.Name()
	if err != nil {
		_ = t.Close()
		return nil, err
	}
	tn, err := New(t, name, logOut)
	if err != nil {
		_ = t.Close()
		return nil, err
	}
	if err := tn.serve(); err != nil {
		tn.Close()
		return nil, err
	}
	return tn, nil
}

// New puts a device on t, which is named name. The name must be a utun's.
func New(t tun.Device, name string, logOut io.Writer) (*Tunnel, error) {
	if !utun.MatchString(name) {
		return nil, fmt.Errorf("the interface, %q, is no utun", name)
	}
	logger := device.NewLogger(device.LogLevelError, "")
	logger.Errorf = log.New(logOut, "", 0).Printf
	return &Tunnel{dev: device.NewDevice(t, conn.NewDefaultBind(), logger), name: name, logger: logger}, nil
}

// serve opens the UAPI socket and hands each connection to the device.
func (t *Tunnel) serve() error {
	file, err := ipc.UAPIOpen(t.name)
	if err != nil {
		return fmt.Errorf("cannot open the UAPI socket: %w", err)
	}
	t.listener, err = ipc.UAPIListen(t.name, file)
	if err != nil {
		_ = file.Close()
		return fmt.Errorf("cannot listen on the UAPI socket: %w", err)
	}
	go t.accept(t.listener)
	return nil
}

// accept hands each connection to the device. Where the listener fails, as it does when something
// removes the socket's file, the device closes, as in wireguard-go's own main, so that the daemon
// sees it exit. A close of the listener by Close ends it quietly.
func (t *Tunnel) accept(l net.Listener) {
	for {
		c, err := l.Accept()
		if err != nil {
			if !errors.Is(err, net.ErrClosed) {
				t.logger.Errorf("UAPI listener failed: %v", err)
				t.dev.Close()
			}
			return
		}
		go t.dev.IpcHandle(c)
	}
}

// Name is the utun's name.
func (t *Tunnel) Name() string { return t.name }

// IpcGet returns the device's state in UAPI text.
func (t *Tunnel) IpcGet() (string, error) { return t.dev.IpcGet() }

// IpcSet sets the device from UAPI text.
func (t *Tunnel) IpcSet(text string) error { return t.dev.IpcSet(text) }

// Done is closed where the device has closed.
func (t *Tunnel) Done() <-chan struct{} { return t.dev.Wait() }

// Close closes the socket and the device, which takes the utun with it.
func (t *Tunnel) Close() {
	if t.listener != nil {
		_ = t.listener.Close()
	}
	t.dev.Close()
}
