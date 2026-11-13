package daemon

import (
	"sync"
	"time"
)

// maxAge is how old a GPU's reading may be before it counts as missing. nvidia-smi prints one a
// second.
const maxAge = 10 * time.Second

// reading is a GPU's temperature at a time.
type reading struct {
	celsius float64
	at      time.Time
}

// GPUs holds the latest temperature of each NVIDIA GPU, which nvidia-smi streams. ws-fans reads
// the temperature alone, as it read it through NVML.
type GPUs struct {
	// Now returns the time. Nil means time.Now.
	Now func() time.Time
	// Retry is how long Run waits to start the stream again. Zero means 5 s.
	Retry time.Duration

	mu      sync.Mutex
	temps   map[int]reading
	problem string // why there is no stream, or ""
}

func (g *GPUs) now() time.Time {
	if g.Now != nil {
		return g.Now()
	}
	return time.Now()
}

// record keeps a line's reading. A line that is not one is the stream's problem.
func (g *GPUs) record(line string) {
	i, c, err := parseGPULine(line)
	g.mu.Lock()
	defer g.mu.Unlock()
	if err != nil {
		g.problem = err.Error()
		return
	}
	if g.temps == nil {
		g.temps = map[int]reading{}
	}
	g.temps[i], g.problem = reading{c, g.now()}, ""
}

// Temperature returns the temperature in °C of the GPU of the index, or a missing error where
// nvidia-smi has not read it within maxAge.
func (g *GPUs) Temperature(gpu int) (float64, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	r, ok := g.temps[gpu]
	switch {
	case g.problem != "":
		return 0, missing("no NVML: %s", g.problem)
	case !ok || g.now().Sub(r.at) > maxAge:
		return 0, missing("no reading of GPU %d in %v", gpu, maxAge)
	}
	return r.celsius, nil
}
