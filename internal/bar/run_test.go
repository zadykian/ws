package bar

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

func TestRunWritesTheLineThenRemovesIt(t *testing.T) {
	file := filepath.Join(t.TempDir(), "line")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	command := func(name string, _ ...string) func(context.Context) (io.ReadCloser, error) {
		if name != "nvidia-smi" {
			t.Errorf("command ran %s, want nvidia-smi", name)
		}
		return func(context.Context) (io.ReadCloser, error) {
			return io.NopCloser(strings.NewReader("0, 5, 33\n")), nil
		}
	}
	b := &Bar{FS: fstest.MapFS{"proc/meminfo": {Data: []byte("MemTotal: 10 kB\nMemAvailable: 5 kB\n")}}, Interval: 100 * time.Millisecond}
	done := make(chan error)
	go func() { done <- b.Run(ctx, file, command) }()
	want := "gpu 5% 33C  ram 50%\n"
	for deadline := time.Now().Add(5 * time.Second); ; time.Sleep(time.Millisecond) {
		if data, _ := os.ReadFile(file); string(data) == want { //nolint:gosec // the test's own file
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("no line %q in %s", want, file)
		}
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(file); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("the line outlives the run: %v", err)
	}
}
