package kind

import (
	"strings"
	"testing"
)

func TestLineAtEndGoesLastAndKeepsACopy(t *testing.T) {
	e, m, _, _ := testEnv(false)
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
	e, m, _, _ := testEnv(false)
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
	e, _, _, _ := testEnv(false)
	for spec, want := range map[string]string{
		lineSpec("line_at = \"middle\"\n"):                             "line_at \"middle\" is top or end",
		"[file.a]\npath = \"/a\"\nsource = \"f\"\nline_at = \"end\"\n": "line_at goes with line",
	} {
		_, err := Load(e, tree(map[string]string{"r/units.toml": spec, "r/f": ""}))
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("load of\n%s= %v, want %q", spec, err, want)
		}
	}
	if _, err := Load(e, tree(map[string]string{"r/units.toml": lineSpec("line_at = \"top\"\n")})); err != nil {
		t.Errorf("load with line_at top = %v", err)
	}
}
