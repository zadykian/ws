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
	Shared    `yaml:",inline"`
	Base      Base      `yaml:",inline"`
	Claude    Claude    `yaml:",inline"`
	Devtools  Devtools  `yaml:",inline"`
	Docker    Docker    `yaml:",inline"`
	Fans      Fans      `yaml:",inline"`
	Ikeward   Ikeward   `yaml:",inline"`
	Network   Network   `yaml:",inline"`
	Security  Security  `yaml:",inline"`
	Shell     Shell     `yaml:",inline"`
	Signoz    Signoz    `yaml:",inline"`
	Wireguard Wireguard `yaml:",inline"`
	Zed       Zed       `yaml:",inline"`
}

// Shared holds the keys that the roles share.
type Shared struct {
	// User is the user the roles set the machine up for.
	User string `yaml:"ws_user"`
	// Home is the user's home. Left empty, it follows User: /root for root, else /home/USER.
	Home string `yaml:"ws_home"`
	// GitSigningKey is the SSH key that signs git's commits, with its public key beside it as .pub.
	// Left empty, it defaults to Home/.ssh/id_ed25519_signing.
	GitSigningKey string `yaml:"ws_git_signing_key"`
}

// Parse returns the values that data gives over Default. It refuses a key that no role knows, a key
// twice, and a path that underHome refuses. Empty data gives Default.
func Parse(data []byte) (Values, error) {
	v, err := decode(data)
	if err != nil {
		return Values{}, err
	}
	if err := v.underHome(); err != nil {
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

// derive fills the values that follow ws_home when the file leaves them empty. It lists the Mac
// at wg0's address plus one where wireguard_peers is unset: an empty list keeps no peer.
func (v *Values) derive() {
	if v.Home == "" {
		v.Home = "/root"
		if v.User != "root" {
			v.Home = "/home/" + v.User
		}
	}
	if v.GitSigningKey == "" {
		v.GitSigningKey = v.Home + "/.ssh/id_ed25519_signing"
	}
	if v.Shell.GitAllowedSigners == "" {
		v.Shell.GitAllowedSigners = v.Home + "/.config/git/allowed_signers"
	}
	if v.Wireguard.Peers == nil {
		if p, err := netip.ParsePrefix(v.Wireguard.Address); err == nil && p.Addr().Is4() {
			v.Wireguard.Peers = []WireguardPeer{{Name: "mac", Address: p.Addr().Next().String()}}
		}
	}
}

// underHome refuses a signing key or an allowed signers file that is not below ws_home. A role
// reaches such a file through the account, by a path under the home, so that it can refuse a link
// there (roles/server/ubuntu/security/README.md).
func (v *Values) underHome() error {
	home := path.Clean(v.Home)
	for _, k := range []struct{ key, val string }{
		{"ws_git_signing_key", v.GitSigningKey},
		{"shell_git_allowed_signers", v.Shell.GitAllowedSigners},
	} {
		if !strings.HasPrefix(path.Clean(k.val), strings.TrimSuffix(home, "/")+"/") {
			return fmt.Errorf("%s: %s is outside ws_home %s", k.key, k.val, v.Home)
		}
	}
	return nil
}
