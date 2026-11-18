package cli

import (
	"testing"

	"github.com/zadykian/ws/engine/cli/clitest"
)

// TestNoRoleSetsAFramesKey keeps each role off the keys of the server's frame, which loads before
// the roles through the side's link.
func TestNoRoleSetsAFramesKey(t *testing.T) {
	clitest.FrameKeys(t, []string{treeRoot + "/mise.toml", treeRoot + "/mise.server.toml"}, rolesDir+"/*/mise.*.toml")
}
