package devtools_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/resource/file"
)

// The token file's group unit gives the file the account's group, by id's answer, and its check
// fails on another group and passes without the file.
func TestGHTokenGroupScripts(t *testing.T) {
	r := readRuns(t)["devtools-gh-token-group"]
	if len(r.Command) != 6 || len(r.Check) != 6 || r.Command[4] != "/etc/ws/gh-token" || r.Command[5] != "{{ .User }}" {
		t.Fatalf("command %q, check %q; want a shell line of the file and the account", r.Command, r.Check)
	}
	bin, dir := t.TempDir(), t.TempDir()
	tokenFile, calls := filepath.Join(dir, "gh-token"), filepath.Join(dir, "calls")
	fake(t, bin, "id", `[ "$*" = "-g -- acct" ] && echo 4242`)
	fake(t, bin, "chgrp", `echo "chgrp $*" >>`+calls)
	fake(t, bin, "stat", `cat `+filepath.Join(dir, "gid"))
	group := func(argv []string) (string, bool) {
		cmd := exec.Command("sh", "-c", argv[2], argv[3], tokenFile, "acct") //nolint:gosec // G204: the role's own line
		cmd.Env = []string{"PATH=" + bin + ":/usr/bin:/bin"}
		out, err := cmd.CombinedOutput()
		return string(out), err == nil
	}
	if out, ok := group(r.Check); !ok {
		t.Errorf("the check failed without the file: %s", out)
	}
	if out, ok := group(r.Command); !ok {
		t.Errorf("the command failed without the file: %s", out)
	}
	// A file by hand stops the command, which names it and none of its text, on the account's group too.
	if err := os.WriteFile(tokenFile, []byte("by hand, dummy\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "gid"), []byte("4242\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, ok := group(r.Check); ok {
		t.Error("the check passed on a file by hand")
	}
	if out, ok := group(r.Command); ok || !strings.Contains(out, tokenFile+": not written by ws") || strings.Contains(out, "dummy") {
		t.Errorf("the command on a file by hand: ok %v, output %q; want a failure that names the file alone", ok, out)
	}
	if _, err := os.Stat(calls); err == nil {
		t.Error("the command changed the group of a file by hand")
	}
	for name, text := range map[string]string{"gh-token": file.Header("#") + "\n" + "dummy\n", "gid": "0\n"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if _, ok := group(r.Check); ok {
		t.Error("the check passed on root's group")
	}
	if out, ok := group(r.Command); !ok {
		t.Errorf("the command failed: %s", out)
	}
	if got, _ := os.ReadFile(calls); string(got) != "chgrp -- 4242 "+tokenFile+"\n" { //nolint:gosec // the test's own file
		t.Errorf("calls %q, want the account's gid set on the file", got)
	}
	if err := os.WriteFile(filepath.Join(dir, "gid"), []byte("4242\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if out, ok := group(r.Check); !ok {
		t.Errorf("the check failed on the account's group: %s", out)
	}
}
