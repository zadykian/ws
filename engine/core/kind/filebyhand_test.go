package kind

import (
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/resource/file"
)

func TestLineAtEndGoesLastAndKeepsACopy(t *testing.T) {
	e, m, _, _ := testEnv()
	withAccount(m)
	m.Files["/home/u/.bashrc"], m.Owners["/home/u/.bashrc"] = []byte("mine"), [2]int{1001, 1001}
	units := loadOne(t, e, lineSpec("line_at = \"end\"\n"), nil)
	if err := run(t, e, units); err != nil {
		t.Fatal(err)
	}
	if got := string(m.Files["/home/u/.bashrc"]); got != "mine\n. /etc/x\n" {
		t.Errorf("file = %q, want the line last, after a newline the file lacked", got)
	}
	if kept := copies(m); len(kept) != 1 || string(m.Files[kept[0]]) != "mine" {
		t.Errorf("copies = %q, want the old file once", kept)
	}
	if got := headlines(t, units[0].Step); got != nil {
		t.Errorf("second plan = %q, want none", got)
	}
}

func TestLineAtEndRefusesALink(t *testing.T) {
	e, m, _, _ := testEnv()
	withAccount(m)
	m.Links["/home/u/.bashrc"] = "/home/u/dots/bashrc"
	m.Files["/home/u/dots/bashrc"] = []byte("mine\n")
	_, err := loadOne(t, e, lineSpec("line_at = \"end\"\n"), nil)[0].Step.Plan(t.Context())
	if err == nil || !strings.Contains(err.Error(), "is a link to /home/u/dots/bashrc") {
		t.Errorf("plan = %v, want the link refused", err)
	}
	if got := string(m.Files["/home/u/dots/bashrc"]); got != "mine\n" {
		t.Errorf("the link's file = %q, want it left", got)
	}
}

func TestLineAtRefusesAnotherPlace(t *testing.T) {
	e, _, _, _ := testEnv()
	for spec, want := range map[string]string{
		lineSpec("line_at = \"middle\"\n"):                             "line_at \"middle\" is top or end",
		"[file.a]\npath = \"/a\"\nsource = \"f\"\nline_at = \"end\"\n": "line_at goes with line",
	} {
		_, err := Load(e, tree(map[string]string{"r/units.toml.tmpl": spec, "r/f": ""}))
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("load of\n%s= %v, want %q", spec, err, want)
		}
	}
	if _, err := Load(e, tree(map[string]string{"r/units.toml.tmpl": lineSpec("line_at = \"top\"\n")})); err != nil {
		t.Errorf("load with line_at top = %v", err)
	}
}

// zedSpec is a file of u's that ws writes whole, ours by a JSON comment's mark, with the extra
// fields.
func zedSpec(extra string) string {
	return "[file.z]\npath = \"/home/u/.config/zed/keymap.json\"\nowner = \"u\"\nsource = \"k\"\nours = true\nmark = \"//\"\n" + extra
}

// zedKeymap is the file's source, which starts with ws's mark after a JSON comment's sign.
const zedKeymap = "// " + file.Mark + ", whose next run replaces any change made here.\n[]\n"

func TestOursOnAnAccountFile(t *testing.T) {
	e, m, _, _ := testEnv()
	withAccount(m)
	units := loadOne(t, e, zedSpec(""), map[string]string{"k": zedKeymap})
	if err := run(t, e, units); err != nil {
		t.Fatal(err)
	}
	p := "/home/u/.config/zed/keymap.json"
	if string(m.Files[p]) != zedKeymap || m.Owners[p] != [2]int{1001, 1001} || m.Owners["/home/u/.config/zed"] != [2]int{1001, 1001} {
		t.Errorf("file %q, owner %v, its directory's %v; want ws's, u's", m.Files[p], m.Owners[p], m.Owners["/home/u/.config/zed"])
	}
	m.Files[p] = []byte("// " + file.Mark + ", from an older run.\n[1]\n")
	if got := headlines(t, units[0].Step); !slices.Equal(got, []string{"write " + p}) {
		t.Errorf("plan = %q, want ws's own file replaced", got)
	}
	m.Links["/home/u/.config/zed"] = "/etc"
	if _, err := units[0].Step.Plan(t.Context()); err == nil || !strings.Contains(err.Error(), "is a link to /etc") {
		t.Errorf("plan = %v, want a link on the way refused before the read", err)
	}
}

func TestOursMarkTakesItsSign(t *testing.T) {
	e, m, _, _ := testEnv()
	withAccount(m)
	m.Files["/home/u/.config/zed/keymap.json"] = []byte("# " + file.Mark + "\n[]\n")
	_, err := loadOne(t, e, zedSpec(""), map[string]string{"k": zedKeymap})[0].Step.Plan(t.Context())
	if err == nil || !strings.Contains(err.Error(), "lacks the line '// "+file.Mark+"' at its top") {
		t.Errorf("plan = %v, want a mark after another sign refused", err)
	}
	for spec, want := range map[string]string{
		"[file.a]\npath = \"/a\"\nsource = \"k\"\nmark = \"//\"\n":                    "mark and keep_foreign go with ours",
		"[file.a]\npath = \"/a\"\nsource = \"k\"\nkeep_foreign = true\n":              "mark and keep_foreign go with ours",
		"[file.a]\npath = \"/a\"\nsource = \"k\"\nours = true\nmark = \"// x\"\n":     "is a comment sign, with no white space",
		"[file.a]\npath = \"/home/u/.x\"\nowner = \"u\"\nline = \"x\"\nours = true\n": "a line takes owner",
	} {
		_, err := Load(e, tree(map[string]string{"r/units.toml.tmpl": spec, "r/k": ""}))
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("load of\n%s= %v, want %q", spec, err, want)
		}
	}
}

