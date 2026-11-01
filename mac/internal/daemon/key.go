package daemon

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"path"

	"github.com/zadykian/ws/mac/internal/effect"
	"github.com/zadykian/ws/mac/internal/wgkey"
)

// Key makes the Mac's private key in dir where none is, and returns the public key, in base64. It
// runs as root, so the private key never leaves root's file: it is written under mode 0600 to a new
// file and moved over private.key, so that a stop in between leaves no half of a key. A key that
// is there but not a key fails, and is never replaced.
func Key(ctx context.Context, m effect.Machine, random io.Reader, dir string) (string, error) {
	file := path.Join(dir, "private.key")
	if _, err := m.ReadFile(file); errors.Is(err, fs.ErrNotExist) {
		return create(ctx, m, random, file)
	}
	private, err := ReadKey(m, dir)
	if err != nil {
		return "", err
	}
	return wgkey.Public(private)
}

// create writes a new private key to file, and returns its public key.
func create(ctx context.Context, m effect.Machine, random io.Reader, file string) (string, error) {
	private, err := wgkey.Generate(random)
	if err != nil {
		return "", err
	}
	if err := m.WriteFile(file+".new", []byte(private+"\n"), 0o600); err != nil {
		return "", err
	}
	if _, err := m.Run(ctx, "mv", "-f", file+".new", file); err != nil {
		return "", err
	}
	return wgkey.Public(private)
}
