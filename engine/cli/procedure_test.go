package cli_test

import (
	"context"
	"errors"
	"slices"
	"testing"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/cli/clitest"
	"github.com/zadykian/ws/engine/core/effect"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/resource"
)

// noted is a change that notes its apply in a shared list.
type noted struct {
	name string
	done *[]string
}

func (n noted) String() string { return n.name }

func (n noted) Apply(context.Context) error {
	*n.done = append(*n.done, n.name)
	return nil
}

// fixedPlan is a resource whose plan is its changes.
type fixedPlan []resource.Change

func (p fixedPlan) Plan(context.Context) ([]resource.Change, error) { return p, nil }

func TestSettleCheckFailsOnAChange(t *testing.T) {
	var done []string
	rs := []resource.Resource{fixedPlan{noted{"a", &done}}, fixedPlan(nil), fixedPlan{noted{"b", &done}}}
	err := cli.Settle(t.Context(), &effecttest.Out{}, rs, true)
	if !errors.Is(err, cli.ErrDue) || err.Error() != "to do: a; b" || done != nil {
		t.Errorf("settle = %v, applied %q; want both due, and none applied", err, done)
	}
	if err := cli.Settle(t.Context(), &effecttest.Out{}, []resource.Resource{fixedPlan(nil)}, true); err != nil {
		t.Errorf("nothing due: settle = %v", err)
	}
}

func TestSettleAppliesInOrder(t *testing.T) {
	var done []string
	out := &effecttest.Out{}
	rs := []resource.Resource{fixedPlan{noted{"a", &done}, noted{"b", &done}}, fixedPlan{noted{"c", &done}}}
	if err := cli.Settle(t.Context(), out, rs, false); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(done, []string{"a", "b", "c"}) || !slices.Equal(out.Lines, []string{"say: a", "say: b", "say: c"}) {
		t.Errorf("applied %q, said %q", done, out.Lines)
	}
}

// runProcedure runs a procedure of the resources rs, with args, and returns its status and output.
func runProcedure(rs []resource.Resource, args ...string) (status int, stdout, stderr string) {
	c := cli.Procedure("part", "settle the part", "It settles the part.",
		func(context.Context, effect.Out) ([]resource.Resource, error) { return rs, nil })
	return clitest.Run(clitest.Env(), []*cobra.Command{c}, "", append([]string{"part"}, args...)...)
}

func TestProcedureCheckExitsOneOnAChange(t *testing.T) {
	var done []string
	status, stdout, stderr := runProcedure([]resource.Resource{fixedPlan{noted{"a", &done}}}, "--check")
	if status != 1 || stdout != "" || done != nil {
		t.Errorf("--check: status = %d, stdout = %q, stderr = %q, applied %q; want 1, nothing on stdout, none applied",
			status, stdout, stderr, done)
	}
	if status, stdout, _ := runProcedure([]resource.Resource{fixedPlan(nil)}, "--check"); status != 0 || stdout != "" {
		t.Errorf("--check in sync: status = %d, stdout = %q; want 0 and nothing", status, stdout)
	}
}

func TestProcedureApplies(t *testing.T) {
	var done []string
	status, stdout, stderr := runProcedure([]resource.Resource{fixedPlan{noted{"a", &done}}})
	if status != 0 || stdout != "" || !slices.Equal(done, []string{"a"}) {
		t.Errorf("run: status = %d, stdout = %q, stderr = %q, applied %q; want a applied, its line on stderr",
			status, stdout, stderr, done)
	}
}
