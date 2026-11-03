// Package secrets holds the values that ws server apply takes from an age-encrypted file: the
// secrets that the roles need and that stay out of host_vars.yml. The file is YAML, decrypted in
// ws's process, and typed beside the values. Nothing here prints a value.
package secrets

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/zadykian/ws/internal/server/values"
)

// Secrets is every key of the decrypted file. Each role's keys sit in a struct of its name,
// inlined, as in values.Values.
type Secrets struct {
	Shell   Shell   `yaml:",inline"`
	Ikeward Ikeward `yaml:",inline"`
	// Files are the ikeward profile's files, decrypted by Load. They are never in the file itself.
	Files Files `yaml:"-"`
}

// Shell is roles/shell's secrets.
type Shell struct {
	// SecretEnv is the content of ~/.secrets/env: names to values, all strings.
	SecretEnv map[string]string `yaml:"shell_secret_env,omitempty"`
	// GitCredentials are git's tokens for servers besides GitHub. They replace the list of the
	// values file.
	GitCredentials []values.GitCredential `yaml:"shell_git_credentials,omitempty"`
}

// Ikeward is roles/ikeward's secrets.
type Ikeward struct {
	// P12File and SswanFile name the profile's files, each age-encrypted to the same recipient. A
	// relative name is relative to the secrets file.
	P12File   string `yaml:"ikeward_p12_file,omitempty"`
	SswanFile string `yaml:"ikeward_sswan_file,omitempty"`
	// P12Password is the .p12's password, empty where it has none.
	P12Password string `yaml:"ikeward_p12_password,omitempty"`
}

// Files are the decrypted contents of the files that Ikeward names.
type Files struct {
	P12   []byte
	Sswan []byte
}

// Parse reads the decrypted file. It refuses a key that no role knows, and a key twice. Empty data
// gives no secrets.
func Parse(data []byte) (Secrets, error) {
	var s Secrets
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&s); err != nil && !errors.Is(err, io.EOF) {
		return Secrets{}, scrub(err)
	}
	return s, nil
}

// atLine matches a line number in YAML's type errors.
var atLine = regexp.MustCompile(`line \d+`)

// scrub keeps a value out of a type error, which YAML words with the value it could not read.
func scrub(err error) error {
	var te *yaml.TypeError
	if !errors.As(err, &te) {
		return err
	}
	var lines []string
	for _, e := range te.Errors {
		if l := atLine.FindString(e); l != "" && !strings.Contains(e, "not found in type") {
			lines = append(lines, l)
		} else {
			lines = append(lines, e)
		}
	}
	return fmt.Errorf("a value has the wrong type at %s", strings.Join(lines, ", "))
}
