package answers

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/stages/stage"
)

const (
	confPathForTest   = "/h/.config/ws/mac.conf"
	serverPathForTest = "/h/.config/ws/servers/ws/server.conf"
)

// newAnswers returns the stage over a Mac whose mac.conf holds mac, and a server's file holds
// server; a text left empty is a file that is missing.
func newAnswers(t *testing.T, mac, server string) (*Answers, *effecttest.Out) {
	m := effecttest.NewMachine()
	if mac != "" {
		m.Files[confPathForTest] = []byte(mac)
	}
	if server != "" {
		m.Files[serverPathForTest] = []byte(server)
	}
	out := &effecttest.Out{}
	return &Answers{Conf: confPathForTest, Server: serverPathForTest, Defaults: []byte("shell_git_name: ada\nshell_git_email: ada@example.com\n"), Prog: "ws client macos wizard", Tty: true, M: m, P: &effecttest.Prompt{T: t}, Out: out,
		Env: func(string) (string, bool) { return "", false }}, out
}

func TestAnswersLoadRefusesABadLine(t *testing.T) {
	a, out := newAnswers(t, "", "WS_TARGET=moon\n")
	if _, _, err := a.load(); err == nil || !strings.Contains(err.Error(), "cannot read "+serverPathForTest) ||
		len(out.Lines) != 1 {
		t.Errorf("Plan = %v, out %v", err, out.Lines)
	}
}

