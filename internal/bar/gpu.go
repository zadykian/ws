package bar

import (
	"bufio"
	"context"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"
)

// smiArgs make nvidia-smi print each GPU's index, use and temperature, a line each, every
// interval. One process serves the whole run, as each start of nvidia-smi loads the driver's
// library.
func smiArgs(interval time.Duration) []string {
	seconds := max(int(interval/time.Second), 1)
	return []string{"--query-gpu=index,utilization.gpu,temperature.gpu", "--format=csv,noheader,nounits", "-l", strconv.Itoa(seconds)}
}

// gpuReading is a GPU's use, in percent, and temperature, in °C, at a time.
type gpuReading struct {
	use, celsius int
	at           time.Time
}

// gpus holds the latest reading of each NVIDIA GPU, which nvidia-smi streams.
type gpus struct {
	now      func() time.Time
	mu       sync.Mutex
	readings map[string]gpuReading // by the GPU's index
}

// record keeps a line's reading. A line that holds none, as an error or "[N/A]", is dropped.
func (g *gpus) record(line string) {
	fields := strings.Split(line, ",")
	if len(fields) != 3 {
		return
	}
	use, err := strconv.Atoi(strings.TrimSpace(fields[1]))
	if err != nil {
		return
	}
	celsius, err := strconv.Atoi(strings.TrimSpace(fields[2]))
	if err != nil {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.readings == nil {
		g.readings = map[string]gpuReading{}
	}
	g.readings[strings.TrimSpace(fields[0])] = gpuReading{use, celsius, g.now()}
}

// reading returns the highest use and temperature of the GPUs read within maxAge. A GPU that
// nvidia-smi stopped reading, or a machine with none, has no reading.
func (g *gpus) reading(maxAge time.Duration) (use, celsius int, ok bool) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, r := range g.readings {
		if g.now().Sub(r.at) > maxAge {
			continue
		}
		use, celsius, ok = max(use, r.use), max(celsius, r.celsius), true
	}
	return use, celsius, ok
}

// run reads the stream of start until ctx ends. Where the stream ends or cannot start, as on a
// machine with no driver, run starts it again after wait.
func (g *gpus) run(ctx context.Context, start func(context.Context) (io.ReadCloser, error), wait time.Duration) {
	for ctx.Err() == nil {
		if r, err := start(ctx); err == nil {
			sc := bufio.NewScanner(r)
			for sc.Scan() {
				g.record(sc.Text())
			}
			_ = r.Close() // a read only stream
		}
		select {
		case <-ctx.Done():
		case <-time.After(wait):
		}
	}
}
