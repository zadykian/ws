package cli

import (
	"bytes"
	"strings"
	"testing"

	enginecli "github.com/zadykian/ws/engine/cli"
	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

const guestsFile = "WS_INCUS_GUESTS:\n  - {name: lab, address: 10.100.0.10, cpu: 4, memory: 8GiB, disk: 200GiB}\n"

// runGuests runs ws ubuntu host guests with stdin, and returns the standard output and the error.
func runGuests(m *effecttest.Machine, stdin string, args ...string) (string, error) {
	c := newGuestsCmd(enginecli.Env{Machine: m})
	var out bytes.Buffer
	c.SetArgs(args)
	c.SetIn(strings.NewReader(stdin))
	c.SetOut(&out)
	c.SetErr(&bytes.Buffer{})
	err := c.Execute()
	return out.String(), err
}

func TestGuestsListShowsTheListThenTheStrays(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files[hostVars] = []byte(guestsFile + "  - {name: build, address: 10.100.0.11, cpu: 2, memory: 4GiB, disk: 100GiB}\n")
	m.Results["incus list --format json"] = effecttest.Result{Out: []byte(
		`[{"name":"lab","status":"Running","profiles":["ws"]},{"name":"old","status":"Stopped","profiles":["ws"]}]`)}
	out, err := runGuests(m, "", "list")
	want := "lab\t10.100.0.10\tRunning\nbuild\t10.100.0.11\tmissing\nold\t-\tStopped\tnot in WS_INCUS_GUESTS\n"
	if err != nil || out != want {
		t.Errorf("list = %q, %v, want %q", out, err, want)
	}
}

func TestGuestsSetupRunsTheGuestsSetupWhateverItsPlanSays(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files[hostVars] = []byte(guestsFile)
	m.Files["/usr/local/bin/ws"] = []byte("binary")
	m.Results["incus list --format json"] = effecttest.Result{Out: []byte(`[{"name":"lab","status":"Stopped","profiles":["ws"]}]`)}
	m.Results["incus file push --create-dirs --uid 0 --gid 0 --mode 0755 /usr/local/bin/ws lab/usr/local/bin/ws"] = effecttest.Result{}
	m.Results["incus exec lab -- sh -c systemctl is-system-running --wait >/dev/null 2>&1; cloud-init status --wait >/dev/null sh"] = effecttest.Result{}
	m.Results["incus exec lab -- /usr/local/bin/ws ubuntu guest setup --secrets"] = effecttest.Result{}
	if _, err := runGuests(m, "", "setup", "lab", "--secrets"); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, c := range m.Calls {
		found = found || c == "run incus exec lab -- /usr/local/bin/ws ubuntu guest setup --secrets"
	}
	if !found {
		t.Errorf("calls %q lack the setup with --secrets", m.Calls)
	}
	if _, err := runGuests(m, "", "setup", "other"); err == nil || !strings.Contains(err.Error(), "names no guest other") {
		t.Errorf("a guest that the list lacks is refused: %v", err)
	}
}

func TestGuestsRemoveAsksThenDeletesAndNamesTheMacsCommand(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files[hostVars] = []byte(guestsFile)
	m.Results["incus list --format json"] = effecttest.Result{Out: []byte(`[{"name":"old","status":"Stopped","profiles":["ws"]},{"name":"x","profiles":["default"]}]`)}
	m.Results["incus delete --force old"] = effecttest.Result{}
	if _, err := runGuests(m, "no\n", "remove", "old"); err == nil || !strings.Contains(err.Error(), "not confirmed") {
		t.Errorf("a no deletes nothing: %v", err)
	}
	if _, err := runGuests(m, "", "remove", "old"); err == nil {
		t.Errorf("no answer deletes nothing")
	}
	for _, c := range m.Calls {
		if strings.Contains(c, "delete") {
			t.Fatalf("calls %q deleted before a yes", m.Calls)
		}
	}
	out, err := runGuests(m, "yes\n", "remove", "old")
	if err != nil || !strings.Contains(out, "ws macos remove old") {
		t.Errorf("remove = %q, %v, want the Mac's command named", out, err)
	}
}

func TestGuestsRemoveRefusesAListedGuestAndAnInstanceOfAnotherProfile(t *testing.T) {
	m := effecttest.NewMachine()
	m.Files[hostVars] = []byte(guestsFile)
	m.Results["incus list --format json"] = effecttest.Result{Out: []byte(`[{"name":"lab","profiles":["ws"]},{"name":"x","profiles":["default"]}]`)}
	if _, err := runGuests(m, "yes\n", "remove", "lab"); err == nil || !strings.Contains(err.Error(), "still names lab") {
		t.Errorf("a listed guest: %v", err)
	}
	if _, err := runGuests(m, "yes\n", "remove", "x"); err == nil || !strings.Contains(err.Error(), "no guest of ws's") {
		t.Errorf("another profile's instance: %v", err)
	}
}
