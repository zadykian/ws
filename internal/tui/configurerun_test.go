package tui

import (
	"bytes"
	"context"
	"errors"
	"io"
	"maps"
	"syscall"
	"testing"
	"time"
)

func TestConfigureSavesAndLeaves(t *testing.T) {
	in, keys := io.Pipe()
	defer func() { _ = keys.Close() }()
	saved := make(chan map[string]string, 1)
	sheet := wayFile()
	sheet.Save = func(_ context.Context, b *Bridge, edits map[string]string) (Sheet, error) {
		b.Say("wrote way.conf")
		saved <- edits
		return wayFile(), nil
	}
	ended := make(chan error, 1)
	var out bytes.Buffer
	go func() {
		ended <- Configure(t.Context(), Config{In: in, Out: &out, Theme: testTheme, Cut: testCut}, "configure", sheet)
	}()
	if _, err := keys.Write([]byte("\t2222\x13")); err != nil {
		t.Fatal(err)
	}
	if got, want := <-saved, (map[string]string{"PORT": "2222"}); !maps.Equal(got, want) {
		t.Errorf("edits = %v, want %v", got, want)
	}
	// The keys wait for the save's end, so Esc goes on until one lands past it, each far enough apart
	// that no two read as one key.
	for {
		select {
		case err := <-ended:
			if err != nil {
				t.Errorf("Configure: %v", err)
			}
			return
		case <-time.After(150 * time.Millisecond):
			if _, err := keys.Write([]byte("\x1b")); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestConfigureCtrlCEndsWithTheCut(t *testing.T) {
	c := noKeys(&bytes.Buffer{})
	in, keys := io.Pipe()
	defer func() { _ = keys.Close() }()
	c.In = in
	go func() { _, _ = keys.Write([]byte("\x03")) }()
	if err := Configure(t.Context(), c, "configure", wayFile()); !errors.Is(err, cutError{syscall.SIGINT}) {
		t.Errorf("Configure: %v", err)
	}
}
