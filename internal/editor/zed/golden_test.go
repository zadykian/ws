package zed_test

import (
	"bytes"
	"testing"

	"github.com/zadykian/ws/internal/editor/zed"
	"github.com/zadykian/ws/mac"
)

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
