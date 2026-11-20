package secrets_test

import (
	"strings"
	"testing"
)

// The profile is the hostname: one DNS label, and no recipient's label, nano or backup, nor a name
// that fnox gives a file of its own, local or default.
func TestTaskRefusesAHostnameThatNamesNoProfile(t *testing.T) {
	f := newFixture(t)
	for _, host := range []string{
		"a/b", "", "Ws", "ws.lan", "-ws", "ws-", strings.Repeat("a", 64),
		"nano", "backup", "local", "default",
	} {
		f.env["FAKE_HOST"] = host
		code, out := f.run()
		f.refused(code, out, "one label")
	}
}

// A label of 63 characters, the most that DNS takes, names a profile.
func TestTaskTakesAHostnameOfTheLongestLabel(t *testing.T) {
	f := newFixture(t)
	host := "ws-0" + strings.Repeat("a", 59)
	f.push("fnox."+host+".toml", "[providers.host]\ntype = \"age\"\n")
	f.env["FAKE_HOST"] = host
	if code, out := f.run(); code != 0 {
		t.Fatalf("the task exits %d for a label of %d characters, want 0:\n%s", code, len(host), out)
	}
	if fnox := f.log("fnox"); !strings.Contains(fnox, "argv=-P "+host+" ") {
		t.Errorf("fnox ran under no profile %s:\n%s", host, fnox)
	}
}
