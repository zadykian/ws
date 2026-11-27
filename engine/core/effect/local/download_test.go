package local_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/local"
	"github.com/zadykian/ws/engine/core/web"
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
	m := local.OSMachine{}
	ctx := context.Background()
	if err := m.Download(ctx, srv.URL+"/f", path, strings.Repeat("0", 64), a); !errors.Is(err, effect.ErrSHA256) || !strings.Contains(err.Error(), "SHA-256 is ") {
		t.Errorf("a wrong sum: err = %v, want an ErrSHA256 error", err)
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

func TestDownloadRefusesARedirectToPlainHTTP(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/f" {
			_, _ = w.Write([]byte("body"))
			return
		}
		http.Redirect(w, r, "/f", http.StatusFound)
	}))
	defer srv.Close()
	path := filepath.Join(t.TempDir(), "f")
	a := effect.Attrs{UID: os.Getuid(), GID: os.Getgid(), Mode: 0o644}
	err := local.OSMachine{}.Download(context.Background(), srv.URL+"/go", path, fmt.Sprintf("%x", sha256.Sum256([]byte("body"))), a)
	if err == nil || !strings.Contains(err.Error(), "not HTTPS") {
		t.Errorf("a redirect to plain HTTP: err = %v, want a refusal", err)
	}
	if _, err := os.Stat(path); err == nil {
		t.Error("a refused redirect left the file")
	}
}

func TestDownloadGivesUpOnAServerThatNeverAnswers(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { <-release }))
	defer srv.Close()
	defer close(release)
	// Download's client shares web.Transport, so a short timeout there stands for the shipped one.
	was := web.Transport.ResponseHeaderTimeout
	web.Transport.ResponseHeaderTimeout = 100 * time.Millisecond
	defer func() { web.Transport.ResponseHeaderTimeout = was }()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	a := effect.Attrs{UID: os.Getuid(), GID: os.Getgid(), Mode: 0o644}
	err := local.OSMachine{}.Download(ctx, srv.URL, filepath.Join(t.TempDir(), "f"), "00", a)
	if err == nil || ctx.Err() != nil {
		t.Errorf("a silent server: err = %v, ctx = %v, want a timeout before the context's", err, ctx.Err())
	}
}
