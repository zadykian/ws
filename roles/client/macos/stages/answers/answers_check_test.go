package answers

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/values"
)

func TestAnswersCheckWritesTheScratchFile(t *testing.T) {
	a, out := newAnswers(t, "", "")
	a.Check, a.Scratch, a.ServerScratch = true, "/lock/values.yml", "/lock/servers/ws/values.yml"
	a.Env = func(k string) (string, bool) {
		return map[string]string{"WS_TARGET": "cloud", "WS_REMOTE": "r.example"}[k], k == "WS_TARGET" || k == "WS_REMOTE"
	}
	m := a.M.(*effecttest.Machine)
	m.Files[serverPathForTest] = []byte(values.Text("WS_USER", "ada"))
	m.Results["mkdir -p /lock"] = effecttest.Result{}
	m.Results["mkdir -p /lock/servers/ws"] = effecttest.Result{}
	a.P.(*effecttest.Prompt).Answers = map[string]string{
		"The VM's SSH port (WS_REMOTE_PORT) [22]":                                                 "",
		"More networks through the tunnel, split by blanks (WS_TUNNEL_NETWORKS) [none]":           "",
		"Nameservers behind the tunnel, as DOMAIN=ADDRESS, split by blanks (WS_RESOLVERS) [none]": "",
	}
	if p, err := a.Plan(context.Background()); err != nil || len(p.Changes) != 0 {
		t.Fatalf("Plan = %v, %v", p, err)
	}
	if got := string(m.Files["/lock/servers/ws/values.yml"]); !strings.HasPrefix(got, values.Text(
		"WS_USER", "ada",
		"WS_SLOT", "0",
		"WS_TUNNEL_PORT", "51820",
		"WS_TARGET", "cloud",
	)) {
		t.Errorf("server scratch = %q", got)
	}
	if got := string(m.Files[serverPathForTest]); got != values.Text("WS_USER", "ada") {
		t.Errorf("server file changed to %q", got)
	}
	if len(out.Lines) == 0 || !strings.HasPrefix(out.Lines[len(out.Lines)-1], "diff: "+serverPathForTest) {
		t.Errorf("out = %q, want the diff of the server's file last", out.Lines)
	}
}

// TestAnswersLeavesAFileThatGainsNothing asks for the server's keys alone, so values.yml, a link
// with no final newline, is neither written nor said.
func TestAnswersLeavesAFileThatGainsNothing(t *testing.T) {
	var mac = values.Text("WS_GIT_SIGNING_PUB_KEY", "~/k.pub")
	for _, check := range []bool{false, true} {
		a, out := newAnswers(t, mac, values.Text("WS_USER", "ada"))
		a.Check, a.Scratch, a.ServerScratch = check, "/lock/values.yml", "/lock/servers/ws/values.yml"
		a.Env = func(k string) (string, bool) {
			return map[string]string{"WS_TARGET": "cloud", "WS_REMOTE": "r.example"}[k], k == "WS_TARGET" || k == "WS_REMOTE"
		}
		m := a.M.(*effecttest.Machine)
		m.Links[confPathForTest] = "/dot/values.yml"
		m.Results["mkdir -p /h/.config/ws/servers/ws"] = effecttest.Result{}
		m.Results["mkdir -p /lock"] = effecttest.Result{}
		m.Results["mkdir -p /lock/servers/ws"] = effecttest.Result{}
		a.P.(*effecttest.Prompt).Answers = map[string]string{
			"The VM's SSH port (WS_REMOTE_PORT) [22]":                                                 "",
			"More networks through the tunnel, split by blanks (WS_TUNNEL_NETWORKS) [none]":           "",
			"Nameservers behind the tunnel, as DOMAIN=ADDRESS, split by blanks (WS_RESOLVERS) [none]": "",
		}
		if _, err := step.Run(context.Background(), []step.Step{homebrewDone{}, a}, out, check); err != nil {
			t.Fatalf("check=%v: Run = %v", check, err)
		}
		if got := string(m.Files[confPathForTest]); got != mac {
			t.Errorf("check=%v: values.yml = %q, want it as it was", check, got)
		}
		for _, l := range out.Lines {
			if strings.Contains(l, confPathForTest) && (strings.Contains(l, "wrote") || strings.Contains(l, "diff")) {
				t.Errorf("check=%v: out has %q, want nothing of values.yml", check, l)
			}
		}
	}
}

