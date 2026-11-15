// Package values holds the machine's values, ws's host_vars.yml, as one typed struct. The file
// is parsed once, and a key that no role knows is refused.
package values

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/netip"
	"path"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/zadykian/ws/engine/core/effect"
)

// Values is every key of the file. A key left out of the file keeps the default that Default gives.
// Each role's keys sit in a struct of its name, inlined, so that a role's steps read v.Role.Key.
type Values struct {
	Shared   `yaml:",inline"`
	Base     Base     `yaml:",inline"`
	Claude   Claude   `yaml:",inline"`
	Devtools Devtools `yaml:",inline"`
	Docker   Docker   `yaml:",inline"`
	Fans     Fans     `yaml:",inline"`
	Ikeward  Ikeward  `yaml:",inline"`
	Cable    Cable    `yaml:",inline"`
	Security Security `yaml:",inline"`
	Git      Git      `yaml:",inline"`
	Signoz   Signoz   `yaml:",inline"`
	Tunnel   Tunnel   `yaml:",inline"`
	Zed      Zed      `yaml:",inline"`
}

// Shared holds the keys that the roles share.
type Shared struct {
	// User is the user the roles set the machine up for.
	User string `yaml:"WS_USER"`
	// Home is the user's home. Left empty, it follows User: /root for root, else /home/USER.
	Home string `yaml:"WS_HOME"`
	// GitSigningPubKey is the public key's path of the SSH key that signs git's commits. The private
	// key lies beside it, without .pub. Left empty, it defaults to Home/.ssh/id_ed25519_signing.pub.
	GitSigningPubKey string `yaml:"WS_GIT_SIGNING_PUB_KEY"`
}

// Parse returns the values that data gives over Default. It refuses a key that no role knows, a key
// twice, a path that underHome refuses, and a signing key that is no .pub's path. Empty data gives Default.
func Parse(data []byte) (Values, error) {
	v, err := decode(data)
	if err != nil {
		return Values{}, err
	}
	if err := v.underHome(); err != nil {
		return Values{}, err
	}
	if err := v.signingPub(); err != nil {
		return Values{}, err
	}
	return v, nil
}

// decode is Parse without underHome, for a part of a file, whose path lies under a home that only
// the whole file gives.
func decode(data []byte) (Values, error) {
	v := Default()
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&v); err != nil && !errors.Is(err, io.EOF) {
		return Values{}, err
	}
	// A key set to null decodes to a nil list, like a key left out: the file's own text tells them apart.
	var keys map[string]yaml.Node
	if err := yaml.Unmarshal(data, &keys); err != nil {
		return Values{}, err
	}
	if _, given := keys["WS_TUNNEL_PEERS"]; given && v.Tunnel.Peers == nil {
		v.Tunnel.Peers = []TunnelPeer{}
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

// derive fills the values that follow WS_HOME when the file leaves them empty. It lists the Mac
// at wg0's address plus one where the file has no WS_TUNNEL_PEERS: an empty or null list keeps no peer.
func (v *Values) derive() {
	if v.Home == "" {
		v.Home = "/root"
		if v.User != "root" {
			v.Home = "/home/" + v.User
		}
	}
	if v.GitSigningPubKey == "" {
		v.GitSigningPubKey = v.Home + "/.ssh/id_ed25519_signing.pub"
	}
	if v.Tunnel.Peers == nil {
		if p, err := netip.ParsePrefix(v.Tunnel.Address); err == nil && p.Addr().Is4() {
			v.Tunnel.Peers = []TunnelPeer{{Name: "mac", Address: p.Addr().Next().String()}}
		}
	}
}

// underHome refuses a signing key that is not below WS_HOME. A role reaches it through the account,
// by a path under the home, so that it can refuse a link there (roles/server/ubuntu/security/README.md).
func (v *Values) underHome() error {
	home := path.Clean(v.Home)
	if !strings.HasPrefix(path.Clean(v.GitSigningPubKey), strings.TrimSuffix(home, "/")+"/") {
		return fmt.Errorf("WS_GIT_SIGNING_PUB_KEY: %s is outside WS_HOME %s", v.GitSigningPubKey, v.Home)
	}
	return nil
}

// signingPub refuses a signing key that is no .pub's path: ws finds the private key by cutting the
// suffix, as the Mac's check does.
func (v *Values) signingPub() error {
	if !strings.HasSuffix(v.GitSigningPubKey, ".pub") {
		return fmt.Errorf("WS_GIT_SIGNING_PUB_KEY: %s is not a .pub's path", v.GitSigningPubKey)
	}
	return nil
}
