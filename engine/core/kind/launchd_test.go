package kind

import (
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/step"
)

const (
	ldProgram = "/usr/local/bin/ws"
	ldLabel   = "com.x.lab"
	ldPlist   = "/Library/LaunchDaemons/com.x.lab.plist"
	ldStamp   = "/var/db/ws/launchd/com.x.lab"
	ldPrint   = "launchctl print system/com.x.lab"
)

// ldSpec is a family of one daemon, lab, which restarts on /etc/x/key and reloads on /etc/x/lab.
const ldSpec = `[launchd.x]
prefix = "com.x."
program = "/usr/local/bin/ws"

[[launchd.x.daemon]]
name = "lab"
arguments = ["/usr/local/bin/ws", "run", "lab"]
log = "/var/log/x.lab.log"
restart_on = ["/etc/x/key"]
reload_on = ["/etc/x/lab"]
`

// ldEnv is a Mac with the program and the watched files, root's alone, and launchd's answers to
// each call; it has loaded no daemon.
func ldEnv(t *testing.T) (Env, *effecttest.Machine) {
	e, m, _, _ := testEnv()
	e.System = MacOS
	for _, d := range []string{"/", "/usr", "/usr/local", "/usr/local/bin", "/Library/LaunchDaemons", "/etc/x/lab"} {
		m.Dirs[d] = 0o755
	}
	m.Files[ldProgram], m.Files["/etc/x/key"], m.Files["/etc/x/lab/conf"] = []byte("ws 1"), []byte("key"), []byte("conf")
	m.Results[ldPrint] = effecttest.Result{Err: exited(t, exec.Command("sh", "-c", "exit 113"))}
	for _, line := range []string{"bootstrap system " + ldPlist, "bootout system/" + ldLabel,
		"kickstart -k system/" + ldLabel, "kill SIGHUP system/" + ldLabel} {
		m.Results["launchctl "+line] = effecttest.Result{}
	}
	return e, m
}

// loadedWith has launchd answer print with a running daemon of the arguments.
func loadedWith(m *effecttest.Machine, args ...string) {
	out := "system/" + ldLabel + " = {\n\tstate = running\n\targuments = {\n\t\t" + strings.Join(args, "\n\t\t") + "\n\t}\n\tpid = 7\n}\n"
	m.Results[ldPrint] = effecttest.Result{Out: []byte(out)}
}

// ldConverged is a Mac where the unit has run once and launchd runs the daemon.
func ldConverged(t *testing.T) (step.Step, *effecttest.Machine) {
	e, m := ldEnv(t)
	s := loadOne(t, e, ldSpec, nil)[0].Step
	if err := applyOnce(t, s); err != nil {
		t.Fatal(err)
	}
	loadedWith(m, ldProgram, "run", "lab")
	if got := headlines(t, s); got != nil {
		t.Fatalf("plan after a run = %q, want none", got)
	}
	m.Calls = nil
	return s, m
}

func TestLaunchdLoadsANewDaemon(t *testing.T) {
	e, m := ldEnv(t)
	if err := applyOnce(t, loadOne(t, e, ldSpec, nil)[0].Step); err != nil {
		t.Fatal(err)
	}
	want := []string{"atomic " + ldPlist + " 644", "root launchctl bootstrap system " + ldPlist, "write root " + ldStamp + " 644"}
	var got []string
	for _, c := range m.Calls {
		if strings.HasPrefix(c, "atomic ") || strings.HasPrefix(c, "write ") || strings.HasPrefix(c, "root launchctl") {
			got = append(got, c)
		}
	}
	if !slices.Equal(got, want) || m.Owners[ldPlist] != [2]int{0, 0} {
		t.Errorf("writes and calls = %q, plist's owner %v; want %q by root", got, m.Owners[ldPlist], want)
	}
	if lines := strings.Count(string(m.Files[ldStamp]), "\n"); lines != 3 || !strings.Contains(string(m.Files[ldStamp]), "  /etc/x/lab\n") {
		t.Errorf("stamp = %q, want a line for the program, the key and the directory", m.Files[ldStamp])
	}
}

func TestLaunchdInSyncAsksNoRoot(t *testing.T) {
	s, m := ldConverged(t)
	if got := headlines(t, s); got != nil {
		t.Fatalf("plan = %q, want none", got)
	}
	for _, c := range m.Calls {
		if strings.HasPrefix(c, "root ") || strings.HasPrefix(c, "write") || strings.HasPrefix(c, "atomic") {
			t.Errorf("an in-sync plan called %q, want reads alone", c)
		}
	}
}

