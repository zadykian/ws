package cli

import (
	"context"
	"io"
	"slices"
	"strings"
	"testing"
	"time"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/tui"
	"github.com/zadykian/ws/roles/client/macos/values"
)

func TestMacConfigureAtATerminalDrawsAForm(t *testing.T) {
	p, m := macTerminal()
	writeConfs(m, "", values.Text("WS_USER", "ada"))
	status, stdout, stderr := runWs(p, "\x03", "client", "macos", "configure")
	if status != 130 || stdout != "" || !strings.Contains(stderr, enterScreen) || !strings.Contains(stderr, leaveScreen) {
		t.Errorf("status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}
}

func TestMacConfigureWritesWhatAFormSaves(t *testing.T) {
	p, m := macTerminal()
	writeConfs(m, "", values.Text("WS_USER", "ada"))
	m.Results["mkdir -p /h/.config/ws/servers/ws"] = effecttest.Result{}
	in, keys := io.Pipe()
	defer func() { _ = keys.Close() }()
	ended := make(chan int, 1)
	go func() {
		root := enginecli.Root(p, clientGroup(p), serverGroup(p))
		ended <- enginecli.Execute(context.Background(), root, []string{"client", "macos", "configure"}, in, io.Discard, io.Discard)
	}()
	if _, err := keys.Write([]byte("\x0e\t\tr.example\x13")); err != nil {
		t.Fatal(err)
	}
	// A save holds the keys, so Esc goes on until one lands past it, far enough apart to read as keys.
	for {
		select {
		case status := <-ended:
			if got, want := string(m.Files["/h/.config/ws/servers/ws/values.yml"]), values.Text("WS_USER", "ada", "WS_REMOTE", "r.example"); status != 0 || got != want {
				t.Errorf("status = %d, values.yml = %q; want %q", status, got, want)
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
	form, err := values.Form{Conf: "/h/.config/ws/values.yml", Server: "/h/.config/ws/servers/ws/values.yml", Scope: values.ServerFile, M: m}.
		Load(&effecttest.Out{})
	if err != nil {
		t.Fatal(err)
	}
	specs := confSheet("servers/ws/values.yml", form).Fields(map[string]string{"WS_TARGET": "cloud"})
	if got := specs[0]; got.Label != "WS_TARGET" || got.Kind != tui.ChoiceField || got.Value != "lan" {
		t.Errorf("target = %+v", got)
	}
	if got := specs[2]; got.Label != "WS_REMOTE_PORT" || got.Kind != tui.TextField || got.Default != "22" || got.Help == "" {
		t.Errorf("port = %+v", got)
	}
}

func TestMacConfigureRefusals(t *testing.T) {
	away, _ := macTerminal()
	away.TTY = false
	linux, _ := macTerminal()
	linux.GOOS = "linux"
	root, _ := macTerminal()
	root.Euid = 0
	bad, badFiles := macTerminal()
	badFiles.Files["/h/.config/ws/values.yml"] = []byte("no line\n")
	plain, _ := macTerminal()
	for name, tc := range map[string]struct {
		p    enginecli.Env
		args []string
		want string
	}{
		"no terminal": {away, []string{"client", "macos", "configure"}, "edit values.yml by hand"},
		"--plain":     {plain, []string{"--plain", "client", "macos", "configure"}, "edit values.yml by hand"},
		"linux":       {linux, []string{"client", "macos", "configure"}, "not Linux"},
		"root":        {root, []string{"client", "macos", "configure"}, "not root"},
		"a bad line":  {bad, []string{"client", "macos", "configure"}, "write one KEY: value a line"},
	} {
		status, stdout, stderr := runWs(tc.p, "", tc.args...)
		if status != 1 || stdout != "" || !strings.Contains(stderr, tc.want) || strings.Contains(stderr, enterScreen) {
			t.Errorf("%s: status = %d, stdout = %q, stderr = %q", name, status, stdout, stderr)
		}
	}
}

func TestMacFormsShowMacConfApartFromTheServersFile(t *testing.T) {
	p, m := macTerminal()
	writeConfs(m, "", values.Text("WS_USER", "ada"))
	sheets, done, err := macForms(strings.NewReader(""), io.Discard, p, new(string))(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer done(t.Context())
	if len(sheets) < 2 || sheets[0].Name != "values.yml" || sheets[1].Name != "servers/ws/values.yml" {
		t.Fatalf("sheets = %v; want values.yml, then the server's file", sheets)
	}
	var mac, server []string
	for _, f := range sheets[0].Fields(nil) {
		mac = append(mac, f.Label)
	}
	for _, f := range sheets[1].Fields(nil) {
		server = append(server, f.Label)
	}
	if len(mac) != 0 || !slices.Contains(server, "WS_TARGET") {
		t.Errorf("values.yml shows %v, the server's file %v", mac, server)
	}
}