func TestAnswersCheckPassesALinkThatLacksNoKey(t *testing.T) {
	var all = values.Text(
		"WS_TARGET", "lan",
		"WS_LAN", "",
		"WS_REMOTE", "",
		"WS_TUNNEL_NETWORKS", "",
		"WS_RESOLVERS", "",
	)
	a, out := newAnswers(t, "", all)
	a.Check, a.Scratch, a.ServerScratch = true, "/lock/values.yml", "/lock/servers/ws/values.yml"
	m := a.M.(*effecttest.Machine)
	m.Links[serverPathForTest], m.Dirs["/h/.config/ws"], m.Dirs["/h/.config/ws/servers/ws"] = "/dot/values.yml", 0o755, 0o755
	m.Results["mkdir -p /lock"] = effecttest.Result{}
	m.Results["mkdir -p /lock/servers/ws"] = effecttest.Result{}
	if p, err := a.Plan(context.Background()); err != nil || len(p.Changes) != 0 {
		t.Fatalf("Plan = %v, %v", p, err)
	}
	if got := out.Lines; len(got) != 0 {
		t.Errorf("out = %q, want no diff of a file that gains nothing", got)
	}
	if got := string(m.Files["/lock/servers/ws/values.yml"]); got != all {
		t.Errorf("scratch = %q", got)
	}
}