func TestLaunchdReloadsAChangedPlist(t *testing.T) {
	s, m := ldConverged(t)
	m.Files[ldPlist] = []byte("edited")
	want := []string{"write " + ldPlist, "boot out " + ldLabel, "bootstrap " + ldLabel + " from " + ldPlist}
	if got := headlines(t, s); !slices.Equal(got, want) {
		t.Errorf("plan = %q, want %q", got, want)
	}
}

func TestLaunchdReloadsAnOlderLoadedPlist(t *testing.T) {
	s, m := ldConverged(t)
	loadedWith(m, ldProgram, "run", "old")
	want := []string{"boot out " + ldLabel, "bootstrap " + ldLabel + " from " + ldPlist}
	if got := headlines(t, s); !slices.Equal(got, want) {
		t.Errorf("plan = %q, want %q", got, want)
	}
}

func TestLaunchdRestartsOnProgram(t *testing.T) {
	s, m := ldConverged(t)
	m.Files[ldProgram] = []byte("ws 2")
	want := []string{"restart " + ldLabel + ": its program or a file of restart_on changed",
		"keep the sums of the files of " + ldLabel + " in " + ldStamp}
	if got := headlines(t, s); !slices.Equal(got, want) {
		t.Fatalf("plan = %q, want %q", got, want)
	}
	if err := applyOnce(t, s); err != nil || !slices.Contains(m.Calls, "root launchctl kickstart -k system/"+ldLabel) {
		t.Errorf("apply = %v, calls %q; want a kickstart", err, m.Calls)
	}
	if got := headlines(t, s); got != nil {
		t.Errorf("plan after the restart = %q, want none", got)
	}
}

func TestLaunchdRestartsOnRestartOn(t *testing.T) {
	s, m := ldConverged(t)
	delete(m.Files, "/etc/x/key")
	if got := headlines(t, s); len(got) != 2 || !strings.HasPrefix(got[0], "restart ") {
		t.Errorf("plan = %q, want a restart where a watched file goes", got)
	}
}

func TestLaunchdRestartIsOwedAfterAFailure(t *testing.T) {
	s, m := ldConverged(t)
	m.Files[ldProgram] = []byte("ws 2")
	m.Results["launchctl kickstart -k system/"+ldLabel] = effecttest.Result{Err: exited(t, exec.Command("sh", "-c", "exit 1"))}
	if err := applyOnce(t, s); err == nil {
		t.Fatal("apply passed, want the kickstart's failure")
	}
	if got := headlines(t, s); len(got) != 2 || !strings.HasPrefix(got[0], "restart ") {
		t.Errorf("plan after a failed restart = %q, want the restart again", got)
	}
}

func TestLaunchdSignalsOnReloadOn(t *testing.T) {
	s, m := ldConverged(t)
	m.Files["/etc/x/lab/routes"] = []byte("10.0.0.0/8")
	got := headlines(t, s)
	if len(got) != 2 || got[0] != "send SIGHUP to "+ldLabel+": a file of reload_on changed" {
		t.Fatalf("plan = %q, want a SIGHUP", got)
	}
	m.Results["launchctl kill SIGHUP system/"+ldLabel] = effecttest.Result{Err: exited(t, exec.Command("sh", "-c", "exit 1"))}
	if err := applyOnce(t, s); err != nil || !slices.Contains(m.Calls, "root launchctl kickstart -k system/"+ldLabel) {
		t.Errorf("apply = %v, calls %q; want a kickstart where the signal is refused", err, m.Calls)
	}
}

func TestLaunchdPrunesAGoneDaemon(t *testing.T) {
	e, m := ldEnv(t)
	m.Dirs["/var/db/ws/launchd"] = 0o755
	for _, f := range []string{"/Library/LaunchDaemons/com.x.old.plist", "/var/db/ws/launchd/com.x.old",
		"/var/db/ws/launchd/com.x.gone", "/Library/LaunchDaemons/com.x.a.b.plist", "/Library/LaunchDaemons/com.y.old.plist"} {
		m.Files[f] = []byte("x")
	}
	m.Results["launchctl print system/com.x.old"] = effecttest.Result{Out: []byte("system/com.x.old = {\n}\n")}
	got := headlines(t, loadOne(t, e, ldSpec, nil)[0].Step)
	want := []string{"boot out com.x.old", "remove /Library/LaunchDaemons/com.x.old.plist",
		"remove /var/db/ws/launchd/com.x.gone", "remove /var/db/ws/launchd/com.x.old"}
	if len(got) < len(want) || !slices.Equal(got[len(got)-len(want):], want) {
		t.Errorf("plan = %q, want it to end with %q", got, want)
	}
}

