package effect_test

import (
	"bufio"
	"context"
	"io"
	"strconv"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/zadykian/ws/effect"
)

// A pipe of the test's own stands for a screen's: exec copies to it, as to any Stderr that is no file.
func TestPassUnderAScreenEnds(t *testing.T) {
	tests := []struct {
		name, script, want string
		cut, fails         bool
	}{
		{"SIGTERM first", `trap 'echo term; exit 0' TERM; echo; while :; do sleep 0.05; done`, "term\n", true, true},
		{"a kill past the grace", `trap '' TERM; echo; while :; do sleep 0.05; done`, "", true, true},
		{"a child holds the pipe", `sleep 30 & echo "$!"; wait`, "", true, true},
		{"a passing command left a holder", `sleep 30 & echo "$!"`, "", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			r, w := io.Pipe()
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			m := effect.OSMachine{Stdin: effect.NoStdin, Stderr: w}
			var err error
			go func() {
				err = m.Pass(ctx, false, "sh", "-c", tt.script)
				_ = w.Close()
			}()
			text := bufio.NewReader(r)
			first, _ := text.ReadString('\n')
			start := time.Now()
			if tt.cut {
				cancel()
			}
			rest, _ := io.ReadAll(text)
			if (err != nil) != tt.fails || string(rest) != tt.want {
				t.Errorf("Pass = %v, printed %q; want a failure %v, and %q", err, rest, tt.fails, tt.want)
			}
			if took := time.Since(start); took > 10*time.Second {
				t.Errorf("Pass took %v", took)
			}
			if pid, err := strconv.Atoi(strings.TrimSpace(first)); err == nil { // The test's own sleep.
				_ = syscall.Kill(pid, syscall.SIGKILL)
			}
		})
	}
}
