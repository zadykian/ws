package effect_test

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect"
)

func TestDownloadRenamesOnAMatchAlone(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/f" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte("body"))
	}))
	defer srv.Close()
	path := filepath.Join(t.TempDir(), "f")
	a := effect.Attrs{UID: os.Getuid(), GID: os.Getgid(), Mode: 0o644}
	m := effect.OSMachine{}
	ctx := context.Background()
	if err := m.Download(ctx, srv.URL+"/f", path, strings.Repeat("0", 64), a); err == nil || !strings.Contains(err.Error(), "SHA-256") {
		t.Errorf("a wrong sum: err = %v, want a SHA-256 error", err)
	}
	if _, err := os.Stat(path); err == nil {
		t.Error("a wrong sum left the file")
	}
	if err := m.Download(ctx, srv.URL+"/no", path, "", a); err == nil || !strings.Contains(err.Error(), "404") {
		t.Errorf("a missing page: err = %v, want 404", err)
	}
	if err := m.Download(ctx, srv.URL+"/f", path, fmt.Sprintf("%x", sha256.Sum256([]byte("body"))), a); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(path); string(got) != "body" { //nolint:gosec // a temporary path of the test
		t.Errorf("file = %q, want body", got)
	}
}
