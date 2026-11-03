// Package values holds the machine's values, ws's host_vars.yml, as one typed struct. The file
// is parsed once, and a key that no role knows is refused.
package values

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"

	"go.yaml.in/yaml/v3"

	"github.com/zadykian/ws/effect"
)

// Values is every key of the file. Each role's keys sit in a struct of its name, inlined, so that
// a role's steps read v.Role.Key. A key left out of the file keeps the default that Default gives.
type Values struct {
	Shared    `yaml:",inline"`
	Base      Base      `yaml:",inline"`
	Claude    Claude    `yaml:",inline"`
	Devtools  Devtools  `yaml:",inline"`
	Docker    Docker    `yaml:",inline"`
	Fans      Fans      `yaml:",inline"`
	Helix     Helix     `yaml:",inline"`
	Ikeward   Ikeward   `yaml:",inline"`
	JetBrains JetBrains `yaml:",inline"`
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
	// GitSigningKey is the SSH key that signs git's commits, with its public key beside it as
	// .pub. Left empty, it is Home/.ssh/id_ed25519_signing.
	GitSigningKey string `yaml:"ws_git_signing_key"`
}

// Parse returns the values that data gives over Default. It refuses a key that no role knows, and
// a key twice. Empty data gives Default.
func Parse(data []byte) (Values, error) {
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

// derive fills the values that follow ws_home when the file leaves them empty.
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
}
