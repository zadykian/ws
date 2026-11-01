package base

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

func TestSysctlFileLimits(t *testing.T) {
	for value, ok := range map[int]bool{-1: false, 0: true, 10: true, 200: true, 201: false} {
		_, err := sysctlFile{M: effecttest.NewMachine(), Swappiness: value}.Plan(t.Context())
		if (err == nil) != ok {
			t.Errorf("swappiness %d: error = %v, want ok = %v", value, err, ok)
		}
		if err != nil && !strings.Contains(err.Error(), "base_swappiness") {
			t.Errorf("swappiness %d: error %v does not name base_swappiness", value, err)
		}
	}
}
