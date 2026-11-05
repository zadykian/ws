package tui

import (
	"bytes"
	"sync"
)

// pipeCap bounds a line that no line end closes, such as a meter of \r alone.
const pipeCap = 64 << 10

// pipe turns what a step's commands print into lines for the screen, as OSMachine.Stderr. It is a
// pointer, as os/exec keeps to one goroutine a command only for a Stdout and Stderr that compare
// equal, and it locks, as a run's commands can overlap.
type pipe struct {
	bridge *Bridge
	lock   sync.Mutex
	// rest is the line that has no end yet.
	rest []byte
}

// Write sends each whole line at once. One message a write keeps a loud command from drawing a
// frame a line, and rest waits for its end, as a write can stop in a line or in a character.
func (p *pipe) Write(data []byte) (int, error) {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.rest = append(p.rest, data...)
	if end := bytes.LastIndexByte(p.rest, '\n'); end >= 0 {
		p.bridge.send(Line{Piped, string(p.rest[:end])})
		p.rest = append(p.rest[:0], p.rest[end+1:]...)
	}
	if len(p.rest) > pipeCap {
		p.send()
	}
	return len(data), nil
}

// flush sends what has no line end, as a command has ended that printed it.
func (p *pipe) flush() {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.send()
}

func (p *pipe) send() {
	if len(p.rest) > 0 {
		p.bridge.send(Line{Piped, string(p.rest)})
		p.rest = p.rest[:0]
	}
}
