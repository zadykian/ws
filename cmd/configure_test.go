package cmd

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/tui"
	"github.com/zadykian/ws/internal/wizard"
)

func TestMacConfigureAtATerminalDrawsAForm(t *testing.T) {
	p, _ := macTerminal()
	status, stdout, stderr := runWs(p, "\x03", "client", "macos", "configure")
	if status != 130 || stdout != "" || !strings.Contains(stderr, enterScreen) || !strings.Contains(stderr, leaveScreen) {
		t.Errorf("status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}
}

func TestMacConfigureWritesWhatAFormSaves(t *testing.T) {
	p, m := macTerminal()
	m.Files["/h/.config/ws/mac.conf"] = []byte("WS_USER=ada\n")
	m.Results["mkdir -p /h/.config/ws"] = effecttest.Result{}
	in, keys := io.Pipe()
	defer func() { _ = keys.Close() }()
	ended := make(chan int, 1)
	go func() {
		root := newRoot(p, newMacCmd(p), newServerCmd(p))
		ended <- execute(context.Background(), root, []string{"client", "macos", "configure"}, in, io.Discard, io.Discard)
	}()
	if _, err := keys.Write([]byte("\t\tr.example\x13")); err != nil {
		t.Fatal(err)
	}
	// A save holds the keys, so Esc goes on until one lands past it, far enough apart to read as keys.
	for {
		select {
		case status := <-ended:
			if got, want := string(m.Files["/h/.config/ws/mac.conf"]), "WS_USER=ada\nWS_REMOTE=r.example\n"; status != 0 || got != want {
				t.Errorf("status = %d, mac.conf = %q; want %q", status, got, want)
			}
			return
		case <-time.After(150 * time.Millisecond):
			if _, err := keys.Write([]byte("\x1b")); err != nil {
				t.Fatal(err)
			}
		}
	}
}

func TestConfSheetDrawsWhatTheFormGives(t *testing.T) {
	_, m := macTerminal()
	form, err := wizard.ConfForm{Conf: "/h/.config/ws/mac.conf", M: m}.Load(&effecttest.Out{})
	if err != nil {
		t.Fatal(err)
	}
	specs := confSheet(form).Fields(map[string]string{"WS_TARGET": "cloud"})
	if got := specs[0]; got.Label != "WS_TARGET" || got.Kind != tui.ChoiceField || got.Value != "lan" {
		t.Errorf("target = %+v", got)
	}
	if got := specs[2]; got.Label != "WS_REMOTE_PORT" || got.Kind != tui.TextField || got.Default != "22" || got.Help == "" {
		t.Errorf("port = %+v", got)
	}
}

func TestMacConfigureRefusals(t *testing.T) {
	away, _ := macTerminal()
	away.tty = false
	linux, _ := macTerminal()
	linux.goos = "linux"
	root, _ := macTerminal()
	root.euid = 0
	bad, badFiles := macTerminal()
	badFiles.Files["/h/.config/ws/mac.conf"] = []byte("no line\n")
	plain, _ := macTerminal()
	for name, tc := range map[string]struct {
		p    process
		args []string
		want string
	}{
		"no terminal": {away, []string{"client", "macos", "configure"}, "edit mac.conf by hand"},
		"--plain":     {plain, []string{"--plain", "client", "macos", "configure"}, "edit mac.conf by hand"},
		"linux":       {linux, []string{"client", "macos", "configure"}, "not Linux"},
		"root":        {root, []string{"client", "macos", "configure"}, "not root"},
		"a bad line":  {bad, []string{"client", "macos", "configure"}, "not a KEY=value line"},
	} {
		status, stdout, stderr := runWs(tc.p, "", tc.args...)
		if status != 1 || stdout != "" || !strings.Contains(stderr, tc.want) || strings.Contains(stderr, enterScreen) {
			t.Errorf("%s: status = %d, stdout = %q, stderr = %q", name, status, stdout, stderr)
		}
	}
}
