// Package secretstest fakes the secrets repository on a Mac's effecttest.Machine. The commands that
// change it through a clone test against it: the clone, fnox in it, gh's deploy keys and the
// landing of a change.
package secretstest

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/roles/client/macos/secrets"
)

// The repository o/n, as the tree's mise.toml names it, the pull request that a landing opens and
// the commit it merges.
const (
	URL  = "git@github.com:o/n.git"
	Slug = "o/n"
	Pull = "https://github.com/o/n/pull/1"
	Head = "abc123"
)

// Mac is a Mac whose tree names o/n and whose gh reads it as private. Its clone holds the files
// that New starts it with, and the checks' workflow. gh lists Keys as the deploy keys, and adds
// and deletes any. Every fnox and git call in the clone passes, both checks pass, and the merge
// passes. A line that the test sets in Results beforehand answers as the test set it.
type Mac struct {
	*effecttest.Machine
	// Base is the directory that mktemp names, and Dir the clone in it.
	Base, Dir string
	// Keys are the deploy keys that gh lists.
	Keys []secrets.DeployKey

	start  map[string][]byte
	ours   map[string]bool
	staged []string
	landed []secrets.Change
	fnoxed [][]string
}

// New gives m the repository o/n, whose clone holds files by name. It keeps the directory of a
// mktemp -d that m already answers, so that the test's other commands keep theirs.
func New(m *effecttest.Machine, files map[string]string) *Mac {
	base := "/tmp/ws.secrets"
	if r, ok := m.Results["mktemp -d"]; ok {
		base = strings.TrimSpace(string(r.Out))
	}
	f := &Mac{Machine: m, Base: base, Dir: base + "/repo", start: map[string][]byte{}, ours: map[string]bool{}}
	m.Files[secrets.Tree+"/mise.toml"] = []byte("[env]\n" + secrets.RepoKey + " = \"" + URL + "\"\n")
	m.Files[f.Dir+"/.github/workflows/check.yml"] = []byte("on: pull_request\n")
	m.Dirs[f.Dir] = 0o755
	for name, text := range files {
		m.Files[f.Dir+"/"+name] = []byte(text)
		f.start[name] = []byte(text)
	}
	for line, out := range map[string]string{
		secrets.GH + " api repos/" + Slug + " --jq .private": "true\n",
		"mktemp -d": base + "\n",
		"env GIT_TERMINAL_PROMPT=0 git clone -q --branch main --single-branch https://github.com/" + Slug + ".git " + f.Dir: "",
		"rm -rf " + base:                      "",
		"git -C " + f.Dir + " rev-parse HEAD": Head + "\n",
		secrets.GH + " pr checks " + Pull + " --json name,bucket": `[{"name":"values","bucket":"pass"},` +
			`{"name":"gitleaks","bucket":"pass"}]`,
		secrets.GH + " pr merge " + Pull + " --merge --delete-branch --match-head-commit " + Head: "",
	} {
		if _, ok := m.Results[line]; !ok {
			m.Results[line] = effecttest.Result{Out: []byte(out)}
		}
	}
	return f
}

// Run answers the clone's lines, then runs the command on the Machine.
func (f *Mac) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	f.answer(name, args)
	return f.Machine.Run(ctx, name, args...)
}

// RunInput answers the clone's lines, then runs the command on the Machine.
func (f *Mac) RunInput(ctx context.Context, stdin []byte, name string, args ...string) ([]byte, error) {
	f.answer(name, args)
	return f.Machine.RunInput(ctx, stdin, name, args...)
}

// Pass answers the clone's lines, then passes the command on the Machine.
func (f *Mac) Pass(ctx context.Context, root bool, name string, args ...string) error {
	f.answer(name, args)
	return f.Machine.Pass(ctx, root, name, args...)
}

// Landed are the changes that pull requests opened, in their order.
func (f *Mac) Landed() []secrets.Change { return slices.Clone(f.landed) }

