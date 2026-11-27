package secrets

import (
	"context"
	"fmt"
	"path"
	"regexp"
	"slices"

	"github.com/pelletier/go-toml/v2"

	"github.com/zadykian/ws/engine/core/effect"
)

// ProfileProvider is the age provider of a server's own file. Its name differs from
// SharedProvider's, as fnox merges providers by name, and a profile's would replace the shared one.
const ProfileProvider = "host"

// ProfileFile is the file of the server hostname's own values, which fnox reads under the profile
// of that name.
func ProfileFile(hostname string) string {
	return "fnox." + hostname + ".toml"
}

// Files are the clone's fnox files by name: SharedFile and each profile's.
type Files map[string][]byte

// profileFileName matches a profile's file, and names its profile.
var profileFileName = regexp.MustCompile(`^fnox\.(.+)\.toml$`)

// isProfileFile says whether name is a profile's file. fnox.local.toml is fnox's local override of
// a checkout, which no server reads.
func isProfileFile(name string) bool {
	m := profileFileName.FindStringSubmatch(name)
	return m != nil && m[1] != "local"
}

// ReadFiles reads the fnox files at the top of the clone's directory dir.
func ReadFiles(m effect.Machine, dir string) (Files, error) {
	names, err := m.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("cannot list the clone %s: %w", dir, err)
	}
	f := Files{}
	for _, name := range names {
		if name != SharedFile && !isProfileFile(name) {
			continue
		}
		data, err := m.ReadFile(path.Join(dir, name))
		if err != nil {
			return nil, fmt.Errorf("cannot read %s: %w", name, err)
		}
		f[name] = data
	}
	return f, nil
}

// WriteFiles writes the files of f that names lists into the clone's directory dir.
func WriteFiles(m effect.Machine, dir string, f Files, names []string) error {
	for _, name := range names {
		data, ok := f[name]
		if !ok {
			return fmt.Errorf("no file %s to write", name)
		}
		if err := m.WriteFile(path.Join(dir, name), data, 0o644); err != nil {
			return fmt.Errorf("cannot write %s: %w", name, err)
		}
	}
	return nil
}

// hostnameForm is one DNS label, which serves unchanged as a profile, a file's name, a label and a
// deploy key's title. #1327's task on the server checks the same.
var hostnameForm = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]{0,61}[a-z0-9])?$`)

// CheckHostname refuses a hostname that names no profile: one that is not a DNS label, or that a
// recipient's label or fnox itself takes. fnox.local.toml is fnox's local override, and default
// its profile without -P.
func CheckHostname(h string) error {
	if !hostnameForm.MatchString(h) || slices.Contains([]string{NanoLabel, BackupLabel, "local", "default"}, h) {
		return fmt.Errorf("the hostname %q names no profile: set a hostname of one label at ws, "+
			"lowercase letters, digits and -, other than %s, %s, local and default", h, NanoLabel, BackupLabel)
	}
	return nil
}

// Reencrypt encrypts the values of each changed file of f again, to the recipients its table holds
// now. fnox.toml takes the shared provider, and a profile's file its own. A file of no value has
// nothing to encrypt, so fnox runs on none. fnox asks for the Nano's PIN and touch on the terminal.
func Reencrypt(ctx context.Context, c *Clone, f Files, changed []string) error {
	for _, name := range changed {
		held, err := HasSecrets(f[name])
		if err != nil {
			return fmt.Errorf("cannot read %s: %w", name, err)
		}
		if !held {
			continue
		}
		args := []string{"reencrypt", "-p", SharedProvider, "-f"}
		if m := profileFileName.FindStringSubmatch(name); m != nil {
			args = []string{"-P", m[1], "reencrypt", "-p", ProfileProvider, "-f"}
		}
		if err := c.FnoxPass(ctx, args...); err != nil {
			return fmt.Errorf("cannot encrypt the values of %s again: %w", name, err)
		}
	}
	return nil
}

// HasSecrets says whether a fnox file holds a secret, as fnox's reencrypt has nothing to do in one
// that holds none.
func HasSecrets(text []byte) (bool, error) {
	var conf struct {
		Secrets map[string]any `toml:"secrets"`
	}
	if err := toml.Unmarshal(text, &conf); err != nil {
		return false, err
	}
	return len(conf.Secrets) > 0, nil
}
