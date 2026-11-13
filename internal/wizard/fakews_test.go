package wizard

import (
	"context"
	"errors"
	"github.com/zadykian/ws/engine/core/remote"
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

// fakeWs is a Shell that plays ws on a test machine. The scripts' command, sh -s -- OP ARG..., runs
// on M as the command "op OP ARG...", so a test names the op alone and not the way to ws. Any other
// command runs on M unchanged.
type fakeWs struct{ M *effecttest.Machine }

// opLine returns the Results key of the op with its args. Each word goes as SSHShell quotes it, so
// a word with a blank keys apart from the words it holds, as on ws.
func opLine(op ...string) string {
	line := "op"
	for _, word := range op {
		line += " " + remote.Quote(word)
	}
	return line
}

// command returns the name and args that M runs for cmd. An op runs as its line, with no args.
func (f fakeWs) command(cmd []string) (string, []string) {
	if len(cmd) >= 4 && slices.Equal(cmd[:3], []string{"sh", "-s", "--"}) {
		return opLine(cmd[3:]...), nil
	}
	return cmd[0], cmd[1:]
}

func (f fakeWs) Run(ctx context.Context, in []byte, cmd ...string) ([]byte, error) {
	name, args := f.command(cmd)
	return f.M.RunInput(ctx, in, name, args...)
}

func (f fakeWs) Pass(ctx context.Context, in []byte, cmd ...string) error {
	name, args := f.command(cmd)
	return f.M.PassInput(ctx, in, name, args...)
}

func (f fakeWs) Lines(ctx context.Context, in []byte, take func(line string) bool, cmd ...string) error {
	name, args := f.command(cmd)
	return f.M.PassLines(ctx, in, take, name, args...)
}

// fakeReach is a Reach whose shell is fakeWs on M.
type fakeReach struct {
	Reach
	M *effecttest.Machine
}

func (r fakeReach) Shell() remote.Shell { return fakeWs{M: r.M} }

func TestFakeWsKeysAScriptByItsOp(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[opLine("link", "eth1")] = effecttest.Result{Out: []byte("ok")}
	m.Results["uname -r"] = effecttest.Result{Out: []byte("7.0")}
	var f remote.Shell = fakeWs{M: m}
	ctx := context.Background()
	if out, err := f.Run(ctx, []byte("echo\n"), script([]string{"link", "eth1"})...); err != nil || string(out) != "ok" {
		t.Errorf("Run of the op = %q, %v", out, err)
	}
	if out, err := f.Run(ctx, nil, "uname", "-r"); err != nil || string(out) != "7.0" {
		t.Errorf("Run of a plain command = %q, %v", out, err)
	}
	if got := m.Inputs[0]; got != "echo\n" {
		t.Errorf("the script on stdin = %q", got)
	}
}

func TestFakeWsPasses(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[opLine("install", "v1")] = effecttest.Result{Err: errors.New("denied")}
	m.Results[opLine("run")] = effecttest.Result{Out: []byte("event\nother\n")}
	var f remote.Shell = fakeWs{M: m}
	ctx := context.Background()
	if err := f.Pass(ctx, nil, script([]string{"install", "v1"})...); err == nil {
		t.Error("Pass of the op: err = nil")
	}
	if err := f.Lines(ctx, nil, func(line string) bool { return line == "event" }, script([]string{"run"})...); err != nil {
		t.Fatal(err)
	}
	if want := []string{"other"}; !slices.Equal(m.Passed, want) {
		t.Errorf("passed = %q; want %q", m.Passed, want)
	}
}

func TestFakeWsKeepsAWordWhole(t *testing.T) {
	m := effecttest.NewMachine()
	m.Results[opLine("run", "--only", "network")] = effecttest.Result{Out: []byte("two")}
	m.Results[opLine("run", "--only network")] = effecttest.Result{Out: []byte("one")}
	var f remote.Shell = fakeWs{M: m}
	for want, op := range map[string][]string{"two": {"run", "--only", "network"}, "one": {"run", "--only network"}} {
		if out, err := f.Run(context.Background(), nil, script(op)...); err != nil || string(out) != want {
			t.Errorf("Run of %q = %q, %v; want %q", op, out, err, want)
		}
	}
}
