package cmd

import (
	"bytes"
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

func TestWizardCheckAtATerminalGoesStraightToTheRun(t *testing.T) {
	p, m := macTerminal()
	_, stdout, stderr := runWs(p, "", "client", "macos", "wizard", "--check")
	if stdout != "" || !strings.Contains(stderr, enterScreen) || !strings.Contains(stderr, leaveScreen) {
		t.Errorf("stdout = %q, stderr = %q", stdout, stderr)
	}
	if !slices.ContainsFunc(m.Calls, func(c string) bool { return strings.HasPrefix(c, "run caffeinate") }) {
		t.Errorf("no run began: calls = %q", m.Calls)
	}
}

func TestWizardPlainDrawsNoScreenAtATerminal(t *testing.T) {
	p, _ := macTerminal()
	_, _, stderr := runWs(p, "", "client", "macos", "wizard", "--plain", "--check")
	if strings.Contains(stderr, enterScreen) {
		t.Errorf("--plain drew: %q", stderr)
	}
}

func TestWizardOpensTheListsAndLeavesWithQ(t *testing.T) {
	p, m := macTerminal()
	status, stdout, stderr := runWs(p, "q", "client", "macos", "wizard")
	if status != 0 || stdout != "" || !strings.Contains(stderr, enterScreen) || !strings.Contains(stderr, leaveScreen) {
		t.Fatalf("status = %d, stdout = %q, stderr = %q", status, stdout, stderr)
	}
	if !slices.Contains(m.Calls, "run xcode-select -p") || slices.Contains(m.Calls, "run xcode-select --install") {
		t.Errorf("calls = %q, want a check of the tools alone", m.Calls)
	}
}

func TestWizardCutAtTheListsEndsWithStatusOne(t *testing.T) {
	p, _ := macTerminal()
	var out, errs bytes.Buffer
	root := cli.Root(p, newMacCmd(p), newServerCmd(p))
	status := cli.Execute(context.Background(), root, []string{"client", "macos", "wizard"}, repeating("\x03"), &out, &errs)
	if status != 1 {
		t.Errorf("status = %d, want the plain wizard's 1: %q", status, errs.String())
	}
}

// freed is a machine whose rm frees the lock, for the run that a scan's end lets in.
type freed struct{ *effecttest.Machine }

func (f freed) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	if name == "rm" {
		delete(f.Dirs, args[len(args)-1])
	}
	return f.Machine.Run(ctx, name, args...)
}

func TestWizardRunsTheStagesChosenAlone(t *testing.T) {
	p, m := macTerminal()
	m.Results["xcode-select -p"] = effecttest.Result{Out: []byte("/x\n")}
	m.Files["/x/usr/bin/git"], m.Modes["/x/usr/bin/git"] = nil, 0o755
	p.Machine = freed{m}
	var out bytes.Buffer
	root := cli.Root(p, newMacCmd(p), newServerCmd(p))
	keys := newAfterScan("n \r")
	status := cli.Execute(context.Background(), root, []string{"client", "macos", "wizard"}, keys, &out, keys)
	if status != 0 || strings.Count(strings.Join(m.Calls, "\n"), "run caffeinate") != 2 {
		t.Fatalf("status = %d, want a scan and a run: calls = %q", status, m.Calls)
	}
	if strings.Contains(strings.Join(m.Calls, "\n"), "run curl") {
		t.Errorf("Homebrew applied, left out: calls = %q", m.Calls)
	}
}
