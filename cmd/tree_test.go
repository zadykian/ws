package cmd

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

var rewriteGolden = flag.Bool("update", false, "write testdata/tree.golden")

// listing returns the help of root and of each command below it, in order, each after a line with
// its path.
func listing(t *testing.T, root *cobra.Command) string {
	var sb strings.Builder
	walk(root, func(c *cobra.Command) {
		path := strings.TrimPrefix(strings.TrimPrefix(c.CommandPath(), "ws"), " ")
		var out, errs bytes.Buffer
		args := append(strings.Fields(path), "--help")
		fresh := newRoot(fakeProcess(), newMacCmd(fakeProcess()), newServerCmd(fakeProcess()))
		if status := execute(context.Background(), fresh, args, strings.NewReader(""), &out, &errs); status != 0 {
			t.Errorf("ws %s: status = %d, stderr = %q", path, status, errs.String())
		}
		fmt.Fprintf(&sb, "== %s\n%s", c.CommandPath(), out.String())
	})
	return sb.String()
}

// TestTreeHoldsTheCommandsAndTheirHelp holds the whole tree against a golden file, with both
// groups on every system. A move of the commands then shows any change of a name or a text.
func TestTreeHoldsTheCommandsAndTheirHelp(t *testing.T) {
	p := fakeProcess()
	got := listing(t, newRoot(p, newMacCmd(p), newServerCmd(p)))
	const golden = "testdata/tree.golden"
	if *rewriteGolden {
		if err := os.WriteFile(golden, []byte(got), 0o644); err != nil { //nolint:gosec // a test's data file
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if got != string(want) {
		t.Errorf("the tree differs from %s: run go test ./cmd -run TestTree -update and read the diff", golden)
	}
}
