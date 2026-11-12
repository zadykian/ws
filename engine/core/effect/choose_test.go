package effect_test

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect"
)

type choosing struct {
	effect.Prompt
	got effect.ChoiceQuestion
}

func (c *choosing) Choose(_ context.Context, q effect.ChoiceQuestion) (string, error) {
	c.got = q
	return "b", nil
}

func TestChooseUsesChoosing(t *testing.T) {
	c := &choosing{}
	q := effect.ChoiceQuestion{Text: "Pick", Options: []string{"a", "b"}, Default: "a"}
	got, err := effect.Choose(context.Background(), c, q)
	if err != nil || got != "b" || c.got.Text != "Pick" {
		t.Fatalf("got %q, %v with %+v", got, err, c.got)
	}
}

func TestChooseFallbackAsksALine(t *testing.T) {
	var out strings.Builder
	in := strings.NewReader("c\n\n")
	term := effect.NewTerminal(in, &out)
	q := effect.ChoiceQuestion{Text: "Pick", Options: []string{"a", "b"}, Default: "b"}
	got, err := effect.Choose(context.Background(), term, q)
	if err != nil || got != "b" {
		t.Fatalf("got %q, %v", got, err)
	}
	if !strings.Contains(out.String(), "Pick (a, b) [b]: ") || !strings.Contains(out.String(), "answer one of a, b") {
		t.Errorf("the lines were %q", out.String())
	}
}

func TestChooseFallbackUnknownDefault(t *testing.T) {
	term := effect.NewTerminal(strings.NewReader("\n"), &strings.Builder{})
	got, err := effect.Choose(context.Background(), term, effect.ChoiceQuestion{Text: "Pick", Options: []string{"a", "b"}, Default: "z"})
	if err != nil || got != "a" {
		t.Fatalf("got %q, %v", got, err)
	}
}

func TestChooseNoOptions(t *testing.T) {
	term := effect.NewTerminal(strings.NewReader(""), &strings.Builder{})
	if _, err := effect.Choose(context.Background(), term, effect.ChoiceQuestion{Text: "Pick"}); err == nil {
		t.Fatal("no options asked a question")
	}
}
