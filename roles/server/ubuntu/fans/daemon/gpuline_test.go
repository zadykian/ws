package fans

import "testing"

func TestParseGPULineReadsAnIndexAndATemperature(t *testing.T) {
	if i, c, err := parseGPULine("1, 45.5"); err != nil || i != 1 || c != 45.5 {
		t.Errorf("parse = %v, %v, %v; want 1, 45.5", i, c, err)
	}
	for _, line := range []string{"", "45", "x, 45", "0, hot", "0,"} {
		if _, _, err := parseGPULine(line); err == nil {
			t.Errorf("parseGPULine(%q) takes a line that is not an index and a temperature", line)
		}
	}
}