// Fnoxed are the arguments of each fnox run in the clone, in their order.
func (f *Mac) Fnoxed() [][]string { return slices.Clone(f.fnoxed) }

// Files are the clone's fnox files as they are now, by name.
func (f *Mac) Files() map[string]string {
	files := map[string]string{}
	for path, data := range f.Machine.Files {
		if name, ok := strings.CutPrefix(path, f.Dir+"/"); ok && strings.HasPrefix(name, "fnox.") {
			files[name] = string(data)
		}
	}
	return files
}

// answer sets the result of a line of the clone, gh's deploy keys or fnox, where the test set none.
func (f *Mac) answer(name string, args []string) {
	line := strings.Join(append([]string{name}, args...), " ")
	if _, set := f.Results[line]; set && !f.ours[line] {
		return
	}
	out, ok := f.reply(name, args)
	if ok {
		f.Results[line] = effecttest.Result{Out: []byte(out)}
		f.ours[line] = true
	}
}

// reply is the output of a line that the fake answers, and whether it answers it.
func (f *Mac) reply(name string, args []string) (string, bool) {
	if i := slices.Index(args, f.Dir); name == "env" && i >= 0 && i+1 < len(args) && args[i+1] == "fnox" {
		f.fnoxed = append(f.fnoxed, slices.Clone(args[i+2:]))
		return "", true
	}
	if name == "env" && len(args) > 1 && args[0] == "GIT_TERMINAL_PROMPT=0" {
		name, args = args[1], args[2:]
	}
	switch {
	case name == "git" && len(args) > 2 && args[0] == "-C" && args[1] == f.Dir:
		return f.git(args[2:])
	case name == secrets.GH && len(args) > 1 && args[0] == "pr" && args[1] == "create":
		f.landed = append(f.landed, landing(args[2:], f.staged))
		return "Creating pull request\n" + Pull + "\n", true
	case name == secrets.GH && slices.Equal(args, []string{"api", "--paginate", "repos/" + Slug + "/keys", "--jq",
		".[] | [.id, .title, .key, .read_only] | @tsv"}):
		var out strings.Builder
		for _, k := range f.Keys {
			fmt.Fprintf(&out, "%d\t%s\t%s\t%t\n", k.ID, k.Title, k.Key, k.ReadOnly)
		}
		return out.String(), true
	case name == secrets.GH && len(args) > 3 && args[0] == "api" && args[1] == "-X" &&
		strings.HasPrefix(args[3], "repos/"+Slug+"/keys"):
		return "", true
	}
	return "", false
}

// git answers git's commands in the clone: its status from the files' bytes, and the rest empty.
func (f *Mac) git(args []string) (string, bool) {
	files := args[slices.Index(args, "--")+1:]
	switch args[0] {
	case "status":
		var out strings.Builder
		for _, name := range files {
			now, there := f.Machine.Files[f.Dir+"/"+name]
			before, was := f.start[name]
			switch {
			case there && !was:
				fmt.Fprintf(&out, "?? %s\n", name)
			case there && !slices.Equal(now, before):
				fmt.Fprintf(&out, " M %s\n", name)
			}
		}
		return out.String(), true
	case "add":
		f.staged = slices.Clone(files)
		return "", true
	case "switch", "commit", "push":
		return "", true
	}
	return "", false
}

// landing reads the change of gh pr create's flags: the topic from the branch's name,
// ws/TOPIC-YYYYMMDD-HHMMSS, and the files from the last git add.
func landing(args, staged []string) secrets.Change {
	flags := map[string]string{}
	for i := 0; i+1 < len(args); i += 2 {
		flags[args[i]] = args[i+1]
	}
	topic := strings.TrimPrefix(flags["--head"], "ws/")
	if len(topic) > len("-20060102-150405") {
		topic = topic[:len(topic)-len("-20060102-150405")]
	}
	return secrets.Change{Topic: topic, Title: flags["--title"], Body: flags["--body"], Files: slices.Clone(staged)}
}
