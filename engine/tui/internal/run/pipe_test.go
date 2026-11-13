package run

import (
	"io"
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/tui/internal/widget"
)

func TestPipeSendsWholeLines(t *testing.T) {
	b, sent := testBridge(io.Discard)
	_, _ = b.pipe.Write([]byte("one\ntwo\nthr"))
	_, _ = b.pipe.Write([]byte("ee\n"))
	want := []tea.Msg{widget.Line{Kind: widget.Piped, Text: "one\ntwo"}, widget.Line{Kind: widget.Piped, Text: "three"}}
	if !reflect.DeepEqual(*sent, want) {
		t.Errorf("sent %v, want %v", *sent, want)
	}
}

func TestPipeFlushSendsALineWithNoEnd(t *testing.T) {
	b, sent := testBridge(io.Discard)
	_, _ = b.pipe.Write([]byte("no end"))
	b.pipe.flush()
	b.pipe.flush()
	if want := []tea.Msg{widget.Line{Kind: widget.Piped, Text: "no end"}}; !reflect.DeepEqual(*sent, want) {
		t.Errorf("sent %v, want %v", *sent, want)
	}
}

func TestPipeBoundsALineWithNoEnd(t *testing.T) {
	b, sent := testBridge(io.Discard)
	n, err := b.pipe.Write([]byte(strings.Repeat("x", pipeCap+1)))
	if n != pipeCap+1 || err != nil {
		t.Errorf("Write = %d, %v", n, err)
	}
	if len(*sent) != 1 || len(b.pipe.rest) != 0 {
		t.Errorf("sent %d messages, and %d bytes wait", len(*sent), len(b.pipe.rest))
	}
}

func TestMachineSendsWhatCommandsPrintToTheScreen(t *testing.T) {
	b := newBridge(io.Discard)
	m := b.Machine(t.Context(), effect.OSMachine{Sudo: []string{}})
	if m.Stderr != b.pipe || m.Stdin != effect.NoStdin {
		t.Errorf("Stderr %v, Stdin %v", m.Stderr, m.Stdin)
	}
}
