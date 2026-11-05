package tui

import (
	"io"
	"strings"
	"sync"
)

// record keeps a run's words past the screen: the file that holds all of them. A window and its
// scrollback go with the alternate screen, and the log is what the maintainer reads after it.
type record struct {
	lock sync.Mutex
	// log takes every line of the run, as clean makes it. Nil where there is none.
	log io.Writer
}

// note adds msg, as the bridge sends it to the screen. Only lines matter.
func (r *record) note(msg any) {
	r.lock.Lock()
	defer r.lock.Unlock()
	if l, ok := msg.(Line); ok {
		r.write(cleanLines(l.Text))
	}
}

// write adds a line to the log. A log that fails stops, and the run goes on without it.
func (r *record) write(text string) {
	if r.log == nil {
		return
	}
	if _, err := io.WriteString(r.log, text+"\n"); err != nil {
		r.log = nil
	}
}

// close ends the log, as a work left behind may still send lines.
func (r *record) close() {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.log = nil
}

// cleanLines applies clean to each line of a text, as a diff holds several.
func cleanLines(text string) string {
	lines := strings.Split(text, "\n")
	for i, l := range lines {
		lines[i] = clean(l)
	}
	return strings.Join(lines, "\n")
}
