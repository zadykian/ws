package effect

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
)

// Terminal is a Prompt and an Out over a reader and a writer.
type Terminal struct {
	in     *bufio.Reader
	w      io.Writer
	byHand []string
}

// NewTerminal returns a Terminal that reads answers from in and writes to w.
func NewTerminal(in io.Reader, w io.Writer) *Terminal {
	return &Terminal{in: bufio.NewReader(in), w: w}
}

// Ask prints the question with its default, and reads a line. An empty line takes the default.
func (t *Terminal) Ask(ctx context.Context, q Question) (string, error) {
	for {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		if q.Default != "" {
			_, _ = fmt.Fprintf(t.w, "%s [%s]: ", q.Text, q.Default)
		} else {
			_, _ = fmt.Fprintf(t.w, "%s: ", q.Text)
		}
		line, err := t.in.ReadString('\n')
		if err != nil && (!errors.Is(err, io.EOF) || line == "") {
			return "", fmt.Errorf("reading the answer to %q: %w", q.Text, err)
		}
		answer := strings.TrimSpace(line)
		if answer == "" {
			answer = q.Default
		}
		if q.Valid == nil {
			return answer, nil
		}
		verr := q.Valid(answer)
		if verr == nil {
			return answer, nil
		}
		_, _ = fmt.Fprintf(t.w, "%v\n", verr)
	}
}

// Say prints a line.
func (t *Terminal) Say(format string, a ...any) {
	_, _ = fmt.Fprintf(t.w, format+"\n", a...)
}

// Warn prints a line that starts with "warning: ".
func (t *Terminal) Warn(format string, a ...any) {
	_, _ = fmt.Fprintf(t.w, "warning: "+format+"\n", a...)
}

// Diff prints the unified diff of before and after, and nothing where they are equal.
func (t *Terminal) Diff(name string, before, after []byte) {
	_, _ = io.WriteString(t.w, Diff(name, before, after))
}

// ByHand keeps a line for Flush.
func (t *Terminal) ByHand(format string, a ...any) {
	t.byHand = append(t.byHand, fmt.Sprintf(format, a...))
}

// Flush prints the lines that ByHand kept, once, and forgets them.
func (t *Terminal) Flush() {
	if len(t.byHand) == 0 {
		return
	}
	_, _ = io.WriteString(t.w, "To do by hand:\n")
	for _, l := range t.byHand {
		_, _ = fmt.Fprintf(t.w, "  - %s\n", l)
	}
	t.byHand = nil
}
