// Package wgdev runs wireguard-go's device in the daemon's process, on a utun, with its UAPI socket.
package wgdev

import (
	"fmt"
	"io"
	"log"
	"regexp"

	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/tun"
)

var utun = regexp.MustCompile(`^utun[0-9]{1,3}$`)

// Tunnel is a device on a tun.
type Tunnel struct {
	dev  *device.Device
	name string
}

// New puts a device on t, which is named name. The name must be a utun's.
func New(t tun.Device, name string, logOut io.Writer) (*Tunnel, error) {
	if !utun.MatchString(name) {
		return nil, fmt.Errorf("the interface, %q, is no utun", name)
	}
	logger := device.NewLogger(device.LogLevelError, "")
	logger.Errorf = log.New(logOut, "", 0).Printf
	return &Tunnel{dev: device.NewDevice(t, conn.NewDefaultBind(), logger), name: name}, nil
}

// Name is the utun's name.
func (t *Tunnel) Name() string { return t.name }

// IpcGet returns the device's state in UAPI text.
func (t *Tunnel) IpcGet() (string, error) { return t.dev.IpcGet() }

// IpcSet sets the device from UAPI text.
func (t *Tunnel) IpcSet(text string) error { return t.dev.IpcSet(text) }

// Done is closed where the device has closed.
func (t *Tunnel) Done() <-chan struct{} { return t.dev.Wait() }

// Close closes the device, which takes the utun with it.
func (t *Tunnel) Close() {
	t.dev.Close()
}
