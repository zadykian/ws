package tui

import (
	"bytes"
	"io"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/zadykian/ws/effect"
)

var (
	_ effect.Prompt = (*Bridge)(nil)
	_ effect.Hidden = (*Bridge)(nil)
	_ effect.Out    = (*Bridge)(nil)
)

// testBridge returns a bridge that keeps what it sends, and prints on w.
func testBridge(w io.Writer) (*Bridge, *[]tea.Msg) {
	sent := &[]tea.Msg{}
	b := newBridge(w)
	b.send = func(msg tea.Msg) { *sent = append(*sent, msg) }
	return b, sent
}

func TestBridgeLinesHoldWhatAPlainRunPrints(t *testing.T) {
	before, after := []byte("a\n"), []byte("b\n")
	cases := []struct {
		name string
		kind LineKind
		call func(effect.Out)
	}{
		{"say", Said, func(o effect.Out) { o.Say("%d changes", 2) }},
		{"raw", Copied, func(o effect.Out) { o.Raw("ssh %s", "ws") }},
		{"warn", Warned, func(o effect.Out) { o.Warn("no %s", "cable") }},
		{"diff", Diffed, func(o effect.Out) { o.Diff("f", before, after) }},
		{"equal diff", Diffed, func(o effect.Out) { o.Diff("f", before, before) }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var plain bytes.Buffer
			c.call(effect.NewTerminal(strings.NewReader(""), &plain))
			b, sent := testBridge(io.Discard)
			c.call(b)
			var want []tea.Msg
			if plain.Len() > 0 {
				want = []tea.Msg{Line{c.kind, strings.TrimSuffix(plain.String(), "\n")}}
			}
			if !slices.Equal(*sent, want) {
				t.Errorf("sent %v, want %v", *sent, want)
			}
		})
	}
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

func TestBridgeAskSendsTheQuestionAndReturnsTheReply(t *testing.T) {
	b, _ := testBridge(io.Discard)
	var asked prompt
	b.send = func(msg tea.Msg) {
		asked = msg.(askMsg).prompt
		asked.reply <- " kept as sent "
	}
	got, err := b.Ask(t.Context(), effect.Question{Text: "Host", Default: "ws"})
	if err != nil || got != " kept as sent " {
		t.Fatalf("Ask = %q, %v", got, err)
	}
	if asked.text != "Host" || asked.fallback != "ws" || asked.hidden {
		t.Errorf("asked %+v", asked)
	}
	if _, err = b.AskHidden(t.Context(), "Password"); err != nil {
		t.Fatal(err)
	}
	if asked.text != "Password" || !asked.hidden || asked.id != 2 {
		t.Errorf("asked %+v, want the hidden question, as the second", asked)
	}
}
