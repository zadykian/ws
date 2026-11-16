package values

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
	f := File{Path: "/h/.config/ws/values.yml", Stage: "Answers", M: m, Out: &out}
	if err := f.Add(context.Background(), Text("WS_TARGET", "lan")); err != nil {
		t.Fatal(err)
	}
	if err := f.Add(context.Background(), Text("WS_LAN", "10.88.0.0/24", "WS_LAN_ADDRESS", "10.88.0.10")); err != nil {
		t.Fatal(err)
	}
	if got := string(m.Files[f.Path]); got != Text("WS_TARGET", "lan", "WS_LAN", "10.88.0.0/24", "WS_LAN_ADDRESS", "10.88.0.10") || m.Modes[f.Path] != 0o644 {
		t.Errorf("values.yml = %q, mode %o", got, m.Modes[f.Path])
	}
}

func TestConfFileRefusesAndLeavesLinks(t *testing.T) {
	m := effecttest.NewMachine()
	var out effecttest.Out
	f := File{Path: "/h/values.yml", Stage: "Answers", M: m, Out: &out}
	if err := f.Add(context.Background(), Text("WS_LAN", "10.88.0.0/24")); !errors.Is(err, ErrRefused) {
		t.Errorf("a WS_LAN without its address: %v, want a refusal", err)
	}
	if !strings.Contains(out.Lines[0], "/h/values.yml: ") || len(m.Files) != 0 {
		t.Errorf("out = %v, files = %v; want a message that names values.yml and no file", out.Lines, m.Files)
	}
	m.Links[f.Path] = "/dot/values.yml"
	out.Lines = nil
	if err := f.Add(context.Background(), Text("WS_TARGET", "lan")); err == nil || len(out.Lines) != 2 || len(m.Files) != 0 {
		t.Errorf("a link: %v, out = %v, files = %v", err, out.Lines, m.Files)
	}
}

func TestConfFileAddKeepsAValueSetByHand(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/h/values.yml"] = []byte(Text("WS_CABLE_PORT", "eth1"))
	f := File{Path: "/h/values.yml", Stage: "Cable", M: m, Out: &effecttest.Out{}}
	m.Results["mkdir -p /h"] = effecttest.Result{}
	if err := f.Add(context.Background(), Text("WS_CABLE_PORT", "eth2", "WS_CABLE_SERVICE", "USB")); err != nil {
		t.Fatal(err)
	}
	if got, want := string(m.Files[f.Path]), Text("WS_CABLE_PORT", "eth1", "WS_CABLE_SERVICE", "USB"); got != want {
		t.Errorf("values.yml = %q; want %q", got, want)
	}
}

func TestWriteInRootsDirectoryGoesThroughSudo(t *testing.T) {
	m := effecttest.NewMachine()
	var out effecttest.Out
	f := File{Path: "/etc/ws/servers/lab/values.yml", Stage: "Answers", M: m, Out: &out}
	if err := f.Write(context.Background(), Text("WS_TARGET", "lan")); err != nil {
		t.Fatal(err)
	}
	if string(m.Files[f.Path]) != Text("WS_TARGET", "lan") || m.Modes[f.Path] != 0o644 || m.Owners[f.Path] != [2]int{} {
		t.Errorf("file = %q, mode %o, owner %v; want root's at 644", m.Files[f.Path], m.Modes[f.Path], m.Owners[f.Path])
	}
	if _, ok := m.Dirs["/etc/ws/servers/lab"]; !ok {
		t.Errorf("dirs = %v, want /etc/ws/servers/lab made", m.Dirs)
	}
	for _, c := range m.Calls {
		if strings.HasPrefix(c, "run ") {
			t.Errorf("call %q: the user ran a command", c)
		}
	}
}

func TestWriteUnderTheHomeStaysTheUsers(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results["mkdir -p /h/ws"] = effecttest.Result{}
	m.Files["/h/ws/values.yml"] = []byte("old\n")
	m.Modes["/h/ws/values.yml"] = 0o600
	f := File{Path: "/h/ws/values.yml", Stage: "Answers", M: m, Out: &effecttest.Out{}}
	if err := f.Write(context.Background(), "new\n"); err != nil {
		t.Fatal(err)
	}
	for _, c := range m.Calls {
		if strings.HasPrefix(c, "root ") {
			t.Errorf("call %q: sudo wrote a file in the home", c)
		}
	}
	if string(m.Files[f.Path]) != "new\n" || m.Modes[f.Path] != 0o600 {
		t.Errorf("file = %q, mode %o; want the text, mode kept", m.Files[f.Path], m.Modes[f.Path])
	}
}
