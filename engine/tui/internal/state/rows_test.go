package state

import (
	"reflect"
	"testing"

	"github.com/zadykian/ws/engine/core/step"
	"github.com/zadykian/ws/engine/tui/internal/run"
)

// testSections is a server's report: a role of each kind of item, and a step left unread.
func testSections() []run.StateSection {
	return []run.StateSection{
		{Name: "docker", Reports: []step.Report{{
			Step: "Packages",
			Items: []step.Item{
				{Kind: step.PackageItem, Name: "docker-ce", Want: "installed", Found: "29.1"},
				{Kind: step.PackageItem, Name: "podman", Want: "missing", Found: "5.0", Conflict: "held by hand"},
				{Kind: step.ServiceItem, Name: "docker.service", Want: "running", Found: "dead"},
			},
			Changes: []step.Pending{{Kind: step.Restart, Text: "would start docker.service"}},
		}}},
		{Name: "shell", Reports: []step.Report{
			{Step: "Files", Items: []step.Item{
				{Kind: step.FileItem, Name: "/etc/a", Want: "mode 644", Found: "mode 644", Diff: "--- /etc/a\n+++ /etc/a\n@@ -1 +1 @@\n-old\n+new\n"},
				{Kind: step.LinkItem, Name: "/etc/b", Want: "/c", Found: "missing"},
				{Kind: step.DirItem, Name: "/etc/d", Want: "present", Found: "present"},
				{Kind: step.CheckItem, Name: "Files", Want: "passes", Found: "fails", Conflict: "no way"},
			}, Changes: []step.Pending{{Kind: step.Write, Text: "would write /etc/a\n--- /etc/a\n+++ /etc/a\n"}}},
			{Step: "Tools", After: "Files"},
		}},
	}
}

func names(rows []stateRow) []string {
	got := make([]string, 0, len(rows))
	for _, row := range rows {
		got = append(got, row.name)
	}
	return got
}

func TestTabRowsSortsAReportByTab(t *testing.T) {
	want := [tabCount][]string{
		packagesTab:  {"docker-ce", "podman"},
		servicesTab:  {"docker.service"},
		filesTab:     {"/etc/a", "/etc/b", "/etc/d"},
		conflictsTab: {"podman", "Files"},
		pendingTab:   {"would start docker.service", "would write /etc/a", "shell/Tools"},
	}
	for tab, rows := range tabRows(testSections()) {
		if got := names(rows); !reflect.DeepEqual(got, want[tab]) {
			t.Errorf("%s: %v, want %v", tabNames[tab], got, want[tab])
		}
	}
}

func TestTabRowsSayWhatAPartIs(t *testing.T) {
	tabs := tabRows(testSections())
	wantFacts := []stateFact{{"Kind", "package"}, {"Step", "docker/Packages"}, {"Want", "missing"}, {"Found", "5.0"}}
	podman := tabs[packagesTab][1]
	if !reflect.DeepEqual(podman.facts, wantFacts) || podman.conflict != "held by hand" {
		t.Errorf("podman: %+v", podman)
	}
	write := tabs[pendingTab][1]
	if write.diff != "--- /etc/a\n+++ /etc/a\n" || write.facts[0] != (stateFact{"Kind", "write"}) {
		t.Errorf("a file's change: %+v", write)
	}
	if got := tabs[pendingTab][2].facts[0]; got != (stateFact{"Not read", "it waits on Files"}) {
		t.Errorf("a step left unread: %+v", got)
	}
}

func TestTabRowsNameAMacsStepAlone(t *testing.T) {
	sections := []run.StateSection{{Reports: []step.Report{{Step: "Tools", After: "Keys"}}}}
	if got := tabRows(sections)[pendingTab][0].name; got != "Tools" {
		t.Errorf("name %q", got)
	}
}

func TestPrintable(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"words and lines stay", "a b\nc", "a b\nc"},
		{"a tab takes spaces", "a\tb", "a    b"},
		{"an escape goes", "a\x1b[2Jb", "a[2Jb"},
		{"a carriage return goes", "a\r\nb", "a\nb"},
		{"a C1 code goes", "a\u009bb", "ab"},
		{"bytes that are no text show", "a\xffb", "a�b"},
	}
	for _, c := range cases {
		if got := printable(c.in); got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
}
