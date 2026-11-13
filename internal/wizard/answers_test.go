package wizard

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
)

const confPathForTest = "/h/.config/ws/mac.conf"

func newAnswers(t *testing.T, text string) (*Answers, *effecttest.Out) {
	m := effecttest.NewMachine()
	if text != "" {
		m.Files[confPathForTest] = []byte(text)
	}
	out := &effecttest.Out{}
	return &Answers{Conf: confPathForTest, Defaults: []byte("shell_git_name: ada\nshell_git_email: ada@example.com\n"), Prog: "ws client macos wizard", Tty: true, M: m, P: &effecttest.Prompt{T: t}, Out: out,
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
	const all = "WS_TARGET=lan\nWS_LAN=\nWS_REMOTE=\nWS_GIT_NAME=\nWS_TUNNEL_NETWORKS=\nWS_RESOLVERS=\n"
	for name, tc := range map[string]struct{ text, want string }{
		"no file":  {"", "would ask for the values that /h/.config/ws/mac.conf lacks, from WS_TARGET on"},
		"a gap":    {strings.Replace(all, "WS_REMOTE=\n", "", 1), "from WS_REMOTE on"},
		"complete": {all, ""},
		"refused":  {all + "WS_TARGET=cloud\nWS_REMOTE_PORT=22\n", "ws client macos apply would refuse"},
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

// homebrewDone stands in for the Homebrew stage, which Answers waits on.
type homebrewDone struct{}

func (homebrewDone) Name() string                            { return homebrewStage }
func (homebrewDone) WaitsOn() string                         { return "" }
func (homebrewDone) Plan(context.Context) (step.Plan, error) { return step.Plan{}, nil }
func (homebrewDone) Apply(context.Context, step.Plan) error  { return nil }

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
	}
	m := a.M.(*effecttest.Machine)
	m.Results["mkdir -p /h/.config/ws"] = effecttest.Result{}
	changed, err := step.Run(context.Background(), []step.Step{homebrewDone{}, a}, out, false)
	if err != nil || !changed {
		t.Fatalf("Run = %v, %v", changed, err)
	}
	want := "# mine\nWS_USER=ada\nWS_TARGET=lan\nWS_LAN=10.88.0.0/24\nWS_LAN_ADDRESS=10.88.0.10\nWS_REMOTE=\n" +
		"WS_GIT_NAME=ada\nWS_GIT_EMAIL=ada@example.com\nWS_TUNNEL_NETWORKS=\nWS_RESOLVERS=\n"
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
		"More networks through the tunnel, split by blanks (WS_TUNNEL_NETWORKS) [none]": "", "Nameservers behind the tunnel, as DOMAIN=ADDRESS, split by blanks (WS_RESOLVERS) [none]": ""}
	a.M.(*effecttest.Machine).Results["mkdir -p /h/.config/ws"] = effecttest.Result{}
	if _, err := step.Run(context.Background(), []step.Step{homebrewDone{}, a}, out, false); err != nil {
		t.Fatalf("Run = %v", err)
	}
	if got := string(a.M.(*effecttest.Machine).Files[confPathForTest]); !strings.HasPrefix(got, "WS_TARGET=cloud\nWS_REMOTE=r.example\n") {
		t.Errorf("mac.conf = %q", got)
	}
}

func TestAnswersCheckWritesTheScratchFile(t *testing.T) {
	a, out := newAnswers(t, "")
	a.Check, a.Scratch = true, "/lock/mac.conf"
	a.Env = func(k string) (string, bool) {
		return map[string]string{"WS_TARGET": "cloud", "WS_REMOTE": "r.example"}[k], k == "WS_TARGET" || k == "WS_REMOTE"
	}
	m := a.M.(*effecttest.Machine)
	m.Files[confPathForTest] = []byte("WS_USER=ada\n")
	a.P.(*effecttest.Prompt).Answers = map[string]string{
		"The VM's SSH port (WS_REMOTE_PORT) [22]":                                                 "",
		"git's name on the Mac (WS_GIT_NAME) [ada, or none]":                                      "none",
		"More networks through the tunnel, split by blanks (WS_TUNNEL_NETWORKS) [none]":           "",
		"Nameservers behind the tunnel, as DOMAIN=ADDRESS, split by blanks (WS_RESOLVERS) [none]": "",
	}
	if p, err := a.Plan(context.Background()); err != nil || len(p.Changes) != 0 {
		t.Fatalf("Plan = %v, %v", p, err)
	}
	if got := string(m.Files["/lock/mac.conf"]); !strings.HasPrefix(got, "WS_USER=ada\nWS_TARGET=cloud\n") {
		t.Errorf("scratch = %q", got)
	}
	if got := string(m.Files[confPathForTest]); got != "WS_USER=ada\n" {
		t.Errorf("mac.conf changed to %q", got)
	}
	if len(out.Lines) == 0 || !strings.HasPrefix(out.Lines[len(out.Lines)-1], "diff: "+confPathForTest) {
		t.Errorf("out = %q, want the diff of mac.conf last", out.Lines)
	}
}

func TestAnswersCheckPassesALinkThatLacksNoKey(t *testing.T) {
	const all = "WS_TARGET=lan\nWS_LAN=\nWS_REMOTE=\nWS_GIT_NAME=\nWS_TUNNEL_NETWORKS=\nWS_RESOLVERS=\n"
	a, out := newAnswers(t, all)
	a.Check, a.Scratch = true, "/lock/mac.conf"
	m := a.M.(*effecttest.Machine)
	m.Links[confPathForTest], m.Dirs["/h/.config/ws"] = "/dot/mac.conf", 0o755
	if p, err := a.Plan(context.Background()); err != nil || len(p.Changes) != 0 {
		t.Fatalf("Plan = %v, %v", p, err)
	}
	if got := out.Lines; len(got) != 1 || got[0] != "diff: "+confPathForTest+"\n" {
		t.Errorf("out = %q, want an empty diff of mac.conf alone", got)
	}
	if got := string(m.Files["/lock/mac.conf"]); got != all {
		t.Errorf("scratch = %q", got)
	}
}

func TestAnswersCheckRefusesABadEnvironment(t *testing.T) {
	a, _ := newAnswers(t, "")
	a.Check, a.Scratch = true, "/lock/mac.conf"
	a.Env = func(k string) (string, bool) { return "home", k == "WS_TARGET" }
	if _, err := a.Plan(context.Background()); err == nil || !strings.Contains(err.Error(), "WS_TARGET comes from the environment") {
		t.Errorf("Plan = %v", err)
	}
}
