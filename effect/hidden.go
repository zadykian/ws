package effect

import (
	"context"
	"fmt"

	"golang.org/x/term"
)

// AskHidden prints text and reads a line with no echo. It reads the terminal itself, past t.in's
// buffer, as term.ReadPassword needs its descriptor; a terminal gives a line a read, so no line is
// lost. Ctrl-C ends ctx, as ISIG stays on: the read's goroutine stays blocked, so echo comes back
// here.
func (t *Terminal) AskHidden(ctx context.Context, text string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if t.file == nil || !term.IsTerminal(int(t.file.Fd())) {
		return "", fmt.Errorf("reading the answer to %q: no terminal", text)
	}
	fd := int(t.file.Fd())
	state, err := term.GetState(fd)
	if err != nil {
		return "", fmt.Errorf("reading the answer to %q: %w", text, err)
	}
	_, _ = fmt.Fprintf(t.w, "%s: ", text)
	type answer struct {
		line []byte
		err  error
	}
	done := make(chan answer, 1)
	go func() {
		line, err := term.ReadPassword(fd)
		done <- answer{line, err}
	}()
	select {
	case a := <-done:
		_, _ = fmt.Fprintln(t.w)
		if a.err != nil {
			return "", fmt.Errorf("reading the answer to %q: %w", text, a.err)
		}
		return string(a.line), nil
	case <-ctx.Done():
		_ = term.Restore(fd, state)
		_, _ = fmt.Fprintln(t.w)
		return "", ctx.Err()
	}
}
