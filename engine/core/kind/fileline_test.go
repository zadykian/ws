package kind

import (
	"errors"
	"fmt"
	"os/exec"
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

// lineSpec is a line unit for u's ~/.bashrc, with the extra fields.
func lineSpec(extra string) string {
	return "[file.l]\npath = \"/home/u/.bashrc\"\nowner = \"u\"\nline = \". /etc/x\"\n" + extra
}

// exited is the error of a command that a real machine ran, with its exit status and standard error.
func exited(t *testing.T, cmd *exec.Cmd) error {
	_, err := cmd.Output()
	var exit *exec.ExitError
	if !errors.As(err, &exit) {
		t.Fatalf("%v: %v, want an exit status", cmd.Args, err)
	}
	return fmt.Errorf("%v: %w", cmd.Args, err)
}

// copies returns the paths of the copies that a rewrite kept.
func copies(m *effecttest.Machine) []string {
	var kept []string
	for p := range m.Files {
		if strings.HasSuffix(p, "~") {
			kept = append(kept, p)
		}
	}
	return kept
}

func TestLineGoesAtTheTopAndKeepsACopy(t *testing.T) {
	e, m, _, _ := testEnv(false)
	withAccount(m)
	m.Files["/home/u/.bashrc"], m.Owners["/home/u/.bashrc"] = []byte("mine\n"), [2]int{1001, 1001}
	units := loadOne(t, e, lineSpec(""), nil)
	if err := run(t, e, units); err != nil {
		t.Fatal(err)
	}
	if got := string(m.Files["/home/u/.bashrc"]); got != ". /etc/x\n\nmine\n" {
		t.Errorf("file = %q, want the line at the top", got)
	}
	if kept := copies(m); len(kept) != 1 || string(m.Files[kept[0]]) != "mine\n" || !strings.HasPrefix(kept[0], "/home/u/.bashrc.") {
		t.Errorf("copies = %q, want the old file once", kept)
	}
	if got := headlines(t, units[0].Step); got != nil {
		t.Errorf("second plan = %q, want none", got)
	}
}

func TestLineMakesAMissingFileWithTheLineAlone(t *testing.T) {
	e, m, _, _ := testEnv(false)
	withAccount(m)
	if err := run(t, e, loadOne(t, e, lineSpec("mode = 0o600\n"), nil)); err != nil {
		t.Fatal(err)
	}
	p := "/home/u/.bashrc"
	if string(m.Files[p]) != ". /etc/x\n" || m.Owners[p] != [2]int{1001, 1001} || m.Modes[p] != 0o600 {
		t.Errorf("file %q, owner %v, mode %o; want the line alone, u's, 600", m.Files[p], m.Owners[p], m.Modes[p])
	}
}

func TestLineHeldWithSpacesPlansNothing(t *testing.T) {
	e, m, _, _ := testEnv(false)
	withAccount(m)
	m.Files["/home/u/.bashrc"] = []byte("mine\n  . /etc/x \t\n")
	if got := headlines(t, loadOne(t, e, lineSpec(""), nil)[0].Step); got != nil {
		t.Errorf("plan = %q, want none", got)
	}
}

func TestLineForRootTakesRootsHome(t *testing.T) {
	e, m, _, _ := testEnv(false)
	m.Results["getent passwd root"] = effecttest.Result{Out: []byte("root:x:0:0:root:/root:/bin/bash\n")}
	m.Dirs["/root"] = 0o700
	units := loadOne(t, e, "[file.l]\npath = \"/root/.profile\"\nowner = \"root\"\nline = \". /etc/x\"\n", nil)
	if err := run(t, e, units); err != nil {
		t.Fatal(err)
	}
	if got := string(m.Files["/root/.profile"]); got != ". /etc/x\n" || m.Owners["/root/.profile"] != [2]int{} {
		t.Errorf("file = %q, owner %v; want the line, root's", got, m.Owners["/root/.profile"])
	}
	outside := loadOne(t, e, "[file.l]\npath = \"/etc/profile\"\nline = \". /etc/x\"\n", nil)
	if _, err := outside[0].Step.Plan(t.Context()); err == nil || !strings.Contains(err.Error(), "not below /root") {
		t.Errorf("plan = %v, want a path outside root's home refused", err)
	}
}

func TestLineRefusesALinkOnTheWay(t *testing.T) {
	e, m, _, _ := testEnv(false)
	withAccount(m)
	m.Links["/home/u/.config"] = "/etc"
	units := loadOne(t, e, "[file.l]\npath = \"/home/u/.config/rc\"\nowner = \"u\"\nline = \"x\"\n", nil)
	if _, err := units[0].Step.Plan(t.Context()); err == nil || !strings.Contains(err.Error(), "is a link") {
		t.Errorf("plan = %v, want a link on the way refused", err)
	}
	m.Links["/home/u/.bashrc"] = "/etc/hostname"
	if _, err := loadOne(t, e, lineSpec(""), nil)[0].Step.Plan(t.Context()); err == nil || !strings.Contains(err.Error(), "is a link to /etc/hostname") {
		t.Errorf("plan = %v, want a link at the file refused", err)
	}
}

// On the Mac ws reads as the user, so a link at the file that holds the line, as a dotfiles
// manager makes one, is in sync. A link that lacks the line is refused with the remedy. The fake's
// file at the path stands for what the read through the link gives.
func TestLineOnMacOSReadsThroughALinkAtTheFile(t *testing.T) {
	e, m, _, _ := testEnv(false)
	e.System = MacOS
	m.Results["id -u u"] = effecttest.Result{Out: []byte("501\n")}
	m.Results["id -g u"] = effecttest.Result{Out: []byte("20\n")}
	m.Results["dscl . -read /Users/u NFSHomeDirectory"] = effecttest.Result{Out: []byte("NFSHomeDirectory: /Users/u\n")}
	m.Dirs["/Users/u"] = 0o750
	m.Owners["/Users/u"] = [2]int{501, 20}
	m.Links["/Users/u/.zshrc"] = "/Users/u/dotfiles/zshrc"
	m.Files["/Users/u/.zshrc"] = []byte("a\n. /etc/x\n")
	const spec = "[file.l]\npath = \"/Users/u/.zshrc\"\nowner = \"u\"\nline = \". /etc/x\"\n"
	s := loadOne(t, e, spec, nil)[0].Step
	if got := headlines(t, s); got != nil {
		t.Errorf("plan of a link that holds the line = %q, want none", got)
	}
	m.Files["/Users/u/.zshrc"] = []byte("a\n")
	if _, err := s.Plan(t.Context()); err == nil || !strings.Contains(err.Error(), "is a link to /Users/u/dotfiles/zshrc: put '. /etc/x' at the top") {
		t.Errorf("plan of a link that lacks the line = %v, want it refused with the remedy", err)
	}
}

func TestLineHasDecides(t *testing.T) {
	e, m, _, _ := testEnv(false)
	withAccount(m)
	m.Files["/home/u/.bashrc"] = []byte("mine\n")
	units := loadOne(t, e, lineSpec("has = [\"holds\", \"-q\"]\n"), nil)
	m.Results["holds -q"] = effecttest.Result{}
	if got := headlines(t, units[0].Step); got != nil {
		t.Errorf("exit 0: plan = %q, want none", got)
	}
	if !slices.Contains(m.Inputs, "mine\n") {
		t.Errorf("inputs = %q, want the file's text", m.Inputs)
	}
	m.Results["holds -q"] = effecttest.Result{Err: exited(t, exec.Command("sh", "-c", "exit 1"))}
	if got := headlines(t, units[0].Step); len(got) != 2 {
		t.Errorf("exit 1: plan = %q, want the copy and the write", got)
	}
}

func TestLineHasFailingFailsThePlanAndLeavesTheFile(t *testing.T) {
	e, m, _, _ := testEnv(false)
	withAccount(m)
	m.Files["/home/u/.bashrc"] = []byte("mine\n")
	m.Results["holds"] = effecttest.Result{Err: exited(t, exec.Command("sh", "-c", "echo out of place >&2; echo more >&2; exit 2"))}
	err := run(t, e, loadOne(t, e, lineSpec("has = [\"holds\"]\n"), nil))
	if err == nil || !strings.Contains(err.Error(), "holds refuses /home/u/.bashrc, exit status 2: out of place") ||
		strings.Contains(err.Error(), "more") {
		t.Errorf("run = %v, want the first line of the refusal", err)
	}
	if got := string(m.Files["/home/u/.bashrc"]); got != "mine\n" || len(copies(m)) != 0 {
		t.Errorf("file = %q, copies %q; want it left", got, copies(m))
	}
}

func TestLineCheckRefusedLeavesTheFile(t *testing.T) {
	e, m, _, _ := testEnv(false)
	withAccount(m)
	m.Files["/home/u/.bashrc"], m.Modes["/home/u/.bashrc"] = []byte("mine\n"), 0o640
	m.Owners["/home/u/.bashrc"] = [2]int{1001, 1001}
	m.Results["bash -n /home/u/.bashrc.ws-check"] = effecttest.Result{Err: errors.New("exit status 2")}
	err := run(t, e, loadOne(t, e, lineSpec("check = [\"bash\", \"-n\", \"{}\"]\n"), nil))
	if err == nil || !strings.Contains(err.Error(), "bash refuses the new /home/u/.bashrc") {
		t.Errorf("run = %v, want the check's refusal", err)
	}
	p := "/home/u/.bashrc"
	if string(m.Files[p]) != "mine\n" || m.Modes[p] != 0o640 || m.Owners[p] != [2]int{1001, 1001} || len(copies(m)) != 0 {
		t.Errorf("file %q, mode %o, owner %v, copies %q; want it as it was", m.Files[p], m.Modes[p], m.Owners[p], copies(m))
	}
	if _, left := m.Files[p+".ws-check"]; left {
		t.Error("the checked file stayed")
	}
}

func TestLineOfSeveralLinesNeedsHas(t *testing.T) {
	e, _, _, _ := testEnv(false)
	const spec = "[file.l]\npath = \"/home/u/.bashrc\"\nowner = \"u\"\nline = \"a\\nb\\n\"\n"
	if _, err := Load(e, tree(map[string]string{"r/units.toml.tmpl": spec})); err == nil || !strings.Contains(err.Error(), "takes has") {
		t.Errorf("load = %v, want has asked for", err)
	}
	if _, err := Load(e, tree(map[string]string{"r/units.toml.tmpl": spec + "has = [\"holds\"]\n"})); err != nil {
		t.Errorf("load with has = %v", err)
	}
}

func TestLineRefusesTheFieldsItCannotTake(t *testing.T) {
	e, _, _, _ := testEnv(false)
	for extra, want := range map[string]string{
		"template = \"f\"\n":        "a line takes owner, mode, check, has, after and when alone",
		"source = \"f\"\n":          "a line takes owner, mode, check, has, after and when alone",
		"absent = true\n":           "a line takes owner, mode, check, has, after and when alone",
		"secret = true\n":           "a line takes owner, mode, check, has, after and when alone",
		"service = \"a.service\"\n": "a line takes no service",
		"enabled = true\n":          "go with service",
		"reload = true\n":           "go with service",
		"ours = true\n":             "a line takes owner, mode, check, has, after and when alone",
	} {
		_, err := Load(e, tree(map[string]string{"r/units.toml.tmpl": lineSpec(extra), "r/f": ""}))
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("load with %q = %v, want %q", extra, err, want)
		}
	}
	if _, err := Load(e, tree(map[string]string{"r/units.toml.tmpl": "[file.l]\npath = \"/a\"\nsource = \"f\"\nhas = [\"x\"]\n", "r/f": ""})); err == nil ||
		!strings.Contains(err.Error(), "has goes with line") {
		t.Errorf("load of has alone = %v", err)
	}
}
