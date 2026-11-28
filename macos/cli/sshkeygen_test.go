package cli

import (
	"slices"
	"strings"
	"testing"
)

func TestSSHKeygenHandsSewardTheNameAndTheStore(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"ws-auth"}, "exec /usr/local/bin/seward keygen ws-auth"},
		{[]string{"ws-auth", "--store", "enclave"}, "exec /usr/local/bin/seward keygen ws-auth --store enclave"},
		{[]string{"a.b_c", "--store", "file"}, "exec /usr/local/bin/seward keygen a.b_c --store file"},
		{[]string{"ws-auth", "--store", "other"}, "exec /usr/local/bin/seward keygen ws-auth --store other"},
	} {
		status, stderr, ran := sshKeys(501, append([]string{"keygen"}, tc.args...)...)
		if status != 0 || !slices.Equal(ran, []string{tc.want}) {
			t.Errorf("%q: status = %d, stderr = %q, execs = %q; want 0 and %q", tc.args, status, stderr, ran, tc.want)
		}
	}
}

func TestSSHKeygenRefusesABadNameBeforeAnyRun(t *testing.T) {
	for _, name := range []string{"", "../x", "a/b", "id.pub", "a b", "seward.sock", "seward.lock"} {
		status, stderr, ran := sshKeys(501, "keygen", name)
		if status != 2 || len(ran) != 0 || !strings.Contains(stderr, "name") {
			t.Errorf("%q: status = %d, stderr = %q, execs = %q; want 2, the name's rule and no run", name, status, stderr, ran)
		}
	}
}

func TestSSHKeygenTakesOneName(t *testing.T) {
	for _, args := range [][]string{{"keygen"}, {"keygen", "a", "b"}} {
		if status, _, ran := sshKeys(501, args...); status != 2 || len(ran) != 0 {
			t.Errorf("%q: status = %d, execs = %q; want 2 and no run", args, status, ran)
		}
	}
}

func TestSSHKeygenRefusesRoot(t *testing.T) {
	status, stderr, ran := sshKeys(0, "keygen", "ws-auth")
	if status != 1 || len(ran) != 0 || !strings.Contains(stderr, "not root") {
		t.Errorf("status = %d, stderr = %q, execs = %q; want 1, the refusal and no run", status, stderr, ran)
	}
}
