package fans

import (
	"fmt"
	"strconv"
	"strings"
)

// parseGPULine reads a line of `--query-gpu=index,temperature.gpu --format=csv,noheader,nounits`, as
// "0, 45".
func parseGPULine(line string) (int, float64, error) {
	index, temp, ok := strings.Cut(line, ",")
	if !ok {
		return 0, 0, fmt.Errorf("%q is not an index and a temperature", line)
	}
	i, err := strconv.Atoi(strings.TrimSpace(index))
	if err != nil {
		return 0, 0, fmt.Errorf("%q: %w", line, err)
	}
	c, err := strconv.ParseFloat(strings.TrimSpace(temp), 64)
	if err != nil {
		return 0, 0, fmt.Errorf("%q: %w", line, err)
	}
	return i, c, nil
}