func TestKeepForeignSkipsWithAWarning(t *testing.T) {
	e, m, _, out := testEnv()
	withAccount(m)
	p := "/home/u/.config/zed/keymap.json"
	m.Files[p] = []byte("[]\n")
	units := loadOne(t, e, zedSpec("keep_foreign = true\n"), map[string]string{"k": zedKeymap})
	if got := headlines(t, units[0].Step); got != nil {
		t.Errorf("plan = %q, want none", got)
	}
	if want := []string{"warn: " + p + " lacks the line '// " + file.Mark + "' at its top, so it is left as it is: " +
		"remove it, and the next run writes ws's"}; !slices.Equal(out.Lines, want) {
		t.Errorf("lines = %q, want %q", out.Lines, want)
	}
	if err := run(t, e, units); err != nil || string(m.Files[p]) != "[]\n" {
		t.Errorf("run = %v, file %q; want the user's file left", err, m.Files[p])
	}
	delete(m.Files, p)
	if err := run(t, e, units); err != nil || string(m.Files[p]) != zedKeymap {
		t.Errorf("run = %v, file %q; want ws's written once the user's went", err, m.Files[p])
	}
}

func TestForeignFailsWithoutKeep(t *testing.T) {
	e, m, _, _ := testEnv()
	withAccount(m)
	p := "/home/u/.config/zed/keymap.json"
	m.Files[p] = []byte("[\"mine\"]\n")
	err := run(t, e, loadOne(t, e, zedSpec(""), map[string]string{"k": zedKeymap}))
	if err == nil || !strings.Contains(err.Error(), p+" lacks the line") || strings.Contains(err.Error(), "mine") {
		t.Errorf("run = %v, want the file by hand refused by its path alone", err)
	}
	if string(m.Files[p]) != "[\"mine\"]\n" {
		t.Errorf("file = %q, want it left", m.Files[p])
	}
}

// seedSpec is Zed's settings.json of u's, which ws writes where missing alone.
const seedSpec = "[file.s]\npath = \"/home/u/.config/zed/settings.json\"\nowner = \"u\"\nsource = \"s\"\nseed = true\n"

func TestSeedWritesWhereMissing(t *testing.T) {
	e, m, _, _ := testEnv()
	withAccount(m)
	units := loadOne(t, e, seedSpec, map[string]string{"s": "{}\n"})
	if err := run(t, e, units); err != nil {
		t.Fatal(err)
	}
	p := "/home/u/.config/zed/settings.json"
	if string(m.Files[p]) != "{}\n" || m.Owners[p] != [2]int{1001, 1001} {
		t.Errorf("file %q, owner %v; want the seed, u's", m.Files[p], m.Owners[p])
	}
	// The home's missing directories are made on the way, the account's, at 0755.
	for _, d := range []string{"/home/u/.config", "/home/u/.config/zed"} {
		if m.Dirs[d] != 0o755 || m.Owners[d] != [2]int{1001, 1001} {
			t.Errorf("%s: mode %o, owner %v; want 755, u's", d, m.Dirs[d], m.Owners[d])
		}
	}
	if got := headlines(t, units[0].Step); got != nil {
		t.Errorf("second plan = %q, want none", got)
	}
}

func TestSeedLeavesAnEditedFile(t *testing.T) {
	e, m, _, _ := testEnv()
	withAccount(m)
	p := "/home/u/.config/zed/settings.json"
	m.Files[p], m.Modes[p] = []byte("{ \"theme\": \"mine\" }\n"), 0o600
	units := loadOne(t, e, seedSpec, map[string]string{"s": "{}\n"})
	if err := run(t, e, units); err != nil || string(m.Files[p]) != "{ \"theme\": \"mine\" }\n" || m.Modes[p] != 0o600 {
		t.Errorf("run = %v, file %q, mode %o; want it left as it is", err, m.Files[p], m.Modes[p])
	}
	delete(m.Files, p)
	m.Links[p] = "/home/u/dots/zed.json"
	if got := headlines(t, units[0].Step); got != nil {
		t.Errorf("plan over a link = %q, want none", got)
	}
	for _, extra := range []string{"backup = true\n", "ours = true\n", "absent_when_empty = true\n", "check = [\"x\"]\n"} {
		_, err := Load(e, tree(map[string]string{"r/units.toml.tmpl": seedSpec + extra, "r/s": ""}))
		if err == nil || !strings.Contains(err.Error(), "seed takes a file that ws writes once") {
			t.Errorf("load with %q = %v, want seed refused", extra, err)
		}
	}
}