// TestAnswersCheckCopiesTheOtherServersFiles keeps lab's file as it stands, so that the stages
// after see it beside ws's rehearsal as the run would.
func TestAnswersCheckCopiesTheOtherServersFiles(t *testing.T) {
	var all = values.Text(
		"WS_TARGET", "lan",
		"WS_LAN", "",
		"WS_REMOTE", "",
		"WS_TUNNEL_NETWORKS", "",
		"WS_RESOLVERS", "",
	)
	var lab = values.Text("WS_USER", "ada", "WS_SLOT", "5")
	a, _ := newAnswers(t, "", all)
	a.Check, a.Scratch, a.ServerScratch = true, "/lock/values.yml", "/lock/servers/ws/values.yml"
	m := a.M.(*effecttest.Machine)
	m.Dirs["/h/.config/ws/servers"], m.Dirs["/h/.config/ws/servers/ws"], m.Dirs["/h/.config/ws/servers/lab"] = 0o755, 0o755, 0o755
	m.Files["/h/.config/ws/servers/lab/values.yml"] = []byte(lab)
	for _, dir := range []string{"/lock", "/lock/servers/ws", "/lock/servers/lab"} {
		m.Results["mkdir -p "+dir] = effecttest.Result{}
	}
	if _, err := a.Plan(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := string(m.Files["/lock/servers/lab/values.yml"]); got != lab {
		t.Errorf("lab's scratch = %q, want its file as it stands", got)
	}
	if got := string(m.Files["/lock/servers/ws/values.yml"]); got != all {
		t.Errorf("ws's scratch = %q, want the rehearsal", got)
	}
}

// TestAnswersCheckRefusesAnUnreadableSiblingAsTheRunDoes words the refusal as Setup's: the server,
// the cause and the way out.
func TestAnswersCheckRefusesAnUnreadableSiblingAsTheRunDoes(t *testing.T) {
	var all = values.Text(
		"WS_TARGET", "lan",
		"WS_LAN", "",
		"WS_REMOTE", "",
		"WS_TUNNEL_NETWORKS", "",
		"WS_RESOLVERS", "",
	)
	a, _ := newAnswers(t, "", all)
	a.Check, a.Scratch, a.ServerScratch = true, "/lock/values.yml", "/lock/servers/ws/values.yml"
	m := a.M.(*effecttest.Machine)
	lab := "/h/.config/ws/servers/lab/values.yml"
	m.Dirs["/h/.config/ws/servers"], m.Dirs["/h/.config/ws/servers/ws"], m.Dirs["/h/.config/ws/servers/lab"] = 0o755, 0o755, 0o755
	m.Files[lab], m.Root[lab] = []byte(values.Text("WS_USER", "ada")), effect.Attrs{}
	for _, dir := range []string{"/lock", "/lock/servers/ws"} {
		m.Results["mkdir -p "+dir] = effecttest.Result{}
	}
	want := "server lab: open " + lab + ": permission denied: fix its file, or ws client macos remove lab"
	if _, err := a.Plan(context.Background()); err == nil || err.Error() != want {
		t.Errorf("Plan = %v, want %q", err, want)
	}
}

func TestAnswersCheckRefusesABadEnvironment(t *testing.T) {
	a, _ := newAnswers(t, "", "")
	a.Check, a.Scratch = true, "/lock/values.yml"
	a.Env = func(k string) (string, bool) { return "home", k == "WS_TARGET" }
	if _, err := a.Plan(context.Background()); err == nil || !strings.Contains(err.Error(), "WS_TARGET comes from the environment") {
		t.Errorf("Plan = %v", err)
	}
}

func TestAnswersPlaceAServerFileThatLacksItsSlot(t *testing.T) {
	var box = values.Text(
		"WS_SLOT", "0",
		"WS_TARGET", "lan",
		"WS_REMOTE", "r.example",
		"WS_REMOTE_PORT", "22022",
	)
	for name, tc := range map[string]struct{ server, other, remote, want string }{
		"new file":  {"", "", "none", values.Text("WS_SLOT", "0", "WS_TUNNEL_PORT", "51820", "WS_TARGET", "lan")},
		"slot kept": {values.Text("WS_SLOT", "3"), "", "none", values.Text("WS_SLOT", "3", "WS_TARGET", "lan")},
		"file lacking": {values.Text("WS_USER", "ada"), "", "none", values.Text(
			"WS_USER", "ada",
			"WS_SLOT", "0",
			"WS_TUNNEL_PORT", "51820",
			"WS_TARGET", "lan",
		)},
		"port kept":  {values.Text("WS_TUNNEL_PORT", "51830"), "", "none", values.Text("WS_TUNNEL_PORT", "51830", "WS_SLOT", "0", "WS_TARGET", "lan")},
		"beside one": {"", box, "none", values.Text("WS_SLOT", "1", "WS_TUNNEL_PORT", "51820", "WS_TARGET", "lan")},
		"same router": {"", box, "r.example", values.Text(
			"WS_SLOT", "1",
			"WS_TUNNEL_PORT", "51821",
			"WS_TARGET", "lan",
			"WS_LAN", "",
			"WS_REMOTE", "r.example",
			"WS_REMOTE_PORT", "22023",
		)},
	} {
		a, out := newAnswers(t, values.Text("WS_GIT_SIGNING_PUB_KEY", "~/k.pub"), tc.server)
		m := a.M.(*effecttest.Machine)
		if tc.other != "" {
			m.Dirs["/h/.config/ws/servers"] = 0o755
			m.Dirs["/h/.config/ws/servers/box"] = 0o755
			m.Files["/h/.config/ws/servers/box/values.yml"] = []byte(tc.other)
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
		if got := string(m.Files[confPathForTest]); strings.Contains(got, "WS_SLOT") || strings.Contains(got, "WS_TUNNEL_PORT") {
			t.Errorf("%s: values.yml = %q, want no slot or port", name, got)
		}
	}
}
