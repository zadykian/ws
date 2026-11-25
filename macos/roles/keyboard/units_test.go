package keyboard_test

import (
	"bytes"
	"encoding/binary"
	"os"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
	"unicode/utf16"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
	"github.com/zadykian/ws/engine/core/kind"
	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/macos/facts"
)

// bundle is where the layout lives on the Mac, and where the role keeps its copy of the files.
const (
	bundle = "/Library/Keyboard Layouts/KeyboardLayout.bundle/"
	source = "files/KeyboardLayout.bundle/"
)

// layoutFiles are the bundle's six files, by their path below the bundle.
var layoutFiles = []string{
	"Contents/Info.plist",
	"Contents/version.plist",
	"Contents/Resources/ENG.keylayout",
	"Contents/Resources/RUS.keylayout",
	"Contents/Resources/en.lproj/InfoPlist.strings",
	"Contents/Resources/ru.lproj/InfoPlist.strings",
}

// roleFile reads a file of the role, which the test names.
func roleFile(t *testing.T, name string) []byte {
	text, err := os.ReadFile(name) //nolint:gosec // a file of the role, named by the test
	if err != nil {
		t.Fatal(err)
	}
	return text
}

// layoutUnits loads the role's units on a Mac of the user v, with the sources that the spec names.
func layoutUnits(t *testing.T, m *effecttest.Machine) []kind.Unit {
	fsys := fstest.MapFS{}
	for _, name := range append([]string{"units.toml.tmpl"}, sources()...) {
		fsys["keyboard/"+name] = &fstest.MapFile{Data: roleFile(t, name)}
	}
	m.Results["id -u v"] = effecttest.Result{Out: []byte("501\n")}
	m.Results["id -g v"] = effecttest.Result{Out: []byte("20\n")}
	m.Results["dscl . -read /Users/v NFSHomeDirectory"] = effecttest.Result{Out: []byte("NFSHomeDirectory: /Users/v\n")}
	m.Dirs["/Users/v"] = 0o750
	m.Owners["/Users/v"] = [2]int{501, 20}
	f := facts.Mac{User: "v", Home: "/Users/v"}
	units, err := kind.Load(kind.Env{M: m, Out: &effecttest.Out{}, Data: f, System: kind.MacOS, User: "v"}, fsys)
	if err != nil {
		t.Fatal(err)
	}
	return units
}

// sources lists the role's copies of the bundle's files, as the spec names them.
func sources() []string {
	var names []string
	for _, f := range layoutFiles {
		names = append(names, source+f)
	}
	return names
}

// TestTheUnitsAreTheBundlesSixFiles holds the role's units to the bundle's six files. Each is
// written where macOS reads a layout, root's at 0644, byte for byte as the role keeps it.
func TestTheUnitsAreTheBundlesSixFiles(t *testing.T) {
	m := effecttest.NewMachine()
	m.Dirs["/Library"] = 0o755
	units := layoutUnits(t, m)
	want := []string{"file/layout-en-strings", "file/layout-eng", "file/layout-info",
		"file/layout-ru-strings", "file/layout-rus", "file/layout-version"}
	if got := kind.Names(units); !slices.Equal(got, want) {
		t.Fatalf("the units are %v, want %v", got, want)
	}
	if _, err := step.Run(t.Context(), kind.Steps(units), &effecttest.Out{}, false); err != nil {
		t.Fatal(err)
	}
	for _, f := range layoutFiles {
		text := roleFile(t, source+f)
		if got := m.Files[bundle+f]; !bytes.Equal(got, text) {
			t.Errorf("%s holds %d bytes, want the role's %d", f, len(got), len(text))
		}
		if m.Owners[bundle+f] != [2]int{0, 0} || m.Modes[bundle+f] != 0o644 {
			t.Errorf("%s: owner %v, mode %o; want root's at 0644", f, m.Owners[bundle+f], m.Modes[bundle+f])
		}
	}
	for _, u := range units {
		p, err := u.Step.Plan(t.Context())
		if err != nil || len(p.Changes) > 0 {
			t.Errorf("%s: second plan = %v, %v; want none", u.Name, p.Changes, err)
		}
	}
}

// TestTheStringsStayUTF16 keeps the two InfoPlist.strings as the maintainer sent them: UTF-16
// with a byte order mark. macOS reads each file whole, and a conversion would change the bytes.
func TestTheStringsStayUTF16(t *testing.T) {
	want := []byte{0xff, 0xfe}
	for _, unit := range utf16.Encode([]rune("\"ENG\" = \"ENG\";\n\"RUS\" = \"RUS\";\n")) {
		want = binary.LittleEndian.AppendUint16(want, unit)
	}
	for _, lang := range []string{"en", "ru"} {
		text := roleFile(t, source+"Contents/Resources/"+lang+".lproj/InfoPlist.strings")
		if !bytes.Equal(text, want) {
			t.Errorf("%s.lproj/InfoPlist.strings = % x, want % x", lang, text, want)
		}
	}
}

// TestTheBundleSaysItsNameOnce holds the bundle to the name KeyboardLayout, in its paths and in
// the two files that name it, and to no mark of ws's.
func TestTheBundleSaysItsNameOnce(t *testing.T) {
	for _, f := range layoutFiles {
		text := roleFile(t, source+f)
		if bytes.Contains(text, []byte("# Written by")) {
			t.Errorf("%s holds a mark of ws's", f)
		}
		named := strings.HasSuffix(f, ".plist")
		if has := bytes.Contains(text, []byte("<string>KeyboardLayout</string>")); named != has {
			t.Errorf("%s names the bundle: %v, want %v", f, has, named)
		}
	}
}
