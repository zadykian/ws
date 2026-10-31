package wizard

import (
	"strings"
	"testing"

	"github.com/zadykian/ws/mac/internal/effect/effecttest"
)

const confPathForTest = "/h/.config/ws/mac.conf"

func newAnswers(t *testing.T, text string) (*Answers, *effecttest.Out) {
	m := effecttest.NewMachine()
	if text != "" {
		m.Files[confPathForTest] = []byte(text)
	}
	m.Files["/r/defaults.yml"] = []byte("shell_git_name: ada\nshell_git_email: ada@example.com\n")
	out := &effecttest.Out{}
	return &Answers{Conf: confPathForTest, Roles: "/r/defaults.yml", Prog: "ws wizard", M: m, P: &effecttest.Prompt{T: t}, Out: out,
		Env: func(string) (string, bool) { return "", false }}, out
}

func TestAnswersLoadRefusesABadLine(t *testing.T) {
	a, out := newAnswers(t, "WS_TARGET=moon\n")
	if _, _, err := a.load(); err == nil || !strings.Contains(err.Error(), "cannot read /h/.config/ws/mac.conf") ||
		len(out.Lines) != 1 {
		t.Errorf("Plan = %v, out %v", err, out.Lines)
	}
}

func TestAnswersRole(t *testing.T) {
	a, _ := newAnswers(t, "")
	if got := a.role("shell_git_email"); got != "ada@example.com" || a.role("shell_other") != "" {
		t.Errorf("role = %q", got)
	}
}
