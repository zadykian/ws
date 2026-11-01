package wizard

import (
	"context"
	"strings"
	"testing"

	"github.com/zadykian/ws/effect/effecttest"
	"github.com/zadykian/ws/mac/internal/step"
)

const hashOf = "git -C / hash-object --stdin"

func newWords(login effecttest.Result, remote string) (*Words, *effecttest.Machine) {
	r, m := newRoot(login)
	m.Results["ssh -G ws-ssh"] = effecttest.Result{Out: []byte("user root\nhostname 10.1.1.2\nport 22\n" +
		"userknownhostsfile ~/.ssh/known_hosts\n")}
	files := &Remote{SSH: r.Host.SSH, Script: []byte("f")}
	m.Results[remoteLine(files, "words")] = effecttest.Result{Out: []byte(remote)}
	m.Results[hashOf] = effecttest.Result{Out: []byte("aaa\n")}
	return &Words{Host: r.Host, Root: r, Files: files, Home: "/h", M: m, Out: r.Host.Out}, m
}

func TestWordsPlan(t *testing.T) {
	ok := effecttest.Result{Out: []byte("ok\n")}
	for name, tc := range map[string]struct {
		login       effecttest.Result
		remote, mac string
		link        string
		want        string
	}{
		"no list on ws":   {ok, "-\n", "", "", ""},
		"same":            {ok, "aaa\n", "x", "", ""},
		"differs":         {ok, "bbb\n", "x", "", "would copy ws's word list to /h/.config/ws/forbidden-words, mode 600"},
		"none on the Mac": {ok, "bbb\n", "", "", "would copy"},
		"link":            {ok, "bbb\n", "", "/elsewhere", "is a link to /elsewhere"},
		"root refused":    {effecttest.Result{Err: sshFailed("denied")}, "bbb\n", "x", "", ""},
	} {
		w, m := newWords(tc.login, tc.remote)
		if tc.mac != "" {
			m.Files[w.file()] = []byte(tc.mac)
		}
		if tc.link != "" {
			m.Links[w.file()] = tc.link
		}
		p, err := w.Plan(context.Background())
		got := errText(err)
		for _, c := range p.Changes {
			got += c.String()
		}
		if !strings.Contains(got, tc.want) || (tc.want == "" && got != "") {
			t.Errorf("%s: Plan = %q; want %q", name, got, tc.want)
		}
	}
}

func TestWordsApply(t *testing.T) {
	w, m := newWords(effecttest.Result{Out: []byte("ok\n")}, "aaa\n")
	files := w.Files
	m.Results[remoteLine(files, "words", "aaa")] = effecttest.Result{Out: []byte("list")}
	p := step.Plan{Changes: []step.Change{wordsCopy{w.file(), "aaa"}}}
	tmp := w.file() + ".new"
	m.Results["mv -f "+tmp+" "+w.file()] = effecttest.Result{}
	if err := w.Apply(context.Background(), p); err != nil || string(m.Files[tmp]) != "list" {
		t.Fatalf("Apply = %v, new file %q; want the list, renamed over the old", err, m.Files[tmp])
	}
	m.Results[hashOf] = effecttest.Result{Out: []byte("other\n")}
	delete(m.Files, tmp)
	if err := w.Apply(context.Background(), p); err == nil || m.Files[tmp] != nil {
		t.Errorf("Apply with a changed hash = %v, file %q; want a refusal and no file", err, m.Files[tmp])
	}
}
