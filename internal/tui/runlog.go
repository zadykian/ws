package tui

import (
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/zadykian/ws/step"
)

// tailLines is how many of a failed step's last lines stay for the end of a run.
const tailLines = 30

// record keeps a run's words past the screen: the file that holds all of them, the warnings and the
// changes that --check planned, and the end of the failed step's output. A window and its scrollback
// go with the alternate screen, and these are what the maintainer reads after it.
type record struct {
	lock sync.Mutex
	// log takes every line of the run, as clean makes it. Nil where there is none.
	log io.Writer
	// kept are the warnings and the lines of --check's plan, in the order of the run.
	kept []Line
	// recent are the last lines since a step last ended: the current step's, its Plan's included.
	recent []string
	// failed names the step that failed, and tail holds its recent lines.
	failed string
	tail   []string
}

// note adds msg, as the bridge sends it to the screen. Only lines and events matter.
func (r *record) note(msg any) {
	r.lock.Lock()
	defer r.lock.Unlock()
	switch m := msg.(type) {
	case Line:
		text := cleanLines(m.Text)
		r.write(text)
		r.recent = append(r.recent, strings.Split(text, "\n")...)
		if len(r.recent) > tailLines {
			r.recent = r.recent[len(r.recent)-tailLines:]
		}
	case step.Event:
		r.event(m)
	}
}

// event logs how a step ended, and keeps a failed step's output. recent clears at a step's last event
// and not at Started, which a failed Plan and --check never send. A stage fails behind the step that
// failed in it, whose output is the one to keep.
func (r *record) event(e step.Event) {
	name := e.Step
	if e.Parent != "" {
		name = e.Parent + ": " + e.Step
	}
	switch e.Kind {
	case step.Planned, step.Started:
		return
	case step.Failed:
		r.write(fmt.Sprintf("%s: failed: %v", name, e.Err))
		if r.failed == "" {
			r.failed, r.tail = name, r.recent
		}
	case step.Skipped:
		r.write(name + ": skipped")
	}
	r.recent = nil
}

// keep holds a warning or a change line of --check for the end of the run.
func (r *record) keep(l Line) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.kept = append(r.kept, l)
}

// logText adds text to the log alone, cleaned, and without its last newline. Empty text adds nothing.
func (r *record) logText(text string) {
	text = strings.TrimSuffix(text, "\n")
	if text == "" {
		return
	}
	r.lock.Lock()
	defer r.lock.Unlock()
	r.write(cleanLines(text))
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

// report prints what a run leaves on the normal screen: its warnings, --check's changes, the end of
// a failed step's output, and the log's path where there is something to read in it.
func (r *record) report(w io.Writer, path string, cut bool) {
	r.lock.Lock()
	defer r.lock.Unlock()
	var b strings.Builder
	for _, l := range r.kept {
		b.WriteString(cleanLines(l.Text) + "\n")
	}
	if r.failed != "" {
		fmt.Fprintf(&b, "%s failed.", r.failed)
		if len(r.tail) > 0 {
			b.WriteString(" The end of its output:")
		}
		b.WriteByte('\n')
		for _, line := range r.tail {
			b.WriteString("  " + line + "\n")
		}
	}
	if path != "" && (cut || r.failed != "" || len(r.kept) > 0) {
		fmt.Fprintf(&b, "The whole output is in %s\n", path)
	}
	_, _ = io.WriteString(w, b.String())
}

// cleanLines applies clean to each line of a text, as a diff holds several.
func cleanLines(text string) string {
	lines := strings.Split(text, "\n")
	for i, l := range lines {
		lines[i] = clean(l)
	}
	return strings.Join(lines, "\n")
}
