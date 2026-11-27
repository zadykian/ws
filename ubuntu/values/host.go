package values

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"

	"go.yaml.in/yaml/v3"

	"github.com/zadykian/ws/engine/core/effect"
)

// Host is every key of a host's file. A role that runs on the guest and the host reads the same
// section by the same name, so a section is the guest's. Fans, Cable and Incus are the host's own:
// the roles that read them run on the host alone. Tunnel is the host's own tunnel, wg0.
type Host struct {
	Base   Base   `yaml:",inline"`
	Docker Docker `yaml:",inline"`
	Fans   Fans   `yaml:",inline"`
	Cable  Cable  `yaml:",inline"`
	Signoz Signoz `yaml:",inline"`
	Incus  Incus  `yaml:",inline"`
	Tunnel Tunnel `yaml:",inline"`
}

// DefaultHost returns each key's default value, the guest's for a section it shares.
func DefaultHost() Host {
	d := Default()
	return Host{
		Base: d.Base, Docker: d.Docker, Fans: defaultFans(), Cable: Cable{}, Signoz: d.Signoz, Incus: defaultIncus(), Tunnel: d.Tunnel,
	}
}

// ParseHost returns the values that data gives over DefaultHost. It refuses a key that no role of
// the host knows, a key twice, and a value that its role cannot set up. Empty data gives
// DefaultHost.
func ParseHost(data []byte) (Host, error) {
	h, err := decodeHost(data)
	if err != nil {
		return Host{}, err
	}
	for _, check := range []func() error{h.Base.check, h.Docker.check, h.Cable.check, h.Incus.check, h.Tunnel.check} {
		if err := check(); err != nil {
			return Host{}, err
		}
	}
	return h, nil
}

// decodeHost is ParseHost without the checks, for a part of a file.
func decodeHost(data []byte) (Host, error) {
	h := DefaultHost()
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&h); err != nil && !errors.Is(err, io.EOF) {
		return Host{}, err
	}
	if err := h.Tunnel.keepNull(data); err != nil {
		return Host{}, err
	}
	h.Tunnel.derive()
	return h, nil
}

// LoadHost reads the host's values at path on m, and resolves each guest from its own values. A
// file that does not exist gives DefaultHost. A guest's file that does not read or names no slot
// fails nothing here: the guest holds the reason, and GuestTunnels asks for it. The self-contained
// mode fails the load, as no guest under a host may turn it on.
func LoadHost(m effect.Machine, path string) (Host, error) {
	data, err := m.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return Host{}, err
	}
	h, err := ParseHost(data)
	if err != nil {
		return Host{}, fmt.Errorf("%s: %w", path, err)
	}
	for i, g := range h.Incus.Guests {
		g = g.Load(m)
		if errors.Is(g.noTunnel, errSelfContained) {
			return Host{}, fmt.Errorf("%s: %w", path, g.noTunnel)
		}
		h.Incus.Guests[i] = g
	}
	return h, nil
}