func TestAnswersPlan(t *testing.T) {
	const all = "WS_TARGET=lan\nWS_LAN=\nWS_REMOTE=\nWS_TUNNEL_NETWORKS=\nWS_RESOLVERS=\n"
	const mac = "WS_GIT_NAME=\nWS_GIT_EMAIL=\n"
	for name, tc := range map[string]struct{ mac, server, want string }{
		"no file":   {"", "", "would ask for the values that " + serverPathForTest + " lacks, from WS_TARGET on"},
		"a gap":     {mac, strings.Replace(all, "WS_REMOTE=\n", "", 1), "from WS_REMOTE on"},
		"the Mac's": {"", all, "would ask for the values that " + confPathForTest + " lacks, from WS_GIT_NAME on"},
		"complete":  {mac, all, ""},
		"refused":   {mac, all + "WS_TARGET=cloud\nWS_REMOTE_PORT=22\n", "ws client macos apply would refuse " + serverPathForTest},
	} {
		a, _ := newAnswers(t, tc.mac, tc.server)
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

func (homebrewDone) Name() string                            { return stage.Homebrew }
func (homebrewDone) WaitsOn() string                         { return "" }
func (homebrewDone) Plan(context.Context) (step.Plan, error) { return step.Plan{}, nil }
func (homebrewDone) Apply(context.Context, step.Plan) error  { return nil }

func TestAnswersAskAndWrite(t *testing.T) {
	a, out := newAnswers(t, "# mine", "WS_USER=ada")
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
	m.Results["mkdir -p /h/.config/ws/servers/ws"] = effecttest.Result{}
	changed, err := step.Run(context.Background(), []step.Step{homebrewDone{}, a}, out, false)
	if err != nil || !changed {
		t.Fatalf("Run = %v, %v", changed, err)
	}
	const server = "WS_USER=ada\nWS_SLOT=0\nWS_PORT=51820\nWS_TARGET=lan\nWS_LAN=10.88.0.0/24\nWS_LAN_ADDRESS=10.88.0.10\nWS_REMOTE=\n" +
		"WS_TUNNEL_NETWORKS=\nWS_RESOLVERS=\n"
	if got := string(m.Files[serverPathForTest]); got != server {
		t.Errorf("server file = %q, want %q", got, server)
	}
	if got, want := string(m.Files[confPathForTest]), "# mine\nWS_GIT_NAME=ada\nWS_GIT_EMAIL=ada@example.com\n"; got != want {
		t.Errorf("mac.conf = %q, want %q", got, want)
	}
}

func TestAnswersFromEnvironment(t *testing.T) {
	a, out := newAnswers(t, "", "")
	a.Env = func(k string) (string, bool) {
		return map[string]string{"WS_TARGET": "cloud", "WS_REMOTE": "r.example"}[k], k == "WS_TARGET" || k == "WS_REMOTE"
	}
	a.P.(*effecttest.Prompt).Answers = map[string]string{"The VM's SSH port (WS_REMOTE_PORT) [22]": "", "git's name on the Mac (WS_GIT_NAME) [ada, or none]": "none",
		"More networks through the tunnel, split by blanks (WS_TUNNEL_NETWORKS) [none]": "", "Nameservers behind the tunnel, as DOMAIN=ADDRESS, split by blanks (WS_RESOLVERS) [none]": ""}
	a.M.(*effecttest.Machine).Results["mkdir -p /h/.config/ws"] = effecttest.Result{}
	a.M.(*effecttest.Machine).Results["mkdir -p /h/.config/ws/servers/ws"] = effecttest.Result{}
	if _, err := step.Run(context.Background(), []step.Step{homebrewDone{}, a}, out, false); err != nil {
		t.Fatalf("Run = %v", err)
	}
	if got := string(a.M.(*effecttest.Machine).Files[serverPathForTest]); !strings.HasPrefix(got, "WS_SLOT=0\nWS_PORT=51820\nWS_TARGET=cloud\nWS_REMOTE=r.example\n") {
		t.Errorf("server file = %q", got)
	}
}

func TestAnswersCheckWritesTheScratchFile(t *testing.T) {
	a, out := newAnswers(t, "", "")
	a.Check, a.Scratch, a.ServerScratch = true, "/lock/mac.conf", "/lock/servers/ws/server.conf"
	a.Env = func(k string) (string, bool) {
		return map[string]string{"WS_TARGET": "cloud", "WS_REMOTE": "r.example"}[k], k == "WS_TARGET" || k == "WS_REMOTE"
	}
	m := a.M.(*effecttest.Machine)
	m.Files[serverPathForTest] = []byte("WS_USER=ada\n")
	m.Results["mkdir -p /lock"] = effecttest.Result{}
	m.Results["mkdir -p /lock/servers/ws"] = effecttest.Result{}
	a.P.(*effecttest.Prompt).Answers = map[string]string{
		"The VM's SSH port (WS_REMOTE_PORT) [22]":                                                 "",
		"git's name on the Mac (WS_GIT_NAME) [ada, or none]":                                      "none",
		"More networks through the tunnel, split by blanks (WS_TUNNEL_NETWORKS) [none]":           "",
		"Nameservers behind the tunnel, as DOMAIN=ADDRESS, split by blanks (WS_RESOLVERS) [none]": "",
	}
	if p, err := a.Plan(context.Background()); err != nil || len(p.Changes) != 0 {
		t.Fatalf("Plan = %v, %v", p, err)
	}
	if got := string(m.Files["/lock/servers/ws/server.conf"]); !strings.HasPrefix(got, "WS_USER=ada\nWS_SLOT=0\nWS_PORT=51820\nWS_TARGET=cloud\n") {
		t.Errorf("server scratch = %q", got)
	}
	if got := string(m.Files["/lock/mac.conf"]); got != "WS_GIT_NAME=\n" {
		t.Errorf("mac scratch = %q, want the answer for git's name alone", got)
	}
	if got := string(m.Files[serverPathForTest]); got != "WS_USER=ada\n" {
		t.Errorf("server file changed to %q", got)
	}
	if len(out.Lines) == 0 || !strings.HasPrefix(out.Lines[len(out.Lines)-1], "diff: "+serverPathForTest) {
		t.Errorf("out = %q, want the diff of the server's file last", out.Lines)
	}
}

func TestAnswersCheckPassesALinkThatLacksNoKey(t *testing.T) {
	const all = "WS_TARGET=lan\nWS_LAN=\nWS_REMOTE=\nWS_TUNNEL_NETWORKS=\nWS_RESOLVERS=\n"
	a, out := newAnswers(t, "WS_GIT_NAME=\nWS_GIT_EMAIL=\n", all)
	a.Check, a.Scratch, a.ServerScratch = true, "/lock/mac.conf", "/lock/servers/ws/server.conf"
	m := a.M.(*effecttest.Machine)
	m.Links[serverPathForTest], m.Dirs["/h/.config/ws"], m.Dirs["/h/.config/ws/servers/ws"] = "/dot/server.conf", 0o755, 0o755
	m.Results["mkdir -p /lock"] = effecttest.Result{}
	m.Results["mkdir -p /lock/servers/ws"] = effecttest.Result{}
	if p, err := a.Plan(context.Background()); err != nil || len(p.Changes) != 0 {
		t.Fatalf("Plan = %v, %v", p, err)
	}
	if got := out.Lines; len(got) != 2 || got[0] != "diff: "+confPathForTest+"\n" || got[1] != "diff: "+serverPathForTest+"\n" {
		t.Errorf("out = %q, want an empty diff of each file", got)
	}
	if got := string(m.Files["/lock/servers/ws/server.conf"]); got != all {
		t.Errorf("scratch = %q", got)
	}
}

func TestAnswersCheckRefusesABadEnvironment(t *testing.T) {
	a, _ := newAnswers(t, "", "")
	a.Check, a.Scratch = true, "/lock/mac.conf"
	a.Env = func(k string) (string, bool) { return "home", k == "WS_TARGET" }
	if _, err := a.Plan(context.Background()); err == nil || !strings.Contains(err.Error(), "WS_TARGET comes from the environment") {
		t.Errorf("Plan = %v", err)
	}
}

func TestAnswersPlaceAServerFileThatLacksItsSlot(t *testing.T) {
	const box = "WS_SLOT=0\nWS_TARGET=lan\nWS_REMOTE=r.example\nWS_REMOTE_PORT=22022\n"
	for name, tc := range map[string]struct{ server, other, remote, want string }{
		"new file":     {"", "", "none", "WS_SLOT=0\nWS_PORT=51820\nWS_TARGET=lan\n"},
		"slot kept":    {"WS_SLOT=3\n", "", "none", "WS_SLOT=3\nWS_TARGET=lan\n"},
		"file lacking": {"WS_USER=ada\n", "", "none", "WS_USER=ada\nWS_SLOT=0\nWS_PORT=51820\nWS_TARGET=lan\n"},
		"port kept":    {"WS_PORT=51830\n", "", "none", "WS_PORT=51830\nWS_SLOT=0\nWS_TARGET=lan\n"},
		"beside one":   {"", box, "none", "WS_SLOT=1\nWS_PORT=51820\nWS_TARGET=lan\n"},
		"same router":  {"", box, "r.example", "WS_SLOT=1\nWS_PORT=51821\nWS_TARGET=lan\nWS_LAN=\nWS_REMOTE=r.example\nWS_REMOTE_PORT=22023\n"},
	} {
		a, out := newAnswers(t, "WS_GIT_NAME=ada\nWS_GIT_EMAIL=a@b.c\n", tc.server)
		m := a.M.(*effecttest.Machine)
		if tc.other != "" {
			m.Dirs["/h/.config/ws/servers"] = 0o755
			m.Dirs["/h/.config/ws/servers/box"] = 0o755
			m.Files["/h/.config/ws/servers/box/server.conf"] = []byte(tc.other)
		}
		a.P.(*effecttest.Prompt).Answers = map[string]string{
			"The machine ws runs on: lan, cloud or mac-vm (WS_TARGET) [lan]":                          "",
			"ws's home LAN (WS_LAN) [10.88.0.0/24, or none]":                                          "none",
			"The router's name on the internet (WS_REMOTE) [none]":                                    tc.remote,
			"The router's port forwarded to ws's 22 (WS_REMOTE_PORT) [22023, or none]":                "",
			"More networks through the tunnel, split by blanks (WS_TUNNEL_NETWORKS) [none]":           "",
			"Nameservers behind the tunnel, as DOMAIN=ADDRESS, split by blanks (WS_RESOLVERS) [none]": "",
		}
		m.Results["mkdir -p /h/.config/ws"] = effecttest.Result{}
		m.Results["mkdir -p /h/.config/ws/servers/ws"] = effecttest.Result{}
		if _, err := step.Run(context.Background(), []step.Step{homebrewDone{}, a}, out, false); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if got := string(m.Files[serverPathForTest]); !strings.HasPrefix(got, tc.want) {
			t.Errorf("%s: server file = %q, want it to start with %q", name, got, tc.want)
		}
		if got := string(m.Files[confPathForTest]); strings.Contains(got, "WS_SLOT") || strings.Contains(got, "WS_PORT") {
			t.Errorf("%s: mac.conf = %q, want no slot or port", name, got)
		}
	}
}
