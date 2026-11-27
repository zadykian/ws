package secrets

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
)

// keySources are where fnox finds its age key file, in its order: each variable that is set, with
// the path below it. ws keeps the stub there, so that fnox by hand finds it too.
var keySources = []struct{ name, below string }{
	{"FNOX_AGE_KEY_FILE", ""},
	{"FNOX_CONFIG_DIR", "age.txt"},
	{"XDG_CONFIG_HOME", "fnox/age.txt"},
	{"HOME", ".config/fnox/age.txt"},
}

// KeyFile is the age key file that fnox reads where its config names none, on macOS too.
func KeyFile(getenv func(string) string) (string, error) {
	for _, s := range keySources {
		dir := getenv(s.name)
		if dir == "" {
			continue
		}
		if !filepath.IsAbs(dir) {
			return "", fmt.Errorf("%s is %s, not an absolute path: fnox's key file has no place", s.name, dir)
		}
		return filepath.Join(dir, s.below), nil
	}
	return "", errors.New("HOME is not set: fnox's key file has no place")
}

// KeepStub appends the Nano's identity file, stub, to fnox's key file, where its stub line is not
// there yet. It keeps the file's other lines, and makes its missing directories 0700 and the file
// 0600, the user's. It reports whether it wrote. The file may hold the user's own age key. So the
// new text goes in a pending file renamed over the old one, and a failed write leaves the old file.
func KeepStub(m effect.Machine, file, stub string) (bool, error) {
	line := stubOf(stub)
	if line == "" {
		return false, fmt.Errorf("no %s line to keep in %s", stubPrefix, file)
	}
	data, err := m.ReadFile(file)
	switch {
	case errors.Is(err, fs.ErrNotExist):
		if err := makeDirs(m, filepath.Dir(file)); err != nil {
			return false, err
		}
	case err != nil:
		return false, fmt.Errorf("cannot read fnox's key file %s: %w", file, err)
	}
	text := string(data)
	// fnox reads a key file that parses as an SSH key as that key alone. A stub beside it would
	// never open a value.
	if strings.Contains(text, "-----BEGIN") {
		return false, fmt.Errorf("fnox's key file %s holds an SSH key, which fnox reads alone: set "+
			"FNOX_AGE_KEY_FILE to another file", file)
	}
	for l := range strings.Lines(text) {
		if strings.TrimSpace(l) == line {
			return false, nil
		}
	}
	if text != "" && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	if !strings.HasSuffix(stub, "\n") {
		stub += "\n"
	}
	// An owner of -1 is ws's own user, the Mac's user, so no sudo writes it.
	if err := m.WriteAtomic(file, []byte(text+stub), effect.Attrs{UID: -1, GID: -1, Mode: 0o600}); err != nil {
		return false, fmt.Errorf("cannot write fnox's key file %s: %w", file, err)
	}
	return true, nil
}

// makeDirs makes dir and each parent of it that is missing, 0700, from the top down.
func makeDirs(m effect.Machine, dir string) error {
	var missing []string
	for d := dir; ; d = filepath.Dir(d) {
		_, err := m.Stat(d)
		if err == nil || d == filepath.Dir(d) {
			break
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("cannot stat %s: %w", d, err)
		}
		missing = append(missing, d)
	}
	for i := len(missing) - 1; i >= 0; i-- {
		if err := m.Mkdir(missing[i], 0o700); err != nil {
			return fmt.Errorf("cannot make %s: %w", missing[i], err)
		}
	}
	return nil
}

// fnoxEnv is the environment that ws sets for fnox: /usr/local/bin first on PATH, where fnox
// finds age-plugin-yubikey, and the key file. env sets PATH as given, so ws expands it here.
func fnoxEnv(getenv func(string) string, keyFile string) []string {
	path := "/usr/local/bin"
	if p := getenv("PATH"); p != "" {
		path += ":" + p
	}
	return []string{"PATH=" + path, "FNOX_AGE_KEY_FILE=" + keyFile}
}
