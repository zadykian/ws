package values

import (
	"strings"
	"testing"
)

func TestBaseRefusesANegativeSwapSize(t *testing.T) {
	for in, ok := range map[string]bool{"": true, "WS_BASE_SWAP_SIZE_MB: 0\n": true, "WS_BASE_SWAP_SIZE_MB: 64\n": true,
		"WS_BASE_SWAP_SIZE_MB: -1\n": false} {
		_, err := Parse([]byte(in))
		if ok != (err == nil) || !ok && !strings.Contains(err.Error(), "WS_BASE_SWAP_SIZE_MB is -1") {
			t.Errorf("Parse(%q) = %v, want taken %v", in, err, ok)
		}
	}
}

func TestBaseRefusesASwappinessOutOfRange(t *testing.T) {
	for value, ok := range map[string]bool{"0": true, "200": true, "-1": false, "201": false} {
		_, err := Parse([]byte("WS_BASE_SWAPPINESS: " + value + "\n"))
		if ok != (err == nil) || !ok && !strings.Contains(err.Error(), "WS_BASE_SWAPPINESS is "+value+": it takes") {
			t.Errorf("WS_BASE_SWAPPINESS %s: Parse = %v, want taken %v", value, err, ok)
		}
	}
}
