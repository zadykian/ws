package setup_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/roles/client/macos/conf"
	"github.com/zadykian/ws/roles/client/macos/internal/change"
	"github.com/zadykian/ws/roles/client/macos/internal/mactest"
	"github.com/zadykian/ws/roles/client/macos/setup"
	"github.com/zadykian/ws/roles/internal/sshcfg"
)

const (
	gitHome = "/Users/me"
	gitPub  = gitHome + "/.ssh/git.pub"
	gitKey  = "ssh-ed25519 AAAAC3Nz me@mac\n"
	gitList = "/opt/bin/gh api --paginate user/ssh_signing_keys --jq .[].key"
)

func gitStep(m *effecttest.Machine, look func(string) (string, error)) *setup.Git {
	m.Results["ssh-keygen -l -f "+gitPub] = effecttest.Result{}
	m.Results["hostname -s"] = effecttest.Result{Out: []byte("mac\n")}
	m.Results[gitList] = effecttest.Result{Out: []byte("ssh-ed25519 AAAAC3Nz\n")}
	if look == nil {
		look = func(name string) (string, error) { return "/opt/bin/" + name, nil }
	}
	return &setup.Git{
		Conf: conf.Conf{GitName: "A B", GitEmail: "a@b.c", GitSigningPubKey: gitPub},
		Home: gitHome, M: m, Out: &effecttest.Out{}, LookPath: look,
	}
}

// prereqs returns the changes that install gh, which the files wait on.
func prereqs(p step.Plan) []step.Change {
	var cs []step.Change
	for _, c := range p.Changes {
		if run, ok := c.(change.Run); ok {
			cs = append(cs, run)
		}
	}
	return cs
}

func TestGitWithoutNameLeavesGitAlone(t *testing.T) {
	p, err := (&setup.Git{M: effecttest.NewMachine(), Out: &effecttest.Out{}}).Plan(context.Background())
	if err != nil || len(p.Changes) != 0 {
		t.Errorf("Plan = %v, %v; want nothing", p.Changes, err)
	}
}

func TestGitPlansGh(t *testing.T) {
	missing := func(n string) (string, error) {
		if n == "gh" {
			return "", errors.New("not found")
		}
		return "/opt/bin/brew", nil
	}
	for name, tc := range map[string]struct {
		look func(string) (string, error)
		want int
		err  string
	}{
		"gh there":    {nil, 0, ""},
		"no gh":       {missing, 1, ""},
		"no brew":     {func(string) (string, error) { return "", errors.New("none") }, 0, "install Homebrew first"},
		"relative gh": {func(string) (string, error) { return "bin/gh", nil }, 0, "drop PATH's relative directories"},
	} {
		m := effecttest.NewMachine()
		m.Results["brew --prefix"] = effecttest.Result{Out: []byte("/opt\n")}
		m.Files[gitPub] = []byte(gitKey)
		p, err := gitStep(m, tc.look).Plan(context.Background())
		if (err == nil) != (tc.err == "") || (err != nil && !strings.Contains(err.Error(), tc.err)) {
			t.Errorf("%s: err = %v, want %q", name, err, tc.err)
		} else if len(prereqs(p)) != tc.want {
			t.Errorf("%s: changes = %v, want %d", name, p.Changes, tc.want)
		}
	}
}

func TestGitApplyInstallsGh(t *testing.T) {
	m, _ := gitReady()
	m.Results["brew install gh"] = effecttest.Result{}
	install := change.Run{Name: "brew", Args: []string{"install", "gh"}}
	err := gitStep(m, nil).Apply(context.Background(), step.Plan{Changes: []step.Change{install}})
	if err != nil || m.Calls[0] != "run brew install gh" {
		t.Errorf("Apply = %v, calls %q; want brew install gh first", err, m.Calls)
	}
}

const (
	includeText   = "[include]\n    path = ~/.config/ws/gitconfig\n"
	gitConfigFile = gitHome + "/.gitconfig"
)

// gitConfig has ~/.gitconfig hold text, and git config answer with the include paths and the names
// of the settings in it.
func gitConfig(m *effecttest.Machine, text, paths, names string) {
	m.Files[gitConfigFile] = []byte(text)
	const cmd = "git config --file " + gitConfigFile
	m.Results[cmd+" --type path --get-all include.path"] = effecttest.Result{Out: []byte(paths)}
	m.Results[cmd+" --name-only --list"] = effecttest.Result{Out: []byte(names)}
}

// gitReady is a Mac with the key and gh there.
func gitReady() (*effecttest.Machine, *setup.Git) {
	m := effecttest.NewMachine()
	m.Files[gitPub] = []byte(gitKey)
	m.Dirs[gitHome+"/.config"], m.Dirs[gitHome+"/.config/ws"] = 0o755, 0o700
	gitConfig(m, includeText, gitHome+"/.config/ws/gitconfig\n", "include.path\n")
	return m, gitStep(m, nil)
}

