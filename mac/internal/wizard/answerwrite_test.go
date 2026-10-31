package wizard

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/zadykian/ws/mac/internal/effect/effecttest"
)

func TestConfFileAdd(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["mkdir -p /h/.config/ws"] = effecttest.Result{}
	var out effecttest.Out
	f := confFile{"/h/.config/ws/mac.conf", "Answers", m, &out}
	if err := f.add(context.Background(), "WS_TARGET=lan\n"); err != nil {
		t.Fatal(err)
	}
	if err := f.add(context.Background(), "WS_LAN=10.88.0.0/24\nWS_LAN_ADDRESS=10.88.0.10\n"); err != nil {
		t.Fatal(err)
	}
	if got := string(m.Files[f.path]); got != "WS_TARGET=lan\nWS_LAN=10.88.0.0/24\nWS_LAN_ADDRESS=10.88.0.10\n" || m.Modes[f.path] != 0o644 {
		t.Errorf("mac.conf = %q, mode %o", got, m.Modes[f.path])
	}
}

func TestConfFileRefusesAndLeavesLinks(t *testing.T) {
	m := effecttest.NewMachine()
	var out effecttest.Out
	f := confFile{"/h/mac.conf", "Answers", m, &out}
	if err := f.add(context.Background(), "WS_LAN=10.88.0.0/24\n"); !errors.Is(err, errRefused) {
		t.Errorf("a WS_LAN without its address: %v, want a refusal", err)
	}
	if !strings.Contains(out.Lines[0], "/h/mac.conf: ") || len(m.Files) != 0 {
		t.Errorf("out = %v, files = %v; want a message that names mac.conf and no file", out.Lines, m.Files)
	}
	m.Links[f.path] = "/dot/mac.conf"
	out.Lines = nil
	if err := f.add(context.Background(), "WS_TARGET=lan\n"); err == nil || len(out.Lines) != 2 || len(m.Files) != 0 {
		t.Errorf("a link: %v, out = %v, files = %v", err, out.Lines, m.Files)
	}
}
