package cli_test

import (
	"context"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/cli/clitest"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

// A plan, which writes no input's file, reads the input from the run's env, as the run writes it. It
// does so in the run of the side's task alone: a shell's stale value counts nowhere else.
func TestUnitReadTakesAnInputOfTheRun(t *testing.T) {
	specs := map[string]string{
		"r/units.toml.tmpl": "[file.a]\npath = \"/etc/a\"\ntemplate = \"a.tmpl\"\n",
		"r/a.tmpl":          `{{ envvar "/etc/env" "IN" }}` + "\n",
	}
	for _, c := range []struct {
		env  map[string]string
		want string
	}{
		{map[string]string{"MISE_TASK_NAME": "inputs"}, "file/a\tin-sync\n"},
		{map[string]string{"MISE_TASK_NAME": "inputs", "IN": "old"}, "file/a\tin-sync\n"},
		{map[string]string{"MISE_TASK_NAME": "inputs", "IN": "new"}, "file/a\twrite /etc/a"},
		{map[string]string{"IN": "new"}, "file/a\tin-sync\n"},
		{map[string]string{"MISE_TASK_NAME": "other", "IN": "new"}, "file/a\tin-sync\n"},
	} {
		e, m := unitEnv(t, specs)
		c.env["WS_UNITS"] = e.Getenv("WS_UNITS")
		e = clitest.WithEnv(e, c.env)
		m.Files["/etc/env"], m.Files["/etc/a"] = []byte("IN=old\n"), []byte("old\n")
		side := cli.UnitSide{
			Dir: "/nowhere",
			Values: func(context.Context, effect.Machine) (any, error) {
				return unitValues{User: "u"}, nil
			},
			Units: &effecttest.Units{}, Root: true, Hidden: []string{"IN"}, Task: "inputs",
		}
		status, stdout, stderr := clitest.Run(e, []*cobra.Command{cli.Unit(e, side)}, "", "unit", "read", "file/a")
		if status != 0 || !strings.HasPrefix(stdout, c.want) {
			t.Errorf("env %q: status = %d, stdout = %q, stderr = %q; want %q", c.env, status, stdout, stderr, c.want)
		}
		if strings.Contains(stdout+stderr, "new") {
			t.Errorf("env %q: the output holds the value: %q, %q", c.env, stdout, stderr)
		}
	}
}

// A side with no task reads no input, though MISE_TASK_NAME is unset and so equals its empty name.
func TestTaskEnvReadsTheNamedTasksRunAlone(t *testing.T) {
	for _, c := range []struct {
		task string
		env  map[string]string
		ok   bool
	}{
		{"inputs", map[string]string{"MISE_TASK_NAME": "inputs", "IN": ""}, true},
		{"inputs", map[string]string{"MISE_TASK_NAME": "inputs"}, false},
		{"inputs", map[string]string{"IN": "v"}, false},
		{"", map[string]string{"IN": "v"}, false},
		{"", map[string]string{"MISE_TASK_NAME": "", "IN": "v"}, false},
	} {
		e := clitest.WithEnv(cli.Env{}, c.env)
		if _, ok := e.TaskEnv(c.task)("IN"); ok != c.ok {
			t.Errorf("task %q, env %q: IN set = %v, want %v", c.task, c.env, ok, c.ok)
		}
	}
}
