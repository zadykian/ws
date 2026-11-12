package gitcfg_test

import (
	"testing"

	"github.com/zadykian/ws/roles/client/macos/internal/gitcfg"
)

func TestWhyNamesWhatGhLacks(t *testing.T) {
	for name, tc := range map[string]struct {
		status int
		stderr string
		want   string
	}{
		"no login":  {4, "To get started\n", "gh has no login: gh auth login -h github.com -s write:ssh_signing_key logs in"},
		"no scope":  {1, "gh: Not Found (HTTP 404)\n", "gh's login lacks the write:ssh_signing_key scope: gh auth refresh -h github.com -s write:ssh_signing_key adds it"},
		"other":     {1, "one\nHTTP 500: Server Error.\n", "gh failed: HTTP 500: Server Error"},
		"no errors": {2, "", "gh failed: exit 2"},
	} {
		if got := gitcfg.Why(tc.status, tc.stderr); got != tc.want {
			t.Errorf("%s: Why = %q, want %q", name, got, tc.want)
		}
	}
}
