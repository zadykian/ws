package cli

import (
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/zadykian/ws/engine/cli/clitest"
)

// runIncluded runs ws client macos ROLE included with text on stdin.
func runIncluded(role, text string) (status int, stderr string) {
	p := fakeProcess()
	status, _, stderr = clitest.Run(p, []*cobra.Command{clientGroup(p)}, text, "client", "macos", role, "included")
	return status, stderr
}

func TestSSHIncludedSaysByItsStatus(t *testing.T) {
	for text, want := range map[string]int{
		"Include /etc/ws/ssh_config\n\nHost *\n    User me\n": 0,
		"# mine\ninclude \"/etc/ws/ssh_config\"\n":            0,
		"Host *\n    User me\n":                               1,
		"":                                                    1,
		"User me\nInclude /etc/ws/ssh_config\n":               2,
	} {
		status, stderr := runIncluded("ssh", text)
		if status != want || (want == 2) != strings.Contains(stderr, "line 2 of ~/.ssh/config") {
			t.Errorf("%q: status = %d, stderr = %q; want %d", text, status, stderr, want)
		}
	}
}
