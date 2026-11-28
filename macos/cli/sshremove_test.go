package cli

import (
	"slices"
	"strings"
	"testing"
)

func TestSSHRemoveHandsSewardTheNameAndTheAnswer(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"a.b_c"}, "exec /usr/local/bin/seward remove a.b_c"},
		{[]string{"a.b_c", "--yes"}, "exec /usr/local/bin/seward remove a.b_c --yes"},
	} {
		status, stderr, ran := sshKeys(501, append([]string{"remove"}, tc.args...)...)
		if status != 0 || !slices.Equal(ran, []string{tc.want}) {
			t.Errorf("%q: status = %d, stderr = %q, execs = %q; want 0 and %q", tc.args, status, stderr, ran, tc.want)
		}
	}
}

func TestSSHRemoveRefusesABadNameBeforeAnyRun(t *testing.T) {
	for _, name := range []string{"", "../x", "a/b", "id.pub", "seward.sock"} {
		status, stderr, ran := sshKeys(501, "remove", name, "--yes")
		if status != 2 || len(ran) != 0 || !strings.Contains(stderr, "name") {
			t.Errorf("%q: status = %d, stderr = %q, execs = %q; want 2, the name's rule and no run", name, status, stderr, ran)
		}
	}
}

func TestSSHRemoveTakesOneName(t *testing.T) {
	if status, _, ran := sshKeys(501, "remove"); status != 2 || len(ran) != 0 {
		t.Errorf("status = %d, execs = %q; want 2 and no run", status, ran)
	}
}

func TestSSHRemoveRefusesRoot(t *testing.T) {
	status, stderr, ran := sshKeys(0, "remove", "ws-auth", "--yes")
	if status != 1 || len(ran) != 0 || !strings.Contains(stderr, "not root") {
		t.Errorf("status = %d, stderr = %q, execs = %q; want 1, the refusal and no run", status, stderr, ran)
	}
}
