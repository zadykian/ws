package ikeward

import (
	"context"
	_ "embed" // sysctlFile
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/host"
)

const (
	sysctlPath = "/etc/sysctl.d/99-ikeward.conf"
	ipForward  = "/proc/sys/net/ipv4/ip_forward"
)

//go:embed 99-ikeward.conf
var sysctlFile []byte

// sysctlNow makes systemd-sysctl apply the file's value for this one key, as boot does, and
// nothing else. --strict makes it fail where /proc/sys is read-only, instead of going on as if the
// value were set. It plans nothing without systemd, as /proc/sys can be read-only in a container.
type sysctlNow struct{ m effect.Machine }

func (c sysctlNow) String() string { return "forward IPv4 now" }

func (c sysctlNow) Apply(ctx context.Context) error {
	_, err := c.m.RunRoot(ctx, "/usr/lib/systemd/systemd-sysctl", "--strict", "--prefix=net.ipv4.ip_forward")
	return err
}

// Plan plans the change where IPv4 forwarding is off.
func (c sysctlNow) Plan(context.Context) ([]host.Change, error) {
	if !host.Booted(c.m) {
		return nil, nil
	}
	data, err := c.m.ReadFile(ipForward)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(string(data)) == "1" {
		return nil, nil
	}
	return []host.Change{c}, nil
}

// sysctl returns the resources of IPv4 forwarding at boot and now, or the removal of the file
// where none is wanted.
func sysctl(m effect.Machine, want bool) []host.Resource {
	if !want {
		return []host.Resource{host.Absent{M: m, Path: sysctlPath}}
	}
	return []host.Resource{host.File{M: m, Path: sysctlPath, Content: sysctlFile, Mode: 0o644}, sysctlNow{m}}
}
