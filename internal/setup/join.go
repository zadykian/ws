package setup

import (
	"context"
	"strings"

	"github.com/zadykian/ws/effect"
	"github.com/zadykian/ws/step"
)

// joinHeader is ws-join's second line, under its shebang, by which a run knows the file for its own.
const joinHeader = "# Written by ws mac apply: its next run replaces changes made here."

// Join writes ws-join into ~/.local/bin, where cld is: it joins a cld session on ws again after
// ssh drops (docs/mac-ghostty.md). The file is Script whole. One that is not ws's is left, with a
// warning.
type Join struct {
	Home string
	// Script is mac/ws-join, its second line joinHeader.
	Script []byte
	M      effect.Machine
	Out    effect.Out
}

// Name is join.
func (*Join) Name() string { return "join" }

// WaitsOn is nothing.
func (*Join) WaitsOn() string { return "" }

func (j *Join) dir() string { return j.Home + "/.local/bin" }

// joinOwn says whether a file's second line is ws-join's header.
func joinOwn(have []byte) bool {
	lines := strings.SplitN(string(have), "\n", 3)
	return len(lines) > 1 && lines[1] == joinHeader
}

// Plan plans ~/.local and its bin where missing, then ws-join, mode 755.
func (j *Join) Plan(context.Context) (step.Plan, error) {
	var cs []step.Change
	for _, dir := range []string{j.Home + "/.local", j.dir()} {
		if _, err := j.M.Stat(dir); err != nil {
			cs = append(cs, MakeDir{dir, 0o755})
		}
	}
	cs = append(cs, owned(j.M, j.Out, j.dir()+"/ws-join", j.Script, 0o755, joinOwn)...)
	return step.Plan{Changes: cs}, nil
}

// Apply makes the directories and writes the file.
func (j *Join) Apply(ctx context.Context, p step.Plan) error {
	for _, c := range p.Changes {
		if err := Apply(ctx, j.M, j.Out, c); err != nil {
			return err
		}
	}
	return nil
}

// Report says how ws-join stands.
func (j *Join) Report(ctx context.Context) ([]step.Item, error) {
	return userFile{M: j.M, Path: j.dir() + "/ws-join", Text: j.Script, Mode: 0o755, Own: joinOwn}.Report(ctx)
}
