package local

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/google/renameio/v2"

	"github.com/zadykian/ws/engine/core/effect"
)

// Download streams the URL through SHA-256 into a pending file, and renames it over path only
// where the sum, in hex, matches. Otherwise nothing is left and path is unchanged. Where ws runs as
// a user and path is another's, it keeps the body in a file of ws's own. Then sudo moves that file
// to path, once the sum matches.
func (m OSMachine) Download(ctx context.Context, url, path, sum string, a effect.Attrs) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: %s", url, resp.Status)
	}
	if a.UID >= 0 && m.asRoot(path) && another(path, a.UID) {
		return m.sudoDownload(url, resp.Body, path, sum, a)
	}
	f, err := renameio.NewPendingFile(path,
		renameio.WithTempDir(filepath.Dir(path)), renameio.WithPermissions(a.Mode.Perm()))
	if err != nil {
		return err
	}
	defer func() { _ = f.Cleanup() }()
	if err := copyChecked(f, resp.Body, url, sum); err != nil {
		return err
	}
	if err := f.Chmod(a.Mode.Perm()); err != nil {
		return err
	}
	if err := f.Chown(a.UID, a.GID); err != nil {
		return err
	}
	return f.CloseAtomicallyReplace()
}

// copyChecked copies the body to dst, and fails where its SHA-256, in hex, is not sum.
func copyChecked(dst io.Writer, body io.Reader, url, sum string) error {
	h := sha256.New()
	if _, err := io.Copy(io.MultiWriter(dst, h), body); err != nil {
		return fmt.Errorf("download %s: %w", url, err)
	}
	if got := hex.EncodeToString(h.Sum(nil)); got != sum {
		return fmt.Errorf("download %s: %w is %s, want %s", url, effect.ErrSHA256, got, sum)
	}
	return nil
}

// sudoDownload checks the body in a file of ws's own, which the user alone reads. Then sudo writes
// that file to path, with the owner and mode of a.
func (m OSMachine) sudoDownload(url string, body io.Reader, path, sum string, a effect.Attrs) error {
	f, err := os.CreateTemp("", ".ws-download-*")
	if err != nil {
		return err
	}
	defer func() { _ = f.Close(); _ = os.Remove(f.Name()) }()
	if err := copyChecked(f, body, url, sum); err != nil {
		return err
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return err
	}
	return m.sudoWriteFrom(path, f, fmt.Sprintf("%d:%d", a.UID, a.GID), a.Mode)
}
