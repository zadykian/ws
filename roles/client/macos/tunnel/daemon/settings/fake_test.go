package settings

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

// peerKey is ws's key in tunnel.conf, and macKey the Mac's own, each 32 equal bytes in base64.
const (
	peerKey = "AQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQE="
	macKey  = "AgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgI="
)

var errTest = errors.New("no")

var t0 = time.Date(2026, 10, 31, 12, 0, 0, 0, time.UTC)

// fakeMachine answers each command with success, and keeps the lines it ran. A mv moves a file.
type fakeMachine struct {
	*effecttest.Machine
	mu   sync.Mutex
	runs []string
}

func (m *fakeMachine) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.runs = append(m.runs, strings.Join(append([]string{name}, args...), " "))
	if name == "mv" {
		m.Files[args[2]] = m.Files[args[1]]
		delete(m.Files, args[1])
	}
	return nil, nil
}
