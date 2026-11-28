// Package values holds the machine's values, ws's values.yml, as one typed struct. The file
// is parsed once, and a key that no role knows is refused.
package values

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/netip"

	"go.yaml.in/yaml/v3"

	"github.com/zadykian/ws/engine/core/effect"
)

// Values is every key of the file. A key left out of the file keeps the default that Default gives.
// Each role's keys sit in a struct of its name, inlined, so that a role's steps read v.Role.Key.
type Values struct {
	Shared  `yaml:",inline"`
	Base    Base    `yaml:",inline"`
	Claude  Claude  `yaml:",inline"`
	Docker  Docker  `yaml:",inline"`
	Ikeward Ikeward `yaml:",inline"`
	Git     Git     `yaml:",inline"`
	Nvidia  Nvidia  `yaml:",inline"`
	Signoz  Signoz  `yaml:",inline"`
	Tunnel  Tunnel  `yaml:",inline"`
	Zed     Zed     `yaml:",inline"`
}

// Shared holds the keys that the roles share.
type Shared struct {
	// SelfContained is a guest's mode, off by default: the guest runs SigNoz and its own tunnel,
	// where a guest under a host takes both from the host. A cloud VM has no host, so it runs so.
	// The host's own file takes no such key.
	SelfContained bool `yaml:"WS_SELF_CONTAINED"`
}

// Parse returns the values that data gives over Default. It refuses a key that no role knows, a key
// twice, and a build cache value that dockerd cannot read. It refuses a swap size or a swappiness that the
// kernel cannot take, and a tunnel or an ikeward that its role cannot set up. Empty data
// gives Default.
func Parse(data []byte) (Values, error) {
	v, err := decode(data)
	if err != nil {
		return Values{}, err
	}
	if err := v.Base.check(); err != nil {
		return Values{}, err
	}
	if err := v.Docker.check(); err != nil {
		return Values{}, err
	}
	if err := v.Tunnel.check(); err != nil {
		return Values{}, err
	}
	if err := v.Ikeward.check(); err != nil {
		return Values{}, err
	}
	return v, nil
}

// decode is Parse without the checks, for a part of a file, whose keys may fail them with the
// defaults of the rest.
func decode(data []byte) (Values, error) {
	v := Default()
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&v); err != nil && !errors.Is(err, io.EOF) {
		return Values{}, err
	}
	if err := v.Tunnel.keepNull(data); err != nil {
		return Values{}, err
	}
	v.derive()
	return v, nil
}

// Load reads the values at path on m. A file that does not exist gives Default.
func Load(m effect.Machine, path string) (Values, error) {
	data, err := m.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return Values{}, err
	}
	v, err := Parse(data)
	if err != nil {
		return Values{}, fmt.Errorf("%s: %w", path, err)
	}
	return v, nil
}

// derive fills the tunnel's peers.
func (v *Values) derive() {
	v.Tunnel.derive()
}

// keepNull makes the peers an empty list where the file sets WS_TUNNEL_PEERS to null. That decodes
// to a nil list, like a key left out, so the file's own text tells them apart.
func (t *Tunnel) keepNull(data []byte) error {
	var keys map[string]yaml.Node
	if err := yaml.Unmarshal(data, &keys); err != nil {
		return err
	}
	if _, given := keys["WS_TUNNEL_PEERS"]; given && t.Peers == nil {
		t.Peers = []TunnelPeer{}
	}
	return nil
}

// derive lists the Mac at wg0's address plus one where the file has no WS_TUNNEL_PEERS: an empty or
// null list keeps no peer.
func (t *Tunnel) derive() {
	if t.Peers == nil {
		if p, err := netip.ParsePrefix(t.Address); err == nil && p.Addr().Is4() {
			t.Peers = []TunnelPeer{{Name: macPeer, Address: p.Addr().Next().String()}}
		}
	}
}
