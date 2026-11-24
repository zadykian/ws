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
// the roles that read them run on the host alone.
type Host struct {
	Base   Base   `yaml:",inline"`
	Docker Docker `yaml:",inline"`
	Fans   Fans   `yaml:",inline"`
	Cable  Cable  `yaml:",inline"`
	Signoz Signoz `yaml:",inline"`
	Incus  Incus  `yaml:",inline"`
}

// DefaultHost returns each key's default value, the guest's for a section it shares.
func DefaultHost() Host {
	d := Default()
	return Host{
		Base: d.Base, Docker: d.Docker, Fans: defaultFans(), Cable: Cable{Address: "10.77.0.1/30"}, Signoz: d.Signoz, Incus: defaultIncus(),
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
	for _, check := range []func() error{h.Base.check, h.Docker.check, h.Cable.check, h.Incus.check} {
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
	return h, nil
}

// LoadHost reads the host's values at path on m. A file that does not exist gives DefaultHost.
func LoadHost(m effect.Machine, path string) (Host, error) {
	data, err := m.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return Host{}, err
	}
	h, err := ParseHost(data)
	if err != nil {
		return Host{}, fmt.Errorf("%s: %w", path, err)
	}
	return h, nil
}
