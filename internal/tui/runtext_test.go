package tui

import (
	"io"
	"sync"
	"testing"
)

func TestRunTextTakesWritesFromManyGoroutines(t *testing.T) {
	var text runText
	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() { _, _ = io.WriteString(&text, "a") })
	}
	wg.Wait()
	if got := text.String(); got != "aaaaaaaa" {
		t.Errorf("text = %q", got)
	}
}