func TestLaunchdRefusesAnUnsafeProgram(t *testing.T) {
	e, m := ldEnv(t)
	m.Dirs["/usr/local"] = 0o775
	s := loadOne(t, e, ldSpec, nil)[0].Step
	// launchd runs a plist at boot, so the plan refuses before one is written, and with no program
	// installed yet too.
	for _, program := range []string{"there", "missing"} {
		if program == "missing" {
			delete(m.Files, ldProgram)
		}
		err := applyOnce(t, s)
		if err == nil || !strings.Contains(err.Error(), "/usr/local, on the way to "+ldProgram) {
			t.Errorf("program %s: apply = %v, want /usr/local refused", program, err)
		}
		if _, ok := m.Files[ldPlist]; ok {
			t.Errorf("program %s: the plist was written, want none", program)
		}
		for _, c := range m.Calls {
			if strings.HasPrefix(c, "root ") || strings.HasPrefix(c, "atomic ") || strings.HasPrefix(c, "write ") {
				t.Errorf("program %s: called %q, want reads alone", program, c)
			}
		}
	}
}

func TestLaunchdPlansBeforeTheProgramIsInstalled(t *testing.T) {
	e, m := ldEnv(t)
	delete(m.Files, ldProgram)
	got := headlines(t, loadOne(t, e, ldSpec, nil)[0].Step)
	if len(got) < 2 || got[0] != "create "+ldPlist {
		t.Errorf("plan = %q, want the plist and a bootstrap", got)
	}
}

func TestLaunchdPlistText(t *testing.T) {
	want, err := os.ReadFile("testdata/launchd.plist")
	if err != nil {
		t.Fatal(err)
	}
	args := []string{"/usr/local/bin/ws", "client", "tunnel", "run", "--server", "lab"}
	got := plistText("com.github.zadykian.ws-tunnel.lab", launchdDaemon{Name: "lab", Arguments: args, Log: "/var/log/ws-tunnel.lab.log"})
	if string(got) != string(want) {
		t.Errorf("plist =\n%s\nwant\n%s", got, want)
	}
	if got := plistText("a.", launchdDaemon{Arguments: []string{"/x", "a<b&c"}}); !strings.Contains(string(got), "<string>a&lt;b&amp;c</string>") ||
		strings.Contains(string(got), "StandardOutPath") {
		t.Errorf("plist =\n%s\nwant the argument escaped and no log", got)
	}
}

func TestLaunchdSpecRefusals(t *testing.T) {
	for _, c := range []struct{ name, prefix, dname, args, extra, want string }{
		{"no dot", "com.x", "lab", `"/usr/local/bin/ws"`, "", "no start of a label"},
		{"a bad name", "com.x.", "La.b", `"/usr/local/bin/ws"`, "", "a name takes"},
		{"another program", "com.x.", "lab", `"/bin/sh"`, "", "arguments start with the program"},
		{"no arguments", "com.x.", "lab", ``, "", "arguments start with the program"},
		{"a relative log", "com.x.", "lab", `"/usr/local/bin/ws"`, `log = "x.log"`, "log \"x.log\""},
		{"a relative watch", "com.x.", "lab", `"/usr/local/bin/ws"`, `reload_on = ["x"]`, "a path to watch"},
	} {
		e, _ := ldEnv(t)
		text := fmt.Sprintf("[launchd.x]\nprefix = %q\nprogram = \"/usr/local/bin/ws\"\n\n"+
			"[[launchd.x.daemon]]\nname = %q\narguments = [%s]\n%s\n", c.prefix, c.dname, c.args, c.extra)
		if _, err := Load(e, tree(map[string]string{"r/units.toml.tmpl": text})); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: load = %v, want %q", c.name, err, c.want)
		}
	}
	e, _ := ldEnv(t)
	twice := ldSpec + "\n[[launchd.x.daemon]]\nname = \"lab\"\narguments = [\"/usr/local/bin/ws\"]\n"
	if _, err := Load(e, tree(map[string]string{"r/units.toml.tmpl": twice})); err == nil || !strings.Contains(err.Error(), "a daemon's before it") {
		t.Errorf("a name twice: load = %v, want it refused", err)
	}
	e.System = Linux
	if _, err := Load(e, tree(map[string]string{"r/units.toml.tmpl": ldSpec})); err == nil || !strings.Contains(err.Error(), "the kind launchd does not run on Linux") {
		t.Errorf("on Linux: load = %v, want it refused", err)
	}
}
