package setup

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/zadykian/ws/mac/internal/effect"
	"github.com/zadykian/ws/mac/internal/effect/effecttest"
)

func TestRootFilePlan(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files["/e/same"], m.Files["/e/other"], m.Files["/e/mine"] = []byte("a\n"), []byte("b\n"), []byte("a\n")
	m.Root["/e/mine"] = effect.Attrs{UID: 501, GID: 20, Mode: 0o644}
	tests := map[string]string{
		"/e/new":   "--- /dev/null\n+++ /e/new\n@@ -0,0 +1 @@\n+a\n",
		"/e/other": "--- /e/other\n+++ /e/other\n@@ -1 +1 @@\n-b\n+a\n",
		"/e/mine":  "would make /e/mine root's, with mode 644, rather than 501:20 644",
		"/e/same":  "",
	}
	for path, want := range tests {
		c, err := rootFile(m, path, 0o644, []byte("a\n"))
		if err != nil || (want == "") != (c == nil) || (c != nil && fmt.Sprint(c) != want) {
			t.Errorf("%s: %q, %v; want %q", path, c, err, want)
		}
	}
}

func TestApplyRootFile(t *testing.T) {
	m, out := effecttest.NewMachine(), &effecttest.Out{}
	m.Results["rm -f /e/gone"] = effecttest.Result{}
	for _, c := range []interface{ String() string }{
		RootWrite{Write{Path: "/e/w", After: []byte("x"), Mode: 0o600}},
		RootCopy{From: "/h/ws", To: "/e/ws", Data: []byte("bin"), Mode: 0o755},
		RootRemove{Path: "/e/gone"},
	} {
		if err := Apply(context.Background(), m, out, c); err != nil {
			t.Fatal(err)
		}
	}
	if got, want := strings.Join(out.Lines, ";"), "say: wrote /e/w;say: copied /h/ws to /e/ws;say: removed /e/gone"; got != want {
		t.Errorf("out = %q; want %q", got, want)
	}
	if string(m.Files["/e/ws"]) != "bin" || m.Modes["/e/ws"] != 0o755 {
		t.Errorf("the copy holds %q, mode %o", m.Files["/e/ws"], m.Modes["/e/ws"])
	}
}
