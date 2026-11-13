package conf

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

func TestConfFileAdd(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["mkdir -p /h/.config/ws"] = effecttest.Result{}
	var out effecttest.Out
	f := File{Path: "/h/.config/ws/mac.conf", Stage: "Answers", M: m, Out: &out}
	if err := f.Add(context.Background(), "WS_TARGET=lan\n"); err != nil {
		t.Fatal(err)
	}
	if err := f.Add(context.Background(), "WS_LAN=10.88.0.0/24\nWS_LAN_ADDRESS=10.88.0.10\n"); err != nil {
		t.Fatal(err)
	}
	if got := string(m.Files[f.Path]); got != "WS_TARGET=lan\nWS_LAN=10.88.0.0/24\nWS_LAN_ADDRESS=10.88.0.10\n" || m.Modes[f.Path] != 0o644 {
		t.Errorf("mac.conf = %q, mode %o", got, m.Modes[f.Path])
	}
}

func TestConfFileRefusesAndLeavesLinks(t *testing.T) {
	m := effecttest.NewMachine()
	var out effecttest.Out
	f := File{Path: "/h/mac.conf", Stage: "Answers", M: m, Out: &out}
	if err := f.Add(context.Background(), "WS_LAN=10.88.0.0/24\n"); !errors.Is(err, ErrRefused) {
		t.Errorf("a WS_LAN without its address: %v, want a refusal", err)
	}
	if !strings.Contains(out.Lines[0], "/h/mac.conf: ") || len(m.Files) != 0 {
		t.Errorf("out = %v, files = %v; want a message that names mac.conf and no file", out.Lines, m.Files)
	}
	m.Links[f.Path] = "/dot/mac.conf"
	out.Lines = nil
	if err := f.Add(context.Background(), "WS_TARGET=lan\n"); err == nil || len(out.Lines) != 2 || len(m.Files) != 0 {
		t.Errorf("a link: %v, out = %v, files = %v", err, out.Lines, m.Files)
	}
}