func TestGitWritesAllowedSignersAndGitconfig(t *testing.T) {
	m, g := gitReady()
	p, err := g.Plan(context.Background())
	if err != nil || len(p.Changes) != 3 {
		t.Fatalf("Plan = %v, %v; want two writes and the line", p.Changes, err)
	}
	if err := g.Apply(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	dir := gitHome + "/.config/ws/"
	if got := string(m.Files[dir+"allowed_signers"]); !strings.HasSuffix(got, "\na@b.c ssh-ed25519 AAAAC3Nz\n") {
		t.Errorf("allowed_signers = %q", got)
	}
	if got := string(m.Files[dir+"gitconfig"]); !strings.Contains(got, "signingkey = \""+gitPub+"\"") ||
		!strings.Contains(got, "helper = !/opt/bin/gh auth git-credential") {
		t.Errorf("gitconfig = %q", got)
	}
	if again, err := g.Plan(context.Background()); err != nil || len(again.Changes) != 0 {
		t.Errorf("Plan after Apply = %v, %v; want nothing", again.Changes, err)
	}
}

func TestGitFixesAModeAndMakesDirectories(t *testing.T) {
	m, g := gitReady()
	delete(m.Dirs, gitHome+"/.config/ws")
	p, err := g.Plan(context.Background())
	if err != nil || len(p.Changes) != 4 || p.Changes[0].String() != "would make "+gitHome+"/.config/ws, mode 700" {
		t.Fatalf("Plan = %v, %v; want the directory and two writes", p.Changes, err)
	}
	m, g = gitReady()
	if err := g.Apply(context.Background(), mustPlan(t, g)); err != nil {
		t.Fatal(err)
	}
	m.Modes[gitHome+"/.config/ws/gitconfig"] = 0o600
	if p, err := g.Plan(context.Background()); err != nil || len(p.Changes) != 2 {
		t.Errorf("Plan with a wrong mode = %v, %v; want one change", p.Changes, err)
	}
}

func TestGitTrustsSecretivesKeyThroughItsScript(t *testing.T) {
	m, g := gitReady()
	pub := gitHome + "/" + sshcfg.SecretiveData + "/PublicKeys/a.pub"
	g.Conf.GitSigningPubKey = pub
	m.Files[pub] = []byte(gitKey)
	m.Results["ssh-keygen -l -f "+pub] = effecttest.Result{}
	p, err := g.Plan(context.Background())
	if err != nil || len(p.Changes) != 4 || !strings.Contains(p.Changes[0].String(), "secretive-ssh-keygen") {
		t.Errorf("Plan = %v, %v; want the script, allowed signers and gitconfig", p.Changes, err)
	}
}

func mustPlan(t *testing.T, g *setup.Git) step.Plan {
	p, err := g.Plan(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func gitWithConfig(text, paths, names string) (*effecttest.Machine, *effecttest.Out, *setup.Git) {
	m, g := gitReady()
	delete(m.Files, gitConfigFile)
	if text != "" {
		gitConfig(m, text, paths, names)
	}
	out := &effecttest.Out{}
	g.Out = out
	g.Now = func() time.Time { return time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC) }
	return m, out, g
}

func TestGitStopsAtAnIncludeBelowSettings(t *testing.T) {
	_, _, g := gitWithConfig("x", gitHome+"/.config/ws/gitconfig\n", "user.name\ninclude.path\n")
	if _, err := g.Plan(context.Background()); err == nil || !strings.Contains(err.Error(), "below its own settings") {
		t.Errorf("err = %v; want an include below the file's settings", err)
	}
}

func TestGitStopsAtALinkAndAtGitConfigFailing(t *testing.T) {
	m, _, g := gitWithConfig("x", "", "user.name\n")
	m.Links[gitConfigFile] = "/dotfiles/gitconfig"
	m.Files["/dotfiles/gitconfig"] = []byte("x")
	if _, err := g.Plan(context.Background()); err == nil || !strings.Contains(err.Error(), "is a link to /dotfiles/gitconfig") {
		t.Errorf("a link: err = %v", err)
	}
	m.Results["git config --file "+gitConfigFile+" --name-only --list"] = effecttest.Result{Err: errors.New("bad")}
	if _, err := g.Plan(context.Background()); err == nil || !strings.Contains(err.Error(), "git config cannot read") {
		t.Errorf("git config failing: err = %v", err)
	}
}

func TestGitSaysToOpenSecretiveForADeniedPub(t *testing.T) {
	m := mactest.AgentMachine(effecttest.NewMachine())
	g := gitStep(m, nil)
	in := mactest.AgentData + "/PublicKeys/a.pub"
	m.Root[in] = m.Root[mactest.AgentSock]
	g.Home, g.Conf.GitSigningPubKey = mactest.AgentHome, in
	if _, err := g.Plan(context.Background()); err == nil || !strings.Contains(err.Error(), "open Secretive") {
		t.Errorf("err = %v, want the word to open Secretive", err)
	}
}

func TestGitRefusesAnAgentKeyOfATypeWsLacks(t *testing.T) {
	const line = "ssh-mldsa-65 AAAA me\n"
	m := mactest.AgentMachine(effecttest.NewMachine(), line)
	g := gitStep(m, nil)
	g.Home, g.Conf.GitSigningPubKey = mactest.AgentHome, mactest.CopyOf(line)
	if _, err := g.Plan(context.Background()); err == nil || !strings.Contains(err.Error(), "which ws's OpenSSH lacks") {
		t.Errorf("err = %v, want a type that ws's OpenSSH lacks", err)
	}
}

func TestGitSetsTheModeOfWsDirectoryAndWarnsOfALinkedOne(t *testing.T) {
	m, g := gitReady()
	m.Dirs[gitHome+"/.config/ws"] = 0o755
	p, err := g.Plan(context.Background())
	if err != nil || len(p.Changes) != 4 || p.Changes[0] != (change.SetMode{Path: gitHome + "/.config/ws", Mode: 0o700}) {
		t.Fatalf("Plan = %v, %v; want the mode and two writes", p.Changes, err)
	}
	m, g = gitReady()
	out := g.Out.(*effecttest.Out)
	m.Dirs["/dots"], m.Links[gitHome+"/.config"] = 0o755, "/dots"
	if _, err := g.Plan(context.Background()); err != nil || !strings.Contains(strings.Join(out.Lines, "\n"),
		gitHome+"/.config is a link to /dots: ws's files go there") {
		t.Errorf("Plan = %v, out %v; want the warning", err, out.Lines)
	}
}
