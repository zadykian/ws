package effect

import (
	"bytes"
	"context"
	"io"
	"strings"
)

// PassLines runs the command as PassInput does, with take before Stderr. Its standard output and
// error share a pipe, so its lines keep the order they have on Stderr. A process that it leaves
// behind, as ssh leaves a master, may hold that pipe, which a file passed to a plain Pass never
// waits on: WaitDelay bounds that, screen or no screen.
func (m OSMachine) PassLines(ctx context.Context, stdin []byte, take func(line string) bool, name string, args ...string) error {
	lines := &lineFilter{take: take, out: m.stderr()}
	cmd := m.command(ctx, append([]string{name}, args...))
	cmd.Stdin, cmd.Stdout, cmd.Stderr = bytes.NewReader(stdin), lines, lines
	cmd.WaitDelay = screenGrace
	err := cmd.Run()
	if last := lines.end(); err == nil {
		err = last
	}
	return finished(cmd, err)
}

// lineFilter is a Writer that offers each line to take, and passes on what take leaves as it came.
type lineFilter struct {
	take func(line string) bool
	out  io.Writer
	// part is what came after the last newline.
	part []byte
}

func (f *lineFilter) Write(p []byte) (int, error) {
	f.part = append(f.part, p...)
	for {
		i := bytes.IndexByte(f.part, '\n')
		if i < 0 {
			return len(p), nil
		}
		line := f.part[:i+1]
		f.part = f.part[i+1:]
		if err := f.offer(line); err != nil {
			return 0, err
		}
	}
}

// end offers a last line that no newline ended, such as one that a cut left.
func (f *lineFilter) end() error {
	if len(f.part) == 0 {
		return nil
	}
	return f.offer(f.part)
}

func (f *lineFilter) offer(line []byte) error {
	if f.take(strings.TrimSuffix(string(line), "\n")) {
		return nil
	}
	_, err := f.out.Write(line)
	return err
}
