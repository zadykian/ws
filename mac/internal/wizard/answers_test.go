package wizard

import (
	"context"
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

func TestAnswersPlan(t *testing.T) {
	const all = "WS_TARGET=lan\nWS_LAN=\nWS_REMOTE=\nWS_GIT_NAME=\nWS_TUNNEL_NETWORKS=\nWS_RESOLVERS=\nWS_HOOK=\n"
	for name, tc := range map[string]struct{ text, want string }{
		"no file":  {"", "would ask for the values that /h/.config/ws/mac.conf lacks, from WS_TARGET on"},
		"a gap":    {strings.Replace(all, "WS_REMOTE=\n", "", 1), "from WS_REMOTE on"},
		"complete": {all, ""},
		"refused":  {all + "WS_TARGET=cloud\nWS_REMOTE_PORT=22\n", "setup.sh would refuse"},
	} {
		a, _ := newAnswers(t, tc.text)
		p, err := a.Plan(context.Background())
		got := ""
		if len(p.Changes) > 0 {
			got = p.Changes[0].String()
		}
		if err != nil || !strings.Contains(got, tc.want) || (tc.want == "" && got != "") {
			t.Errorf("%s: Plan = %q, %v; want %q", name, got, err, tc.want)
		}
	}
}
