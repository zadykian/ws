package zed_test

import (
	"bytes"
	"testing"

	"github.com/zadykian/ws/internal/editor/zed"
	"github.com/zadykian/ws/mac"
)

// TestKeymapMatchesTheGoldenFileByteForByte holds the render to mac/zed-keymap.json. A change to
// the data or the adapter that is meant changes the file by hand, in the same commit.
func TestKeymapMatchesTheGoldenFileByteForByte(t *testing.T) {
	got, err := zed.Keymap()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, mac.ZedKeymap) {
		t.Fatalf("keymap differs: %d bytes, want %d\n%s", len(got), len(mac.ZedKeymap), got)
	}
}
