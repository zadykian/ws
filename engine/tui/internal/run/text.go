package run

import (
	"bytes"
	"io"
	"sync"
)

// runText holds what a run's work writes to stdout, which waits for the screen's end: it would
// paint over the screen. A step's goroutine may write past the work's end, hence the lock.
type runText struct {
	lock sync.Mutex
	buf  bytes.Buffer
}

// Write adds p to the text.
func (t *runText) Write(p []byte) (int, error) {
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.buf.Write(p)
}

// String returns the text so far.
func (t *runText) String() string {
	t.lock.Lock()
	defer t.lock.Unlock()
	return t.buf.String()
}

// writeTo writes the text to w, and drops it where w is nil.
func (t *runText) writeTo(w io.Writer) {
	if w == nil {
		return
	}
	t.lock.Lock()
	defer t.lock.Unlock()
	_, _ = t.buf.WriteTo(w)
}
