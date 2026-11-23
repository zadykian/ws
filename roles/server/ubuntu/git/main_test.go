package git_test

import (
	"context"
	"fmt"
	"os"
	"slices"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/local"
	"github.com/zadykian/ws/roles/server/ubuntu/git"
)

// asWs names the variable under which the test's binary stands for ws in git's hooks.
const asWs = "WS_GIT_TEST_AS_WS"

// TestMain runs the tests. Under asWs the binary stands for ws as git's commit-msg hook, and runs
// what ws server git commit-msg does, on the real machine.
func TestMain(m *testing.M) {
	if os.Getenv(asWs) == "" {
		os.Exit(m.Run())
	}
	args := os.Args[1:]
	if len(args) != 4 || !slices.Equal(args[:3], []string{"server", "git", "commit-msg"}) {
		fmt.Fprintln(os.Stderr, "the test's ws runs server git commit-msg FILE alone, not", args)
		os.Exit(2)
	}
	if err := git.CommitMsg(context.Background(), local.OSMachine{}, args[3], os.Getenv("GIT_EDITOR") == ":"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
