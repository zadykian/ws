package tui

import (
	"bytes"
	"io"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/zadykian/ws/effect"
)

// testBridge returns a bridge that keeps what it sends, and prints on w.
func testBridge(w io.Writer) (*Bridge, *[]tea.Msg) {
	sent := &[]tea.Msg{}
	b := newBridge(w)
	b.send = func(msg tea.Msg) { *sent = append(*sent, msg) }
	return b, sent
}

func TestBridgeByHandPrintsAsAPlainRunDoes(t *testing.T) {
	var got, want bytes.Buffer
	plain := effect.NewTerminal(strings.NewReader(""), &want)
	plain.ByHand("restart %s", "ws")
	plain.Flush()
	b, sent := testBridge(&got)
	b.ByHand("restart %s", "ws")
	if len(*sent) != 0 || got.Len() != 0 {
		t.Fatalf("sent %v and printed %q before the end", *sent, got.String())
	}
	b.flush()
	if got.String() != want.String() {
		t.Errorf("printed %q, want %q", got.String(), want.String())
	}
}
