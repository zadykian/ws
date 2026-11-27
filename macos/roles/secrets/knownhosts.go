package secrets

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os/exec"
	"path"
	"strconv"
	"strings"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/macos/values"
)

// KnownHosts is ssh's known_hosts of the user whose home is home. The Mac's ssh config checks a
// server's host key there under the server's name, as its HostKeyAlias.
func KnownHosts(home string) string {
	return path.Join(home, ".ssh", "known_hosts")
}

// HostKey returns name's ed25519 host key in home's known_hosts, as "ssh-ed25519 BASE64". It asks
// ssh-keygen -F, which matches a hashed entry too. A missing file or entry gives "". A @revoked or
// @cert-authority line is no host key, and two keys stop it, as ssh would trust either.
func HostKey(ctx context.Context, m effect.Machine, home, name string) (string, error) {
	if !values.IsName(name) {
		return "", fmt.Errorf("%s is not a server's name: %s", name, values.NameForm)
	}
	file := KnownHosts(home)
	if _, err := m.Stat(file); errors.Is(err, fs.ErrNotExist) {
		return "", nil
	} else if err != nil {
		return "", fmt.Errorf("cannot stat %s: %w", file, err)
	}
	out, err := m.Run(ctx, "ssh-keygen", "-F", name, "-f", file)
	var ee *exec.ExitError
	if errors.As(err, &ee) && ee.ExitCode() == 1 {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("ssh-keygen cannot read %s: %w", file, err)
	}
	key, err := ed25519Key(out)
	if err != nil {
		return "", fmt.Errorf("%s holds %s: %w", file, name, err)
	}
	return key, nil
}

