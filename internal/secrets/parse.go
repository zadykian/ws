package secrets

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// Parse reads the decrypted file: Decode, then Expand. Empty data gives no secrets.
func Parse(data []byte) (Secrets, error) {
	s, err := Decode(data)
	if err != nil {
		return Secrets{}, err
	}
	return Expand(s)
}

// Decode reads the decrypted file and checks it, leaving each reference as written. It refuses a
// key that ws does not know, a key twice and a value of the wrong type. It also refuses an env name
// that the shell does not take, a git key that is no https:// URL and a file that is no base64.
// Edit the result and Marshal it. Expand gives the values the roles take.
func Decode(data []byte) (Secrets, error) {
	var d document
	dec := toml.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&d); err != nil {
		return Secrets{}, scrub(err)
	}
	return fromDocument(d)
}

// scrub keeps a value out of a decode error, which TOML words with the text it could not read. It
// gives the line alone, and the key where the key is unknown.
func scrub(err error) error {
	var strict *toml.StrictMissingError
	if errors.As(err, &strict) {
		var at []string
		for _, e := range strict.Errors {
			row, _ := e.Position()
			at = append(at, fmt.Sprintf("line %d: %s is a key that ws does not know", row, strings.Join(e.Key(), ".")))
		}
		return errors.New(strings.Join(at, "; "))
	}
	var de *toml.DecodeError
	if errors.As(err, &de) {
		row, _ := de.Position()
		return fmt.Errorf("line %d: not valid TOML, or a value of the wrong type", row)
	}
	return errors.New("not valid TOML")
}

var (
	envName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	gitURL  = regexp.MustCompile(`^https://\S+$`)
)

func fromDocument(d document) (Secrets, error) {
	s := Secrets{
		Token:    Tokens{GitHub: d.Token.GitHub, AzureDevOps: d.Token.AzureDevOps},
		Env:      d.Env,
		Password: Passwords{VPNP12: d.Password.VPNP12},
	}
	var bad []string
	for name := range d.Env {
		if !envName.MatchString(name) {
			bad = append(bad, fmt.Sprintf("env %q is not a shell name", name))
		}
	}
	for url, g := range d.Git {
		if !gitURL.MatchString(url) {
			bad = append(bad, fmt.Sprintf("git %q is not an https:// URL", url))
		}
		if s.Git == nil {
			s.Git = map[string]Git{}
		}
		s.Git[url] = Git(g)
	}
	for _, f := range []struct {
		name string
		in   string
		to   *[]byte
	}{{"vpn_p12", d.File.VPNP12, &s.File.VPNP12}, {"vpn_sswan", d.File.VPNSswan, &s.File.VPNSswan}} {
		if f.in == "" {
			continue
		}
		b, err := base64.StdEncoding.DecodeString(f.in)
		if err != nil {
			bad = append(bad, "file "+f.name+" is not base64")
		}
		*f.to = b
	}
	if len(bad) > 0 {
		slices.Sort(bad)
		return Secrets{}, errors.New(strings.Join(bad, "; "))
	}
	return s, nil
}

// Marshal writes the file as TOML, the files in base64. It writes the values as they are: pass the
// result of Decode to keep the references.
func Marshal(s Secrets) ([]byte, error) {
	d := document{
		Token:    tokenTable{GitHub: s.Token.GitHub, AzureDevOps: s.Token.AzureDevOps},
		Env:      s.Env,
		File:     fileTable{VPNP12: b64(s.File.VPNP12), VPNSswan: b64(s.File.VPNSswan)},
		Password: passwordTable{VPNP12: s.Password.VPNP12},
	}
	for url, g := range s.Git {
		if d.Git == nil {
			d.Git = map[string]gitTab{}
		}
		d.Git[url] = gitTab(g)
	}
	return toml.Marshal(d)
}

func b64(b []byte) string { return base64.StdEncoding.EncodeToString(b) }
