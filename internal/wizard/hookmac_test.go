package wizard

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
)

func newHookMac(hook string) (*HookMac, *effecttest.Machine) {
	r, m := newRun("lan", "", false, false)
	m.Files["/h/mac.conf"] = append(m.Files["/h/mac.conf"], []byte(hook)...)
	g := &Gh{LookPath: func(string) (string, error) { return "/bin/gh", nil }, M: m}
	return &HookMac{Host: r.Host, Gh: g, Home: "/h", M: m, Out: r.Host.Out}, m
}

func TestHookMacPlan(t *testing.T) {
	const dir = "/h/repository/tools"
	for name, tc := range map[string]struct {
		conf, url, hooks, want string
		brew                   bool
	}{
		"no hook": {conf: "", want: ""},
		"clone":   {conf: "WS_HOOK=me/tools\n", want: "gh repo clone me/tools " + dir + ", then set"},
		"done":    {conf: "WS_HOOK=me/tools\n", url: "git@github.com:Me/Tools.git", hooks: dir + "/.githooks", brew: true},
		"hooks":   {conf: "WS_HOOK=me/tools\n", url: "https://github.com/me/tools", hooks: "x", want: "core.hooksPath"},
		"brew":    {conf: "WS_HOOK=me/tools\n", url: "https://github.com/me/tools", hooks: dir + "/.githooks", want: "brew bundle"},
		"other":   {conf: "WS_HOOK=me/tools\n", url: "https://github.com/me/other", want: "is not me/tools's clone"},
	} {
		h, m := newHookMac(tc.conf)
		if tc.url != "" {
			m.Dirs[dir], m.Dirs[dir+"/.git"], m.Dirs[dir+"/.githooks"] = 0o755, 0o755, 0o755
			m.Files[dir+"/Brewfile"] = []byte("x")
			m.Results["git -C "+dir+" remote get-url origin"] = effecttest.Result{Out: []byte(tc.url + "\n")}
			m.Results["git -C "+dir+" config core.hooksPath"] = effecttest.Result{Out: []byte(tc.hooks + "\n")}
			bc := effecttest.Result{}
			if !tc.brew {
				bc.Err = context.Canceled
			}
			m.Results["env HOMEBREW_NO_AUTO_UPDATE=1 brew bundle check --no-upgrade --file="+dir+"/Brewfile"] = bc
		}
		p, err := h.Plan(context.Background())
		got := errText(err)
		for _, c := range p.Changes {
			got += c.String()
		}
		if (tc.want == "") != (got == "") || !strings.Contains(got, tc.want) {
			t.Errorf("%s: Plan = %q; want %q", name, got, tc.want)
		}
	}
}

func TestHookMacApply(t *testing.T) {
	const dir = "/h/repository/tools"
	h, m := newHookMac("WS_HOOK=me/tools\n")
	m.Dirs[dir], m.Dirs[dir+"/.git"], m.Dirs[dir+"/.githooks"] = 0o755, 0o755, 0o755
	m.Results["git -C "+dir+" remote get-url origin"] = effecttest.Result{Out: []byte("https://github.com/me/tools\n")}
	m.Results["git -C "+dir+" config core.hooksPath"] = effecttest.Result{}
	set := "git -C " + dir + " config core.hooksPath " + dir + "/.githooks"
	p, err := h.Plan(context.Background())
	if err != nil || len(p.Changes) != 1 {
		t.Fatalf("Plan = %v, %v", p, err)
	}
	if err := h.Apply(context.Background(), p); err == nil || !strings.Contains(err.Error(), "failed, as above") {
		t.Errorf("Apply where git fails = %v", err)
	}
	m.Results[set] = effecttest.Result{}
	if err := h.Apply(context.Background(), p); err != nil {
		t.Errorf("Apply = %v", err)
	}
}

func TestHookMacHand(t *testing.T) {
	for cred, want := range map[string]int{"!/bin/gh auth git-credential": 0, "osxkeychain": 1} {
		h, m := newHookMac("")
		m.Results["git config --global --includes --get-all credential.https://github.com.helper"] =
			effecttest.Result{Out: []byte(cred + "\n")}
		hand := &Hand{}
		h.Hand(context.Background(), hand)
		if len(hand.lines) != want {
			t.Errorf("%s: %d lines in the list; want %d", cred, len(hand.lines), want)
		}
	}
}

func TestHookSplit(t *testing.T) {
	for _, c := range []struct{ in, repo, tag string }{
		{"a/b", "a/b", ""},
		{"a/b@v1.0.0", "a/b", "v1.0.0"},
	} {
		if r, g := hookRepo(c.in), hookTag(c.in); r != c.repo || g != c.tag {
			t.Errorf("split %q = %q, %q, want %q, %q", c.in, r, g, c.repo, c.tag)
		}
	}
}
