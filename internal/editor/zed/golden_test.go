package zed_test

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/zadykian/ws/internal/editor/zed"
	"github.com/zadykian/ws/mac"
)

// golden fails where the render differs. A meant change edits the file by hand, in its commit.
func golden(t *testing.T, name string, got []byte) {
	want, err := os.ReadFile("../../server/zed/files/" + name)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("%s differs: %d bytes, want %d\n%s", name, len(got), len(want), got)
	}
}

func TestKeymapEqualsTheMacsFileByteForByte(t *testing.T) {
	got, err := zed.Keymap()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, mac.ZedKeymap) {
		t.Fatalf("keymap.json differs: %d bytes, want %d\n%s", len(got), len(mac.ZedKeymap), got)
	}
}

func TestSettingsEqualTheMacsFileByteForByte(t *testing.T) {
	got, err := zed.Settings()
	if err != nil {
		t.Fatal(err)
	}
	header := "// Written by ws mac apply: its next run replaces changes made here.\n"
	if want := append([]byte(header), got...); !bytes.Equal(want, mac.ZedSettings) {
		t.Fatalf("global_settings.json differs: %d bytes, want %d\n%s", len(got), len(mac.ZedSettings), got)
	}
}

var paths = map[string]string{
	"gopls":                       "/home/ws/go/bin/gopls",
	"roslyn-language-server":      "/home/ws/.dotnet/tools/roslyn-language-server",
	"postgres-language-server":    "/usr/local/bin/postgres-language-server",
	"ansible-language-server":     "/home/ws/.local/lib/language-servers/node_modules/.bin/ansible-language-server",
	"vscode-html-language-server": "/home/ws/.local/lib/language-servers/node_modules/.bin/vscode-html-language-server",
	"dockerfile-language-server":  "/home/ws/.local/lib/language-servers/node_modules/.bin/docker-langserver",
	"docker-language-server":      "/usr/local/bin/docker-language-server",
}

func TestServerSettingsMatchTheGoldenFileByteForByte(t *testing.T) {
	got, err := zed.ServerSettings(paths, "/home/ws/go/bin/dlv", "/usr/local/lib/ws/zed/terminal")
	if err != nil {
		t.Fatal(err)
	}
	golden(t, "settings.json.tmpl", got)
	_, body, _ := bytes.Cut(got, []byte("\n"))
	if !json.Valid(body) {
		t.Fatalf("not JSON:\n%s", body)
	}
}

func TestAServerOfTheCatalogueWithNoPathFails(t *testing.T) {
	if _, err := zed.ServerSettings(map[string]string{}, "dlv", "terminal"); err == nil {
		t.Fatal("no error")
	}
}
