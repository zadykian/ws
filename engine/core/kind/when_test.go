package kind

import (
	"slices"
	"strings"
	"testing"

	"github.com/zadykian/ws/engine/core/effect/effecttest"
)

// acme is the board's maker that the tests' specs want, and 0x1234 their GPU's vendor.
const acme = "Acme Ltd."

// withBoard gives the machine a board of the maker.
func withBoard(m *effecttest.Machine, maker string) {
	m.Files[boardVendorFile] = []byte(maker + "\n")
}

// withPCIDevice gives the machine one PCI device of the class and the vendor.
func withPCIDevice(m *effecttest.Machine, class, vendor string) {
	dev := pciDevices + "/0000:01:00.0"
	m.Dirs[pciDevices], m.Dirs[dev] = 0o755, 0o755
	m.Files[dev+"/class"], m.Files[dev+"/vendor"] = []byte(class+"\n"), []byte(vendor+"\n")
}

// gatedFile is the spec of file/a at /etc/a, with the when that cond gives.
func gatedFile(cond string) string {
	return "[file.a]\npath = \"/etc/a\"\nsource = \"a\"\nwhen = " + cond + "\n"
}

// planGated loads gatedFile(cond) on m and returns its plan's headlines and what the run said.
func planGated(t *testing.T, m *effecttest.Machine, cond string) ([]string, []string) {
	out := &effecttest.Out{}
	units := loadOne(t, Env{M: m, U: &effecttest.Units{}, Out: out}, gatedFile(cond), map[string]string{"a": "x"})
	return headlines(t, units[0].Step), out.Lines
}

func TestWhenBoardVendor(t *testing.T) {
	cond := `{ board_vendor = "` + acme + `" }`
	for _, tc := range []struct {
		name  string
		maker string
		cond  string
		want  bool
	}{
		{"the board", acme, cond, true},
		{"QEMU's board", "QEMU", cond, false},
		{"no DMI file", "", cond, false},
		{"an empty want", acme, `{ board_vendor = "" }`, false},
	} {
		m := effecttest.NewMachine()
		if tc.maker != "" {
			withBoard(m, tc.maker)
		}
		if got, _ := planGated(t, m, tc.cond); (got != nil) != tc.want {
			t.Errorf("%s: plan = %q, want on %v", tc.name, got, tc.want)
		}
	}
}

func TestWhenGPUVendor(t *testing.T) {
	for _, tc := range []struct {
		name          string
		class, vendor string
		want          bool
	}{
		{"a display device", "0x030000", "0x1234", true},
		{"a 3D controller", "0x030200", "0x1234", true},
		{"an audio device", "0x040300", "0x1234", false},
		{"another vendor's display", "0x030000", "0x1af4", false},
		{"no PCI devices", "", "", false},
	} {
		m := effecttest.NewMachine()
		if tc.class != "" {
			withPCIDevice(m, tc.class, tc.vendor)
		}
		if got, _ := planGated(t, m, `{ gpu_vendor = "0x1234" }`); (got != nil) != tc.want {
			t.Errorf("%s: plan = %q, want on %v", tc.name, got, tc.want)
		}
	}
	m := effecttest.NewMachine()
	withPCIDevice(m, "0x030000", "0x1234")
	delete(m.Files, pciDevices+"/0000:01:00.0/class")
	units := loadOne(t, Env{M: m, Out: &effecttest.Out{}}, gatedFile(`{ gpu_vendor = "0x1234" }`), map[string]string{"a": "x"})
	if _, err := units[0].Step.Plan(t.Context()); err == nil || !strings.Contains(err.Error(), "when gpu_vendor") {
		t.Errorf("plan with a device without class = %v, want when's error", err)
	}
}

func TestWhenOffPlansNothing(t *testing.T) {
	m := effecttest.NewMachine()
	withBoard(m, "QEMU")
	got, said := planGated(t, m, `{ board_vendor = "`+acme+`" }`)
	if got != nil || slices.Contains(m.Calls, "write /etc/a") {
		t.Errorf("plan = %q, calls %q; want none", got, m.Calls)
	}
	want := `say: file/a: off on this machine, as its board_vendor is not "` + acme + `"`
	if !slices.Equal(said, []string{want}) {
		t.Errorf("said %q, want %q", said, want)
	}
}

func TestWhenOnPlansAsWithout(t *testing.T) {
	m := effecttest.NewMachine()
	withBoard(m, acme)
	withPCIDevice(m, "0x030000", "0x1234")
	got, said := planGated(t, m, `{ board_vendor = "`+acme+`", gpu_vendor = "0x1234" }`)
	if !slices.Contains(got, "create /etc/a") || said != nil {
		t.Errorf("plan = %q, said %q; want the write of /etc/a, and no line", got, said)
	}
}

func TestWhenIsReadAtPlan(t *testing.T) {
	m := effecttest.NewMachine()
	units := loadOne(t, Env{M: m, Out: &effecttest.Out{}}, gatedFile(`{ board_vendor = "`+acme+`" }`), map[string]string{"a": "x"})
	if got := headlines(t, units[0].Step); got != nil {
		t.Fatalf("plan without the board = %q, want none", got)
	}
	withBoard(m, acme)
	if got := headlines(t, units[0].Step); !slices.Contains(got, "create /etc/a") {
		t.Errorf("plan once the board is there = %q, want the write", got)
	}
}

func TestWhenRefusesAnUnknownFact(t *testing.T) {
	e, _, _, _ := testEnv(false)
	for cond, want := range map[string]string{
		`{ gpu = "x" }`:        `when: no fact "gpu"; the facts are board_vendor, gpu_vendor`,
		`{ board_vendor = 1 }`: "file.a.when (line 4",
	} {
		_, err := Load(e, tree(map[string]string{"r/units.toml": gatedFile(cond), "r/a": "x"}))
		if err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("load with when = %s: %v, want %q", cond, err, want)
		}
	}
}

func TestWhenOnEveryKind(t *testing.T) {
	const off = "when = { board_vendor = \"" + acme + "\" }\n"
	for name, spec := range map[string]string{
		"guard":   "[guard.a]\nsudo_right = \"u\"\n" + off,
		"state":   "[state.a]\npurged = \"p\"\n" + off,
		"service": "[service.a]\nunit = \"a.service\"\nenabled = true\nrun = \"running\"\n" + off,
	} {
		e, m, _, out := testEnv(true)
		units := loadOne(t, e, spec, nil)
		if got := headlines(t, units[0].Step); got != nil || len(out.Lines) != 1 {
			t.Errorf("%s: plan = %q, said %q; want none and the off line", name, got, out.Lines)
		}
		if slices.ContainsFunc(m.Calls, func(c string) bool { return !strings.HasPrefix(c, "read ") }) {
			t.Errorf("%s: calls %q, want the board's read alone", name, m.Calls)
		}
	}
}
