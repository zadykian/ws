package answers

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
)

// TestAnswersStopsOnARefusedSibling keeps the new server from taking the slot and ports of a file
// that apply refuses: no file is written, and the refusal names the sibling.
func TestAnswersStopsOnARefusedSibling(t *testing.T) {
	a, out := newAnswers(t, "WS_GIT_NAME=ada\nWS_GIT_EMAIL=a@b.c\n", "")
	m := a.M.(*effecttest.Machine)
	m.Dirs["/h/.config/ws/servers"], m.Dirs["/h/.config/ws/servers/lab"] = 0o755, 0o755
	m.Files["/h/.config/ws/servers/lab/server.conf"] = []byte("WS_GIT_NAME=ada\n")
	m.Results["mkdir -p /h/.config/ws"] = effecttest.Result{}
	_, err := step.Run(context.Background(), []step.Step{homebrewDone{}, a}, out, false)
	if err == nil || !strings.Contains(err.Error(), "server lab: ") {
		t.Fatalf("Run = %v, want lab's refusal", err)
	}
	if _, ok := m.Files[serverPathForTest]; ok {
		t.Error("the stage wrote the new server's file beside a refused one")
	}
}
