package wizard

import (
	"context"
	"strings"
	"testing"
)

func TestHandClose(t *testing.T) {
	var b strings.Builder
	h := &Hand{}
	h.Close(context.Background(), &b)
	if b.Len() != 0 {
		t.Errorf("an empty list printed %q", b.String())
	}
	h.Add("a %d", 1)
	h.Late(func(_ context.Context, h *Hand) { h.Add("b") })
	h.Close(context.Background(), &b)
	if want := "\nWhat stays by hand:\n- a 1\n- b\n"; b.String() != want {
		t.Errorf("Close printed %q; want %q", b.String(), want)
	}
}
