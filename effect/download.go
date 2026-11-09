package effect

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"path/filepath"

	"github.com/google/renameio/v2"
)

// Download streams the URL through SHA-256 into a pending file, and renames it over path only
// where the sum, in hex, matches. Otherwise nothing is left and path is unchanged.
func (OSMachine) Download(ctx context.Context, url, path, sum string, a Attrs) error {
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
	f, err := renameio.NewPendingFile(path,
		renameio.WithTempDir(filepath.Dir(path)), renameio.WithPermissions(a.Mode.Perm()))
	if err != nil {
		return err
	}
	defer func() { _ = f.Cleanup() }()
	h := sha256.New()
	if _, err := io.Copy(io.MultiWriter(f, h), resp.Body); err != nil {
		return fmt.Errorf("download %s: %w", url, err)
	}
	if got := hex.EncodeToString(h.Sum(nil)); got != sum {
		return fmt.Errorf("download %s: SHA-256 is %s, want %s", url, got, sum)
	}
	if err := f.Chmod(a.Mode.Perm()); err != nil {
		return err
	}
	if err := f.Chown(a.UID, a.GID); err != nil {
		return err
	}
	return f.CloseAtomicallyReplace()
}
