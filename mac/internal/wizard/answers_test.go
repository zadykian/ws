package wizard

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/mac/internal/effect/effecttest"
	"github.com/zadykian/ws/mac/internal/step"
)

const confPathForTest = "/h/.config/ws/mac.conf"

func newAnswers(t *testing.T, text string) (*Answers, *effecttest.Out) {
	m := effecttest.NewMachine()
	if text != "" {
		m.Files[confPathForTest] = []byte(text)
	}
	m.Files["/r/defaults.yml"] = []byte("shell_git_name: ada\nshell_git_email: ada@example.com\n")
	out := &effecttest.Out{}
	return &Answers{Conf: confPathForTest, Roles: "/r/defaults.yml", Prog: "ws wizard", Tty: true, M: m, P: &effecttest.Prompt{T: t}, Out: out,
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

func TestAnswersAskAndWrite(t *testing.T) {
	a, out := newAnswers(t, "# mine\nWS_USER=ada")
	a.P.(*effecttest.Prompt).Answers = map[string]string{
		"The machine ws runs on: lan, cloud or mac-vm (WS_TARGET) [lan]":                          "",
		"ws's home LAN (WS_LAN) [10.88.0.0/24, or none]":                                          "",
		"ws's address on the LAN (WS_LAN_ADDRESS) [10.88.0.10]":                                   "",
		"The router's name on the internet (WS_REMOTE) [none]":                                    "none",
		"git's name on the Mac (WS_GIT_NAME) [ada, or none]":                                      "",
		"git's email on the Mac (WS_GIT_EMAIL) [ada@example.com]":                                 "",
		"More networks through the tunnel, split by blanks (WS_TUNNEL_NETWORKS) [none]":           "",
		"Nameservers behind the tunnel, as DOMAIN=ADDRESS, split by blanks (WS_RESOLVERS) [none]": "",
		"Another repository whose apply.sh runs on ws, as OWNER/REPO (WS_HOOK) [none]":            "",
	}
	m := a.M.(*effecttest.Machine)
	m.Results["mkdir -p /h/.config/ws"] = effecttest.Result{}
	changed, err := step.Run(context.Background(), []step.Step{a}, out, false)
	if err != nil || !changed {
		t.Fatalf("Run = %v, %v", changed, err)
	}
	want := "# mine\nWS_USER=ada\nWS_TARGET=lan\nWS_LAN=10.88.0.0/24\nWS_LAN_ADDRESS=10.88.0.10\nWS_REMOTE=\n" +
		"WS_GIT_NAME=ada\nWS_GIT_EMAIL=ada@example.com\nWS_TUNNEL_NETWORKS=\nWS_RESOLVERS=\nWS_HOOK=\n"
	if got := string(m.Files[confPathForTest]); got != want {
		t.Errorf("mac.conf = %q, want %q", got, want)
	}
}

func TestAnswersFromEnvironment(t *testing.T) {
	a, out := newAnswers(t, "")
	a.Env = func(k string) (string, bool) {
		return map[string]string{"WS_TARGET": "cloud", "WS_REMOTE": "r.example"}[k], k == "WS_TARGET" || k == "WS_REMOTE"
	}
	a.P.(*effecttest.Prompt).Answers = map[string]string{"The VM's SSH port (WS_REMOTE_PORT) [22]": "", "git's name on the Mac (WS_GIT_NAME) [ada, or none]": "none",
		"More networks through the tunnel, split by blanks (WS_TUNNEL_NETWORKS) [none]": "", "Nameservers behind the tunnel, as DOMAIN=ADDRESS, split by blanks (WS_RESOLVERS) [none]": "",
		"Another repository whose apply.sh runs on ws, as OWNER/REPO (WS_HOOK) [none]": ""}
	a.M.(*effecttest.Machine).Results["mkdir -p /h/.config/ws"] = effecttest.Result{}
	if _, err := step.Run(context.Background(), []step.Step{a}, out, false); err != nil {
		t.Fatalf("Run = %v", err)
	}
	if got := string(a.M.(*effecttest.Machine).Files[confPathForTest]); !strings.HasPrefix(got, "WS_TARGET=cloud\nWS_REMOTE=r.example\n") {
		t.Errorf("mac.conf = %q", got)
	}
}
