package answers

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/stages/stage"
	"github.com/zadykian/ws/roles/client/macos/values"
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
	return &Answers{Conf: confPathForTest, Server: serverPathForTest, Prog: "ws client macos wizard", Tty: true, M: m, P: &effecttest.Prompt{T: t}, Out: out,
		Env: func(string) (string, bool) { return "", false }}, out
}

func TestAnswersLoadRefusesABadLine(t *testing.T) {
	a, out := newAnswers(t, "", values.Text("WS_TARGET", "moon"))
	if _, _, err := a.load(); err == nil || !strings.Contains(err.Error(), "cannot read "+serverPathForTest) ||
		len(out.Lines) != 1 {
		t.Errorf("Plan = %v, out %v", err, out.Lines)
	}
}

func TestAnswersPlan(t *testing.T) {
	var all = values.Text(
		"WS_TARGET", "lan",
		"WS_LAN", "",
		"WS_REMOTE", "",
		"WS_TUNNEL_NETWORKS", "",
		"WS_RESOLVERS", "",
	)
	for name, tc := range map[string]struct{ mac, server, want string }{
		"no file":  {"", "", "would ask for the values that " + serverPathForTest + " lacks, from WS_TARGET on"},
		"a gap":    {"", strings.Replace(all, values.Text("WS_REMOTE", ""), "", 1), "from WS_REMOTE on"},
		"complete": {"", all, ""},
		"refused":  {"", strings.Replace(all, values.Text("WS_TARGET", "lan"), values.Text("WS_TARGET", "cloud", "WS_REMOTE_PORT", "22"), 1), "ws client macos apply would refuse " + serverPathForTest},
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
	a, out := newAnswers(t, "# mine", values.Text("WS_USER", "ada"))
	a.P.(*effecttest.Prompt).Answers = map[string]string{
		"The machine ws runs on: lan, cloud or mac-vm (WS_TARGET) [lan]":                          "",
		"ws's home LAN (WS_LAN) [10.88.0.0/24, or none]":                                          "",
		"ws's address on the LAN (WS_LAN_ADDRESS) [10.88.0.10]":                                   "",
		"The router's name on the internet (WS_REMOTE) [none]":                                    "none",
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
	var server = values.Text(
		"WS_USER", "ada",
		"WS_SLOT", "0",
		"WS_TUNNEL_PORT", "51820",
		"WS_TARGET", "lan",
		"WS_LAN", "10.88.0.0/24",
		"WS_LAN_ADDRESS", "10.88.0.10",
		"WS_REMOTE", "",
		"WS_TUNNEL_NETWORKS", "",
		"WS_RESOLVERS", "",
	)
	if got := string(m.Files[serverPathForTest]); got != server {
		t.Errorf("server file = %q, want %q", got, server)
	}
	if got := string(m.Files[confPathForTest]); got != "# mine" {
		t.Errorf("mac.conf = %q, want it as it was", got)
	}
}

func TestAnswersFromEnvironment(t *testing.T) {
	a, out := newAnswers(t, "", "")
	a.Env = func(k string) (string, bool) {
		return map[string]string{"WS_TARGET": "cloud", "WS_REMOTE": "r.example"}[k], k == "WS_TARGET" || k == "WS_REMOTE"
	}
	a.P.(*effecttest.Prompt).Answers = map[string]string{"The VM's SSH port (WS_REMOTE_PORT) [22]": "",
		"More networks through the tunnel, split by blanks (WS_TUNNEL_NETWORKS) [none]": "", "Nameservers behind the tunnel, as DOMAIN=ADDRESS, split by blanks (WS_RESOLVERS) [none]": ""}
	a.M.(*effecttest.Machine).Results["mkdir -p /h/.config/ws"] = effecttest.Result{}
	a.M.(*effecttest.Machine).Results["mkdir -p /h/.config/ws/servers/ws"] = effecttest.Result{}
	if _, err := step.Run(context.Background(), []step.Step{homebrewDone{}, a}, out, false); err != nil {
		t.Fatalf("Run = %v", err)
	}
	if got := string(a.M.(*effecttest.Machine).Files[serverPathForTest]); !strings.HasPrefix(got, values.Text(
		"WS_SLOT", "0",
		"WS_TUNNEL_PORT", "51820",
		"WS_TARGET", "cloud",
		"WS_REMOTE", "r.example",
	)) {
		t.Errorf("server file = %q", got)
	}
}
