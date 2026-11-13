package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLatestRedirectsToTheTagInTheLatestFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "latest"), []byte("v0.2.0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	handler(dir).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "http://127.0.0.1:9/latest", nil))
	if rec.Code != http.StatusFound || rec.Header().Get("Location") != "http://127.0.0.1:9/tag/v0.2.0" {
		t.Fatalf("got %d to %q", rec.Code, rec.Header().Get("Location"))
	}
}

func TestOtherPathsServeTheirFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "download", "v1"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "download", "v1", "ws.sha256"), []byte("abc\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	handler(dir).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/download/v1/ws.sha256", nil))
	if rec.Code != http.StatusOK || rec.Body.String() != "abc\n" {
		t.Fatalf("got %d %q", rec.Code, rec.Body.String())
	}
}

func TestMissingLatestFails(t *testing.T) {
	rec := httptest.NewRecorder()
	handler(t.TempDir()).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/latest", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("got %d", rec.Code)
	}
}

func TestCommandTakesTwoOrThreeArguments(t *testing.T) {
	for _, args := range [][]string{{}, {"dir"}, {"dir", "0", "port", "extra"}} {
		c := newCommand()
		c.SetArgs(args)
		if err := c.Execute(); err == nil || !strings.Contains(err.Error(), "accepts between 2 and 3 arg(s)") {
			t.Errorf("%q: err = %v, want the count refused", args, err)
		}
	}
}
