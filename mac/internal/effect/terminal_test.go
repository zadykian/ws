package effect_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/zadykian/ws/mac/internal/effect"
)

func TestTerminalAsk(t *testing.T) {
	notEmpty := func(s string) error {
		if s == "" {
			return errors.New("empty")
		}
		return nil
	}
	tests := []struct {
		name    string
		input   string
		q       effect.Question
		want    string
		wantOut string
		wantErr bool
	}{
		{"answer", "mac\n", effect.Question{Text: "Name"}, "mac", "Name: ", false},
		{"default", "\n", effect.Question{Text: "Name", Default: "ws"}, "ws", "Name [ws]: ", false},
		{"again", "\nmac\n", effect.Question{Text: "Name", Valid: notEmpty}, "mac", "empty\nName: ", false},
		{"last line without newline", "mac", effect.Question{Text: "Name"}, "mac", "", false},
		{"end of input", "", effect.Question{Text: "Name"}, "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var w bytes.Buffer
			got, err := effect.NewTerminal(strings.NewReader(tt.input), &w).Ask(context.Background(), tt.q)
			if (err != nil) != tt.wantErr || got != tt.want {
				t.Errorf("Ask = %q, %v; want %q, error %v", got, err, tt.want, tt.wantErr)
			}
			if !strings.Contains(w.String(), tt.wantOut) {
				t.Errorf("output = %q, want it to contain %q", w.String(), tt.wantOut)
			}
		})
	}
}

func TestTerminalAskCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := effect.NewTerminal(strings.NewReader("x\n"), &bytes.Buffer{}).Ask(ctx, effect.Question{Text: "Q"})
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}

func TestTerminalOut(t *testing.T) {
	var w bytes.Buffer
	term := effect.NewTerminal(strings.NewReader(""), &w)
	term.Say("hello %s", "mac")
	term.Warn("careful")
	term.ByHand("open %s", "System Settings")
	term.Diff("f", []byte("a\n"), []byte("b\n"))
	term.Flush()
	term.Flush()
	want := "hello mac\nwarning: careful\n--- f\n+++ f\n@@ -1 +1 @@\n-a\n+b\n" +
		"To do by hand:\n  - open System Settings\n"
	if w.String() != want {
		t.Errorf("output = %q, want %q", w.String(), want)
	}
}