// knownKeys are the ed25519 keys that home's known_hosts gives any host, as "ssh-ed25519 BASE64". A
// missing file gives none. A line that is not of ssh's form gives none, as ssh trusts no key there.
func knownKeys(m effect.Machine, home string) (map[string]bool, error) {
	file := KnownHosts(home)
	text, err := m.ReadFile(file)
	if errors.Is(err, fs.ErrNotExist) {
		return map[string]bool{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", file, err)
	}
	keys := map[string]bool{}
	for line := range strings.Lines(string(text)) {
		if k := lineKey(line); hostKeyForm.MatchString(k) {
			keys[k] = true
		}
	}
	return keys, nil
}

// lineKey is the ed25519 key of a known_hosts line, HOST TYPE KEY, or "". A comment and a line that
// a marker such as @revoked starts give "".
func lineKey(line string) string {
	f := strings.Fields(line)
	if len(f) < 3 || strings.HasPrefix(f[0], "#") || strings.HasPrefix(f[0], "@") || f[1] != "ssh-ed25519" {
		return ""
	}
	return f[1] + " " + f[2]
}

// ed25519Key takes the one ed25519 key of a known_hosts text, as lineKey reads each line.
func ed25519Key(text []byte) (string, error) {
	var key string
	for line := range strings.Lines(string(text)) {
		k := lineKey(line)
		if k == "" {
			continue
		}
		if !hostKeyForm.MatchString(k) {
			return "", errors.New("an ed25519 key that is not of ssh's form")
		}
		if key != "" && key != k {
			return "", errors.New("two ed25519 keys: keep the server's one")
		}
		key = k
	}
	return key, nil
}

// PubKey takes the ed25519 key of a host key's .pub text, as "ssh-ed25519 BASE64", or "" where the
// text holds none.
func PubKey(text []byte) (string, error) {
	var b strings.Builder
	for line := range strings.Lines(string(text)) {
		b.WriteString("host " + line)
	}
	return ed25519Key([]byte(b.String()))
}

// RecordHostKey appends name's line with key to home's known_hosts, made 0600 where missing, in an
// .ssh made 0700. The file's lines stay as they are, and it ends 0600. A link is left to the
// maintainer, who chose where it points.
func RecordHostKey(m effect.Machine, home, name, key string) error {
	if !values.IsName(name) {
		return fmt.Errorf("%s is not a server's name: %s", name, values.NameForm)
	}
	if !hostKeyForm.MatchString(key) {
		return fmt.Errorf("the host key of %s is not ssh-ed25519 BASE64", name)
	}
	file := KnownHosts(home)
	info, err := m.Stat(file)
	var text []byte
	switch {
	case errors.Is(err, fs.ErrNotExist):
		if err := makeDirs(m, path.Dir(file)); err != nil {
			return err
		}
	case err != nil:
		return fmt.Errorf("cannot stat %s: %w", file, err)
	case info.Link != "":
		return fmt.Errorf("%s is a link to %s: append the line %s %s there by hand", file, info.Link, name, key)
	default:
		if text, err = m.ReadFile(file); err != nil {
			return fmt.Errorf("cannot read %s: %w", file, err)
		}
	}
	if len(text) > 0 && text[len(text)-1] != '\n' {
		text = append(text, '\n')
	}
	text = append(text, name+" "+key+"\n"...)
	if err := m.WriteFile(file, text, 0o600); err != nil {
		return fmt.Errorf("cannot write %s: %w", file, err)
	}
	return nil
}

// ReplaceHostKey puts name's line with key in place of the lines that home's known_hosts holds of
// name. ssh-keygen -R drops them, a hashed one too, before the line is appended. A link is left
// to the maintainer, as ssh-keygen would put a file in its place.
func ReplaceHostKey(ctx context.Context, m effect.Machine, home, name, key string) error {
	if !values.IsName(name) {
		return fmt.Errorf("%s is not a server's name: %s", name, values.NameForm)
	}
	if !hostKeyForm.MatchString(key) {
		return fmt.Errorf("the host key of %s is not ssh-ed25519 BASE64", name)
	}
	file := KnownHosts(home)
	info, err := m.Stat(file)
	if err != nil {
		return fmt.Errorf("cannot stat %s: %w", file, err)
	}
	if info.Link != "" {
		return fmt.Errorf("%s is a link to %s: put the line %s %s there in place of %s's by hand", file, info.Link,
			name, key, name)
	}
	if _, err := m.Run(ctx, "ssh-keygen", "-R", name, "-f", file); err != nil {
		return fmt.Errorf("ssh-keygen cannot drop %s from %s: %w", name, file, err)
	}
	return RecordHostKey(m, home, name, key)
}

// ScanHostKey reads the ed25519 host key that the server at host and port offers, as
// "ssh-ed25519 BASE64". Nothing proves it the server's yet: the maintainer compares its
// fingerprint at the server's console.
func ScanHostKey(ctx context.Context, m effect.Machine, host string, port int) (string, error) {
	if host == "" || strings.HasPrefix(host, "-") || strings.ContainsAny(host, " \t\n") {
		return "", fmt.Errorf("%q is not a host to scan", host)
	}
	if port < 1 || port > 65535 {
		return "", fmt.Errorf("%d is not a port", port)
	}
	at := host + ":" + strconv.Itoa(port)
	out, err := m.Run(ctx, "ssh-keyscan", "-t", "ed25519", "-p", strconv.Itoa(port), host)
	if err != nil {
		return "", fmt.Errorf("ssh-keyscan cannot read the host key at %s: %w", at, err)
	}
	key, err := ed25519Key(out)
	switch {
	case err != nil:
		return "", fmt.Errorf("the scan of %s gives %w", at, err)
	case key == "":
		return "", fmt.Errorf("the server at %s offers no ed25519 host key", at)
	}
	return key, nil
}

// Fingerprint is ssh-keygen -l's line of key, such as "256 SHA256:... no comment (ED25519)", which
// the server's console prints for its own key. The key goes in on stdin, as no file holds it.
func Fingerprint(ctx context.Context, m effect.Machine, key string) (string, error) {
	out, err := m.RunInput(ctx, []byte(key+"\n"), "ssh-keygen", "-l", "-f", "-")
	if err != nil {
		return "", fmt.Errorf("ssh-keygen cannot read the host key: %w", err)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) != 1 || lines[0] == "" {
		return "", errors.New("ssh-keygen -l gives no single fingerprint for the host key")
	}
	return lines[0], nil
}
